package talos_service

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"gopkg.in/yaml.v2"
)

var execCommand = exec.Command
var tenSeconds = 10 * time.Second
var fiveMinutes = 5 * time.Minute

var osReadFile = os.ReadFile
var osWriteFile = os.WriteFile

var talosVersionRegex = regexp.MustCompile(`Talos\s+(v[0-9]+\.[0-9]+\.[0-9]+)`)

type TalosService struct{}

func (talosService TalosService) Ping(nodeIp string) bool {
	// First check that this is a Talos device by querying for disks
	cmd := execCommand("talosctl", "-n", nodeIp, "get", "disks", "--insecure")
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return false
	}

	// If it is check if we get turned away by the machineconfig (if so it is likely to be in maintenance mode)
	cmd = execCommand("talosctl", "-n", nodeIp, "get", "machineconfig", "--insecure")
	output, err = cmd.CombinedOutput()
	logger.Debug(string(output))

	return err != nil
}

/**
 * WarnIfTalosVersionMismatch compares the locally installed talosctl client against
 * expectedVersion (the Talos version the node's image was built for) and logs a
 * warning on a mismatch. It does not query the node directly: nodes are still in
 * maintenance mode at the point this runs, and maintenance mode does not implement
 * the version RPC.
 */
func (talosService TalosService) WarnIfTalosVersionMismatch(expectedVersion string) {
	localVersion, err := talosService.GetLocalVersion()
	if err != nil {
		logger.Warning(fmt.Sprintf("Failed to get local Talos version: %v", err))
		return
	}

	if localVersion != expectedVersion {
		logger.Warning(fmt.Sprintf("Talos version mismatch: local talosctl is %s but the node image is %s. The generated config may contain fields the node doesn't recognize.", localVersion, expectedVersion))
	}
}

/**
 * GetLocalVersion returns the version of the locally installed talosctl client.
 * It executes "talosctl version --client --short" and parses the output.
 * Returns the version string (e.g., "v1.14.1") or an error if parsing fails.
 */
func (talosService TalosService) GetLocalVersion() (string, error) {
	cmd := execCommand("talosctl", "version", "--client", "--short")
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return "", err
	}

	return parseTalosVersion(string(output))
}

/**
 * parseTalosVersion extracts the Talos version from the command output.
 * The expected format is "Talos vX.Y.Z".
 * Returns the version string (e.g., "v1.14.1") or an error if parsing fails.
 */
func parseTalosVersion(output string) (string, error) {
	matches := talosVersionRegex.FindStringSubmatch(output)
	if matches == nil {
		return "", fmt.Errorf("could not parse Talos version from output: %s", strings.TrimSpace(output))
	}

	return matches[1], nil
}

// GenerateConfig generates a fresh cluster config, pinning --talos-version so the config
// schema matches the Talos version the node image was built for rather than whatever
// talosctl happens to be installed locally.
func (talosService TalosService) GenerateConfig(helperService interfaces.HelperServiceInterface, controlPlaneIp string, clusterName string, talosVersion string) error {
	cmd := execCommand("talosctl", "gen", "config", clusterName, fmt.Sprintf("https://%s:6443", controlPlaneIp), "--talos-version", talosVersion, "--output", helperService.GetConfigDir())
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		if strings.Contains(string(output), "already exists") {
			return constants.ConfigExistsError
		}
		return err
	}

	return nil
}

func (talosService TalosService) JoinCluster(helperService interfaces.HelperServiceInterface, nodeIp string, nodeConfigFile string) error {
	logger.Infof("Instance %s is joining the cluster", nodeIp)

	cmd := execCommand("talosctl", "apply-config", "--insecure", "-n", nodeIp, "--file", helperService.GetConfigFilePath(nodeConfigFile))
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return err
	}

	return nil
}

func (talosService TalosService) BootstrapCluster(helperService interfaces.HelperServiceInterface, nodeIp string, controlPlaneIp string) error {
	logger.Info("Bootstrapping cluster, this might take a few minutes...")

	configFilePath := helperService.GetConfigFilePath(constants.TalosConfigFile)

	start := time.Now()
	timeout := fiveMinutes
	for {
		cmd := execCommand("talosctl", "bootstrap", "--nodes", nodeIp, "--endpoints", controlPlaneIp, fmt.Sprintf("--talosconfig=%s", configFilePath))
		output, err := cmd.CombinedOutput()
		logger.Debug(string(output))

		if err == nil {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("Bootstrap failed after 5 minutes: %w", err)
		}

		time.Sleep(tenSeconds)
	}
}

func (talosService TalosService) VerifyNodeHealth(helperService interfaces.HelperServiceInterface, nodeIp string, controlPlaneIp string) error {
	logger.Info("Verifying cluster health, this might take a few minutes...")

	configFilePath := helperService.GetConfigFilePath(constants.TalosConfigFile)

	start := time.Now()
	timeout := fiveMinutes
	for {
		cmd := execCommand("talosctl", "--nodes", nodeIp, "--endpoints", controlPlaneIp, "health", fmt.Sprintf("--talosconfig=%s", configFilePath))
		output, err := cmd.CombinedOutput()
		logger.Debug(string(output))

		if err == nil {
			return nil
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("Cluster health check failed after 5 minutes: %w", err)
		}

		time.Sleep(tenSeconds)
	}
}

// GetDisks returns the disks Talos could be installed to. It reads talosctl's structured
// output rather than its table, whose columns are not a stable interface, and leaves out
// read-only devices and CD-ROMs - the loop devices backing the running system among them.
func (talosService TalosService) GetDisks(nodeIp string) ([]models.TalosDisk, error) {
	cmd := execCommand("talosctl", "-n", nodeIp, "get", "disks", "--insecure", "-o", "yaml")
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return nil, err
	}

	return parseDisks(output)
}

func parseDisks(output []byte) ([]models.TalosDisk, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(output))

	disks := []models.TalosDisk{}
	for {
		var disk models.TalosDisk
		err := decoder.Decode(&disk)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not parse disks: %w", err)
		}

		if disk.Spec.DevPath == "" || disk.Spec.ReadOnly || disk.Spec.CdRom {
			continue
		}

		disks = append(disks, disk)
	}

	return disks, nil
}

func (talosService TalosService) GetNetworkInterface(helperService interfaces.HelperServiceInterface, nodeIp string) (string, error) {
	cmd := execCommand("talosctl", "-n", nodeIp, "get", "addresses", "--talosconfig", helperService.GetConfigFilePath(constants.TalosConfigFile), "--insecure")
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return "", err
	}

	return parseNetworkInterface(string(output), nodeIp)
}

// parseNetworkInterface picks the link that currently holds nodeIp out of a
// "talosctl get addresses" table. Rows are matched on the address column rather than on the
// line as a whole: talosctl repeats the node's IP in the NODE column of every row, so a
// whole-line match returns every link on the machine - loopback included - instead of one.
func parseNetworkInterface(output string, nodeIp string) (string, error) {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		for _, field := range fields {
			// the address column carries the address in CIDR form, and the link it belongs
			// to is the last column; the trailing slash keeps 192.168.1.16 from matching
			// 192.168.1.161
			if strings.HasPrefix(field, nodeIp+"/") {
				return fields[len(fields)-1], nil
			}
		}
	}

	return "", fmt.Errorf("could not find a network interface holding %s in:\n%s", nodeIp, strings.TrimSpace(output))
}

// ModifyNetworkInterface, ModifyNetworkGateway and ModifyNetworkNodeIp all target the same
// LinkConfig document - bbe only ever manages a single interface, the same "one interface"
// assumption the old single-document format's Interfaces[0] convention made.
func (talosService TalosService) ModifyNetworkInterface(helperService interfaces.HelperServiceInterface, configFile string, networkInterfaceName string) error {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var linkConfig models.TalosLinkConfig
	index, err := getOrCreateDocument(&documents, models.TalosLinkConfigKind, &linkConfig)
	if err != nil {
		return err
	}

	linkConfig.Name = networkInterfaceName

	if err := setDocument(documents, index, linkConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

func (talosService TalosService) ModifyNetworkGateway(helperService interfaces.HelperServiceInterface, configFile string, gatewayIp string) error {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var linkConfig models.TalosLinkConfig
	index, err := getOrCreateDocument(&documents, models.TalosLinkConfigKind, &linkConfig)
	if err != nil {
		return err
	}

	linkConfig.Routes = []models.TalosLinkRoute{{Gateway: gatewayIp}}

	if err := setDocument(documents, index, linkConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

func (talosService TalosService) ModifyNetworkNodeIp(helperService interfaces.HelperServiceInterface, configFile string, nodeIp string) error {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var linkConfig models.TalosLinkConfig
	index, err := getOrCreateDocument(&documents, models.TalosLinkConfigKind, &linkConfig)
	if err != nil {
		return err
	}

	// LinkConfig addresses require CIDR notation (netip.Prefix); assume a /24, matching
	// ModifyNetworkGateway's existing hardcoded assumption of a typical home-LAN topology.
	linkConfig.Addresses = []models.TalosLinkAddress{{Address: fmt.Sprintf("%s/24", nodeIp)}}

	if err := setDocument(documents, index, linkConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

func (talosService TalosService) ModifyNetworkHostname(helperService interfaces.HelperServiceInterface, configFile string, hostname string) error {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var hostnameConfig models.TalosHostnameConfig
	index, err := getDocument(documents, models.TalosHostnameConfigKind, &hostnameConfig)
	if err != nil {
		return err
	}

	// hostname and auto are mutually exclusive in Talos's schema.
	hostnameConfig.Auto = ""
	hostnameConfig.Hostname = hostname

	if err := setDocument(documents, index, hostnameConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

// ModifyDnsServer points the node at a specific DNS server. Like the time server this has
// to be set explicitly: assigning a static address through LinkConfig disables DHCP on that
// link, leaving the node with no nameservers at all.
func (talosService TalosService) ModifyDnsServer(helperService interfaces.HelperServiceInterface, configFile string, dnsServer string) error {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var resolverConfig models.TalosResolverConfig
	index, err := getOrCreateDocument(&documents, models.TalosResolverConfigKind, &resolverConfig)
	if err != nil {
		return err
	}

	resolverConfig.Nameservers = []models.TalosNameserver{{Address: dnsServer}}

	if err := setDocument(documents, index, resolverConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

// ModifyTimeServer points the node at a specific NTP server. Talos otherwise falls back to
// a public time server, which leaves nodes unable to sync their clock on networks that
// don't allow outbound NTP - and a node whose clock is wrong fails certificate validation
// during boot. This still lives in the primary document; Talos has not moved it into a
// typed document of its own.
func (talosService TalosService) ModifyTimeServer(helperService interfaces.HelperServiceInterface, configFile string, timeServer string) error {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var machineConfig models.TalosMachineConfig
	index, err := getPrimaryDocument(documents, &machineConfig)
	if err != nil {
		return err
	}

	machineConfig.Machine.Time.Servers = []string{timeServer}

	if err := setDocument(documents, index, machineConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

func (talosService TalosService) ModifyConfigDisk(helperService interfaces.HelperServiceInterface, configFile string, disk string) error {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var installConfig models.TalosUnattendedInstallConfig
	index, err := getDocument(documents, models.TalosUnattendedInstallConfigKind, &installConfig)
	if err != nil {
		return err
	}

	// The disk is selected via a CEL expression, not a bare path.
	installConfig.Provisioning.DiskSelector.Match = fmt.Sprintf("disk.dev_path == %q", disk)

	if err := setDocument(documents, index, installConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

func (talosService TalosService) ModifySchedulingOnControlPlane(helperService interfaces.HelperServiceInterface, allowScheduling bool) error {
	configDir := helperService.GetConfigDir()
	configFile := constants.ControlplaneConfigFile

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var kubeNodeConfig models.TalosKubeNodeConfig
	index, err := getDocument(documents, models.TalosKubeNodeConfigKind, &kubeNodeConfig)
	if err != nil {
		return err
	}

	// Scheduling on control planes is controlled by the control-plane NoSchedule taint,
	// not a boolean field - allowing scheduling means removing that taint.
	kubeNodeConfig.Taints = map[string]string{}
	if !allowScheduling {
		kubeNodeConfig.Taints[models.TalosControlPlaneTaint] = "NoSchedule"
	}

	if err := setDocument(documents, index, kubeNodeConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

func (talosService TalosService) GetControlPlaneIp(helperService interfaces.HelperServiceInterface, configFile string) (string, error) {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return "", err
	}

	var kubeClusterConfig models.TalosKubeClusterConfig
	if _, err := getDocument(documents, models.TalosKubeClusterConfigKind, &kubeClusterConfig); err != nil {
		return "", err
	}

	endpoint := strings.TrimPrefix(kubeClusterConfig.Endpoint, "https://")
	endpoint = strings.TrimSuffix(endpoint, ":6443")

	return endpoint, nil
}

func (talosService TalosService) DownloadKubeConfig(helperService interfaces.HelperServiceInterface, nodeIp string, controlPlaneIp string) error {
	cmd := execCommand("talosctl", "kubeconfig", "--nodes", nodeIp, "--endpoints", controlPlaneIp, fmt.Sprintf("--talosconfig=%s", helperService.GetConfigFilePath(constants.TalosConfigFile)))
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return err
	}

	return nil
}

// getParsedDocuments reads every YAML document in a Talos machine config file. Since Talos
// v1.14, gen config splits settings across many small typed documents (separated by "---",
// each with its own "kind") instead of one monolithic document; every document must be
// preserved even though bbe only ever inspects a handful of them.
func getParsedDocuments(configDir string, configFile string) ([]map[string]interface{}, error) {
	initialTalosConfig, err := osReadFile(fmt.Sprintf("%s/%s", configDir, configFile))
	if err != nil {
		panic(err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(initialTalosConfig))

	var documents []map[string]interface{}
	for {
		var document map[string]interface{}
		err := decoder.Decode(&document)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		documents = append(documents, document)
	}

	return documents, nil
}

func writeDocuments(configDir string, configFile string, documents []map[string]interface{}) error {
	marshaledDocuments := make([]string, 0, len(documents))
	for _, document := range documents {
		marshaled, err := yaml.Marshal(document)
		if err != nil {
			return err
		}

		marshaledDocuments = append(marshaledDocuments, string(marshaled))
	}

	configToWrite := strings.Join(marshaledDocuments, "---\n")

	return osWriteFile(fmt.Sprintf("%s/%s", configDir, configFile), []byte(configToWrite), 0644)
}

// findDocument returns the document matching kind along with its position, or -1 when the
// config doesn't contain one.
func findDocument(documents []map[string]interface{}, kind string) (map[string]interface{}, int) {
	for index, document := range documents {
		if document["kind"] == kind {
			return document, index
		}
	}

	return nil, -1
}

// findPrimaryDocument returns the primary v1alpha1.Config document - the one document that
// carries no kind - along with its position, or -1 when the config doesn't contain one.
func findPrimaryDocument(documents []map[string]interface{}) (map[string]interface{}, int) {
	for index, document := range documents {
		if _, hasKind := document["kind"]; !hasKind {
			return document, index
		}
	}

	return nil, -1
}

// getPrimaryDocument decodes the primary document into target, returning its position so it
// can be written back with setDocument.
func getPrimaryDocument(documents []map[string]interface{}, target interface{}) (int, error) {
	document, index := findPrimaryDocument(documents)
	if index == -1 {
		return -1, fmt.Errorf("primary machine config document not found in config")
	}

	return index, decodeDocument(document, target)
}

// getDocument decodes the document matching kind into target, returning its position so it
// can be written back with setDocument.
func getDocument(documents []map[string]interface{}, kind string, target interface{}) (int, error) {
	document, index := findDocument(documents, kind)
	if index == -1 {
		return -1, fmt.Errorf("%s document not found in config", kind)
	}

	return index, decodeDocument(document, target)
}

// getOrCreateDocument behaves like getDocument, but appends a new document when the config
// doesn't contain one yet - Talos doesn't generate every document kind by default.
func getOrCreateDocument(documents *[]map[string]interface{}, kind string, target interface{}) (int, error) {
	document, index := findDocument(*documents, kind)
	if index == -1 {
		document = map[string]interface{}{
			"apiVersion": models.TalosConfigApiVersion,
			"kind":       kind,
		}
		*documents = append(*documents, document)
		index = len(*documents) - 1
	}

	return index, decodeDocument(document, target)
}

// setDocument encodes source back over the document at index, keeping every other document
// in the config untouched.
func setDocument(documents []map[string]interface{}, index int, source interface{}) error {
	encoded, err := yaml.Marshal(source)
	if err != nil {
		return err
	}

	document := make(map[string]interface{})
	if err := yaml.Unmarshal(encoded, &document); err != nil {
		return err
	}

	documents[index] = document

	return nil
}

// decodeDocument moves a raw document into a typed model. It goes through YAML rather than
// a map decoder so the models' yaml tags - including the inlined Unmapped fields that carry
// unknown keys - apply at every level of the document, not just the top one.
func decodeDocument(document map[string]interface{}, target interface{}) error {
	encoded, err := yaml.Marshal(document)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(encoded, target)
}
