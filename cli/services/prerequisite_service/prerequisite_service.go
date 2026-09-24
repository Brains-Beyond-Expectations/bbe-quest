package prerequisite_service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"gopkg.in/yaml.v3"
)

var execCommand = exec.Command
var httpClient = &http.Client{Timeout: 2 * time.Minute}
var factoryUrl = "https://factory.talos.dev"

// Image Factory adds this extension to every image it builds, with the image's schematic ID as its version
const schematicExtension = "schematic"

// How long bbe waits for a node to come back after upgrading it
const upgradeTimeout = "30m"

type PrerequisiteService struct{}

type node struct {
	name         string
	ip           string
	controlPlane bool
}

// What a node lacks for a Talos prerequisite
type nodeGaps struct {
	extensions  []string
	directories []string
	config      nodeConfig
}

// What's set up in a node's machine config
type nodeConfig struct {
	// Configs made for Talos 1.14 and later configure the kubelet in its own document, which can't add mounts
	kubeletDocument bool
	userVolumes     []string
	kubeletMounts   []string
}

// Returns what's missing for the prerequisite, which is nothing when it's met
func (prerequisiteService PrerequisiteService) Check(prerequisite models.ChartPrerequisite, cluster models.ClusterAccess) ([]string, error) {
	var missing []string

	if prerequisite.StorageClass != "" {
		exists, err := storageClassExists(cluster, prerequisite.StorageClass)
		if err != nil {
			return nil, err
		}
		if !exists {
			missing = append(missing, fmt.Sprintf("the `%s` storage class", prerequisite.StorageClass))
		}
	}

	if prerequisite.Talos != nil {
		nodes, err := listNodes(cluster)
		if err != nil {
			return nil, err
		}

		for _, node := range nodes {
			gaps, err := findNodeGaps(node, *prerequisite.Talos, cluster)
			if err != nil {
				return nil, err
			}
			for _, extension := range gaps.extensions {
				missing = append(missing, fmt.Sprintf("the %s extension on node `%s`", extension, node.name))
			}
			for _, directory := range gaps.directories {
				missing = append(missing, fmt.Sprintf("the %s directory on node `%s`", directory, node.name))
			}
		}
	}

	return missing, nil
}

// Adds the missing directories to each node's config, and upgrades each node missing an extension to an
// image with it. Workers go first, so the control plane restarts last.
func (prerequisiteService PrerequisiteService) PrepareNodes(prerequisite models.TalosPrerequisite, cluster models.ClusterAccess) error {
	nodes, err := listNodes(cluster)
	if err != nil {
		return err
	}
	slices.SortStableFunc(nodes, func(a node, b node) int {
		switch {
		case a.controlPlane == b.controlPlane:
			return 0
		case b.controlPlane:
			return -1
		default:
			return 1
		}
	})

	for _, node := range nodes {
		gaps, err := findNodeGaps(node, prerequisite, cluster)
		if err != nil {
			return err
		}

		if len(gaps.directories) > 0 {
			if err := addDirectories(node, gaps.directories, gaps.config, cluster); err != nil {
				return err
			}
			logger.Info(fmt.Sprintf("Node `%s`: added %s", node.name, strings.Join(gaps.directories, ", ")))
		}

		if len(gaps.extensions) > 0 {
			image, err := upgradeImage(node, gaps.extensions, cluster)
			if err != nil {
				return err
			}

			logger.Info(fmt.Sprintf("Node `%s`: upgrading to add %s, it restarts and can take several minutes...", node.name, strings.Join(gaps.extensions, ", ")))
			if err := upgradeNode(node, image, cluster); err != nil {
				return err
			}
			logger.Info(fmt.Sprintf("Node `%s`: upgraded", node.name))
		}
	}

	return nil
}

func storageClassExists(cluster models.ClusterAccess, name string) (bool, error) {
	response, err := kubectl(cluster, "get", "storageclass", name, "-o", "name")
	if err == nil {
		return true, nil
	}
	if strings.Contains(string(response), "NotFound") {
		return false, nil
	}

	return false, fmt.Errorf("Failed to check for the `%s` storage class: %w", name, err)
}

func listNodes(cluster models.ClusterAccess) ([]node, error) {
	response, err := kubectl(cluster, "get", "nodes", "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("Failed to list the cluster's nodes: %w", err)
	}

	var list struct {
		Items []struct {
			Metadata struct {
				Name   string            `json:"name"`
				Labels map[string]string `json:"labels"`
			} `json:"metadata"`
			Status struct {
				Addresses []struct {
					Type    string `json:"type"`
					Address string `json:"address"`
				} `json:"addresses"`
			} `json:"status"`
		} `json:"items"`
	}
	if err := json.Unmarshal(response, &list); err != nil {
		return nil, fmt.Errorf("Failed to read the cluster's nodes: %w", err)
	}

	var nodes []node
	for _, item := range list.Items {
		address := ""
		for _, candidate := range item.Status.Addresses {
			if candidate.Type == "InternalIP" {
				address = candidate.Address
				break
			}
		}
		if address == "" {
			return nil, fmt.Errorf("Node `%s` has no internal IP address", item.Metadata.Name)
		}

		_, controlPlane := item.Metadata.Labels["node-role.kubernetes.io/control-plane"]
		nodes = append(nodes, node{name: item.Metadata.Name, ip: address, controlPlane: controlPlane})
	}

	return nodes, nil
}

func findNodeGaps(node node, prerequisite models.TalosPrerequisite, cluster models.ClusterAccess) (nodeGaps, error) {
	var gaps nodeGaps

	if len(prerequisite.Extensions) > 0 {
		installed, err := nodeExtensions(node, cluster)
		if err != nil {
			return gaps, err
		}
		for _, extension := range prerequisite.Extensions {
			// Nodes report `siderolabs/iscsi-tools` as `iscsi-tools`
			if _, found := installed[path.Base(extension)]; !found {
				gaps.extensions = append(gaps.extensions, extension)
			}
		}
	}

	if len(prerequisite.Directories) > 0 {
		config, err := readNodeConfig(node, cluster)
		if err != nil {
			return gaps, err
		}
		gaps.config = config
		for _, directory := range prerequisite.Directories {
			if !config.hasDirectory(directory) {
				gaps.directories = append(gaps.directories, directory)
			}
		}
	}

	return gaps, nil
}

// Returns the node's extensions by name, with their versions
func nodeExtensions(node node, cluster models.ClusterAccess) (map[string]string, error) {
	response, err := talosctl(node, cluster, "get", "extensions", "-o", "yaml")
	if err != nil {
		return nil, fmt.Errorf("Failed to list the extensions on node `%s`: %w", node.name, err)
	}

	extensions := map[string]string{}
	decoder := yaml.NewDecoder(bytes.NewReader(response))
	for {
		var resource struct {
			Spec struct {
				Metadata struct {
					Name    string `yaml:"name"`
					Version string `yaml:"version"`
				} `yaml:"metadata"`
			} `yaml:"spec"`
		}
		err := decoder.Decode(&resource)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Failed to read the extensions on node `%s`: %w", node.name, err)
		}
		if resource.Spec.Metadata.Name != "" {
			extensions[resource.Spec.Metadata.Name] = resource.Spec.Metadata.Version
		}
	}

	return extensions, nil
}

func readNodeConfig(node node, cluster models.ClusterAccess) (nodeConfig, error) {
	var config nodeConfig

	response, err := talosctl(node, cluster, "get", "machineconfig", "-o", "yaml")
	if err != nil {
		return config, fmt.Errorf("Failed to read the machine config of node `%s`: %w", node.name, err)
	}

	// The resource holds the whole config as a string, with one YAML document per part
	var resource struct {
		Spec string `yaml:"spec"`
	}
	if err := yaml.Unmarshal(response, &resource); err != nil {
		return config, fmt.Errorf("Failed to read the machine config of node `%s`: %w", node.name, err)
	}

	decoder := yaml.NewDecoder(strings.NewReader(resource.Spec))
	for {
		var document struct {
			Kind    string `yaml:"kind"`
			Name    string `yaml:"name"`
			Machine struct {
				Kubelet struct {
					ExtraMounts []struct {
						Destination string `yaml:"destination"`
					} `yaml:"extraMounts"`
				} `yaml:"kubelet"`
			} `yaml:"machine"`
		}
		err := decoder.Decode(&document)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return config, fmt.Errorf("Failed to read the machine config of node `%s`: %w", node.name, err)
		}

		switch document.Kind {
		case "KubeletConfig":
			config.kubeletDocument = true
		case "UserVolumeConfig":
			config.userVolumes = append(config.userVolumes, document.Name)
		case "":
			for _, mount := range document.Machine.Kubelet.ExtraMounts {
				config.kubeletMounts = append(config.kubeletMounts, mount.Destination)
			}
		}
	}

	return config, nil
}

func (config nodeConfig) hasDirectory(directory string) bool {
	if config.kubeletDocument {
		return slices.Contains(config.userVolumes, path.Base(directory))
	}

	return slices.Contains(config.kubeletMounts, directory)
}

// Talos 1.14 configs get a directory user volume, which Talos creates under /var/mnt and pods can use. Older
// configs mount the directory into the kubelet instead, which Talos creates, with rshared so mounts made
// inside a pod, like Longhorn's volumes, reach the kubelet.
func addDirectories(node node, directories []string, config nodeConfig, cluster models.ClusterAccess) error {
	var patches []interface{}
	if config.kubeletDocument {
		for _, directory := range directories {
			patches = append(patches, map[string]interface{}{
				"apiVersion": "v1alpha1",
				"kind":       "UserVolumeConfig",
				"name":       path.Base(directory),
				"volumeType": "directory",
			})
		}
	} else {
		var extraMounts []map[string]interface{}
		for _, directory := range directories {
			extraMounts = append(extraMounts, map[string]interface{}{
				"destination": directory,
				"type":        "bind",
				"source":      directory,
				"options":     []string{"bind", "rshared", "rw"},
			})
		}
		patches = append(patches, map[string]interface{}{
			"machine": map[string]interface{}{"kubelet": map[string]interface{}{"extraMounts": extraMounts}},
		})
	}

	args := []string{"patch", "machineconfig"}
	for _, patch := range patches {
		content, err := json.Marshal(patch)
		if err != nil {
			return err
		}
		args = append(args, "--patch", string(content))
	}

	if _, err := talosctl(node, cluster, args...); err != nil {
		return fmt.Errorf("Failed to add %s on node `%s`: %w", strings.Join(directories, ", "), node.name, err)
	}

	return nil
}

// Builds the node's own image again with the missing extensions added, on the Talos version it already runs
func upgradeImage(node node, extensions []string, cluster models.ClusterAccess) (string, error) {
	installed, err := nodeExtensions(node, cluster)
	if err != nil {
		return "", err
	}
	schematicID, found := installed[schematicExtension]
	if !found {
		return "", fmt.Errorf("Node `%s` wasn't installed from an Image Factory image, so bbe can't add extensions to it", node.name)
	}

	version, err := talosVersion(node, cluster)
	if err != nil {
		return "", err
	}

	newSchematicID, err := addExtensionsToSchematic(schematicID, extensions)
	if err != nil {
		return "", fmt.Errorf("Failed to build an image with %s for node `%s`: %w", strings.Join(extensions, ", "), node.name, err)
	}

	host := strings.TrimPrefix(strings.TrimPrefix(factoryUrl, "https://"), "http://")
	return fmt.Sprintf("%s/metal-installer/%s:%s", host, newSchematicID, version), nil
}

func talosVersion(node node, cluster models.ClusterAccess) (string, error) {
	response, err := talosctl(node, cluster, "version")
	if err != nil {
		return "", fmt.Errorf("Failed to read the Talos version of node `%s`: %w", node.name, err)
	}

	return parseServerVersion(string(response))
}

var tagPattern = regexp.MustCompile(`(?m)^\s*Tag:\s*(\S+)`)

// `talosctl version` lists the client's version first, so only the server's part counts
func parseServerVersion(output string) (string, error) {
	_, server, found := strings.Cut(output, "Server:")
	if match := tagPattern.FindStringSubmatch(server); found && match != nil {
		return match[1], nil
	}

	return "", errors.New("Failed to find the node's Talos version")
}

func addExtensionsToSchematic(schematicID string, extensions []string) (string, error) {
	response, err := httpClient.Get(fmt.Sprintf("%s/schematics/%s", factoryUrl, schematicID))
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("schematic %s returned %d: %s", schematicID, response.StatusCode, strings.TrimSpace(string(body)))
	}

	// Keep everything else in the schematic, such as a Raspberry Pi overlay, as it is
	var schematic map[string]interface{}
	if err := yaml.Unmarshal(body, &schematic); err != nil {
		return "", err
	}
	customization := childMap(schematic, "customization")
	systemExtensions := childMap(customization, "systemExtensions")
	official, _ := systemExtensions["officialExtensions"].([]interface{})
	for _, extension := range extensions {
		if !slices.Contains(official, interface{}(extension)) {
			official = append(official, extension)
		}
	}
	systemExtensions["officialExtensions"] = official

	content, err := yaml.Marshal(schematic)
	if err != nil {
		return "", err
	}

	created, err := httpClient.Post(fmt.Sprintf("%s/schematics", factoryUrl), "application/yaml", bytes.NewReader(content))
	if err != nil {
		return "", err
	}
	defer created.Body.Close()
	body, err = io.ReadAll(created.Body)
	if err != nil {
		return "", err
	}
	if created.StatusCode != http.StatusOK && created.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("creating the schematic returned %d: %s", created.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.ID == "" {
		return "", fmt.Errorf("unexpected answer when creating the schematic: %s", strings.TrimSpace(string(body)))
	}

	return result.ID, nil
}

func childMap(parent map[string]interface{}, key string) map[string]interface{} {
	child, isMap := parent[key].(map[string]interface{})
	if !isMap {
		child = map[string]interface{}{}
		parent[key] = child
	}

	return child
}

func upgradeNode(node node, image string, cluster models.ClusterAccess) error {
	if _, err := talosctl(node, cluster, "upgrade", "--image", image, "--wait", "--timeout", upgradeTimeout); err != nil {
		return fmt.Errorf("Failed to upgrade node `%s` to %s: %w", node.name, image, err)
	}

	return nil
}

func kubectl(cluster models.ClusterAccess, args ...string) ([]byte, error) {
	cmd := execCommand("kubectl", append(args, "--context", cluster.KubeContext)...)
	logger.Debug(fmt.Sprintf("Command: %s", cmd.String()))

	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))
	if errors.Is(err, exec.ErrNotFound) {
		return response, errors.New("kubectl is needed to check this package's prerequisites, please install it")
	}
	if err != nil {
		return response, fmt.Errorf("%w%s", err, output(response))
	}

	return response, nil
}

func talosctl(node node, cluster models.ClusterAccess, args ...string) ([]byte, error) {
	cmd := execCommand("talosctl", append(args, "--nodes", node.ip, "--endpoints", cluster.ControlPlaneIp, "--talosconfig", cluster.TalosConfig)...)
	logger.Debug(fmt.Sprintf("Command: %s", cmd.String()))

	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))
	if err != nil {
		return response, fmt.Errorf("%w%s", err, output(response))
	}

	return response, nil
}

func output(response []byte) string {
	if trimmed := strings.TrimSpace(string(response)); trimmed != "" {
		return "\n" + trimmed
	}

	return ""
}
