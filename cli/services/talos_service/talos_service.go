package talos_service

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/command"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"gopkg.in/yaml.v2"
)

var execCommand = exec.Command
var tenSeconds = 10 * time.Second
var fiveMinutes = 5 * time.Minute

var osReadFile = os.ReadFile
var osWriteFile = os.WriteFile
var yamlMarshal = yaml.Marshal

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

// GenerateConfig generates a fresh cluster config, whose nodes reach the Kubernetes API at endpointIp, pinning
// --talos-version so the config schema matches the Talos version the node image was built for rather than whatever
// talosctl happens to be installed locally.
func (talosService TalosService) GenerateConfig(helperService interfaces.HelperServiceInterface, endpointIp string, clusterName string, talosVersion string) error {
	cmd := execCommand("talosctl", "gen", "config", clusterName, clusterEndpoint(endpointIp), "--talos-version", talosVersion, "--output", helperService.GetConfigDir())
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

// BootstrapCluster starts etcd on the first control plane, which the cluster then grows from. Talos is reached at the
// node's own address, as the address the control planes share only comes up once etcd runs
func (talosService TalosService) BootstrapCluster(helperService interfaces.HelperServiceInterface, nodeIp string) error {
	logger.Info("Bootstrapping cluster, this might take a few minutes...")

	configFilePath := helperService.GetConfigFilePath(constants.TalosConfigFile)

	start := time.Now()
	timeout := fiveMinutes
	for {
		cmd := execCommand("talosctl", "bootstrap", "--nodes", nodeIp, "--endpoints", nodeIp, fmt.Sprintf("--talosconfig=%s", configFilePath))
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

// VerifyNodeHealth waits for the cluster to be healthy, the node just set up included once it has joined. Only a
// control plane answers the health check, so each attempt asks the next of the control planes' addresses, in case
// one of them is down
func (talosService TalosService) VerifyNodeHealth(helperService interfaces.HelperServiceInterface, talosEndpoints []string) error {
	logger.Info("Verifying cluster health, this might take a few minutes...")

	if len(talosEndpoints) == 0 {
		return fmt.Errorf("No control plane to check the cluster's health through")
	}

	configFilePath := helperService.GetConfigFilePath(constants.TalosConfigFile)

	start := time.Now()
	timeout := fiveMinutes
	for attempt := 0; ; attempt++ {
		controlPlane := talosEndpoints[attempt%len(talosEndpoints)]
		cmd := execCommand("talosctl", "--nodes", controlPlane, "--endpoints", controlPlane, "health", fmt.Sprintf("--talosconfig=%s", configFilePath))
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
	output, err := command.Output(cmd)
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
	output, err := command.Output(cmd)
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

// Adds directory user volumes, which Talos creates at /var/mnt/<name> and pods can store data in, keeping any
// already there
func (talosService TalosService) ModifyUserVolumes(helperService interfaces.HelperServiceInterface, configFile string, names []string) error {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	for _, name := range names {
		exists := slices.ContainsFunc(documents, func(document map[string]interface{}) bool {
			return document["kind"] == models.TalosUserVolumeConfigKind && document["name"] == name
		})
		if exists {
			continue
		}

		documents = append(documents, map[string]interface{}{})
		volume := models.TalosUserVolumeConfig{
			ApiVersion: models.TalosConfigApiVersion,
			Kind:       models.TalosUserVolumeConfigKind,
			Name:       name,
			VolumeType: "directory",
		}
		if err := setDocument(documents, len(documents)-1, volume); err != nil {
			return err
		}
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

// ModifyInstallerImage sets the image Talos installs itself to disk from. `talosctl gen config` names a plain one,
// so a node installed from bbe's image would otherwise lose its system extensions, such as Longhorn's iscsi-tools
func (talosService TalosService) ModifyInstallerImage(helperService interfaces.HelperServiceInterface, configFile string, image string) error {
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

	installConfig.Installer.Image = image

	if err := setDocument(documents, index, installConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

// Has Talos run kube-network-policies alongside Flannel, so the NetworkPolicies packages bring are enforced. Talos sets
// up the cluster's network from the control plane's config, so it's only set there
func (talosService TalosService) ModifyNetworkPolicies(helperService interfaces.HelperServiceInterface) error {
	configDir := helperService.GetConfigDir()
	configFile := constants.ControlplaneConfigFile

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var flannelConfig models.TalosKubeFlannelCNIConfig
	index, err := getDocument(documents, models.TalosKubeFlannelCNIConfigKind, &flannelConfig)
	if err != nil {
		return err
	}

	flannelConfig.KubeNetworkPoliciesEnabled = true

	if err := setDocument(documents, index, flannelConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

// Serves the metrics of etcd, the controller manager, the scheduler and kube-proxy on the node's addresses rather than
// only on localhost, so a monitoring package can collect them. The controller manager and the scheduler still ask for
// a token, and the firewall keeps all four to the trusted networks and pods
func (talosService TalosService) ModifyControlPlaneMetrics(helperService interfaces.HelperServiceInterface) error {
	configDir := helperService.GetConfigDir()
	configFile := constants.ControlplaneConfigFile

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var machineConfig models.TalosMachineConfig
	index, err := getPrimaryDocument(documents, &machineConfig)
	if err != nil {
		return err
	}

	if machineConfig.Cluster.Etcd.ExtraArgs == nil {
		machineConfig.Cluster.Etcd.ExtraArgs = map[string]interface{}{}
	}
	machineConfig.Cluster.Etcd.ExtraArgs["listen-metrics-urls"] = "http://0.0.0.0:2381"

	if err := setDocument(documents, index, machineConfig); err != nil {
		return err
	}

	for _, kind := range []string{models.TalosKubeControllerManagerConfigKind, models.TalosKubeSchedulerConfigKind} {
		var componentConfig models.TalosKubeComponentConfig
		index, err := getDocument(documents, kind, &componentConfig)
		if err != nil {
			return err
		}

		if componentConfig.ExtraArgs == nil {
			componentConfig.ExtraArgs = map[string]interface{}{}
		}
		componentConfig.ExtraArgs["bind-address"] = "0.0.0.0"

		if err := setDocument(documents, index, componentConfig); err != nil {
			return err
		}
	}

	var proxyConfig models.TalosKubeProxyConfig
	index, err = getDocument(documents, models.TalosKubeProxyConfigKind, &proxyConfig)
	if err != nil {
		return err
	}

	if proxyConfig.Config == nil {
		proxyConfig.Config = map[string]interface{}{}
	}
	proxyConfig.Config["metricsBindAddress"] = "0.0.0.0:10249"

	if err := setDocument(documents, index, proxyConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

type firewallRule struct {
	name     string
	ports    []interface{}
	protocol string
	// Whether pods may connect too, besides the trusted networks
	fromPods bool
}

// The ports of Talos and Kubernetes themselves, which the firewall keeps to the trusted networks: the nodes and the
// computers managing them. Every other port, the apps' included, stays open. Workers get the same rules, covering
// ports only control planes listen on
var firewallRules = []firewallRule{
	// talosctl, and the nodes reaching each other's Talos API
	{name: "bbe-apid", ports: []interface{}{50000}, protocol: "tcp"},
	// Nodes getting their certificates from the control plane
	{name: "bbe-trustd", ports: []interface{}{50001}, protocol: "tcp"},
	{name: "bbe-etcd", ports: []interface{}{"2379-2380"}, protocol: "tcp"},
	// Pods reach the Kubernetes API through its service, which leads to the node's address
	{name: "bbe-kubernetes-api", ports: []interface{}{6443}, protocol: "tcp", fromPods: true},
	// The API server, for logs and exec, and metrics-server and Prometheus from their pods
	{name: "bbe-kubelet", ports: []interface{}{10250}, protocol: "tcp", fromPods: true},
	// The metrics of etcd, kube-proxy, the controller manager, the scheduler and node-exporter, for Prometheus
	{name: "bbe-metrics", ports: []interface{}{2381, 9100, 10249, 10257, 10259}, protocol: "tcp", fromPods: true},
	// Flannel's VXLAN between the nodes. Open, anyone on the network could send packets into the pod network
	{name: "bbe-flannel", ports: []interface{}{4789}, protocol: "udp"},
}

// Keeps the ports of Talos and Kubernetes to the trusted networks, given as CIDRs, replacing the rules an earlier run
// added
func (talosService TalosService) ModifyFirewall(helperService interfaces.HelperServiceInterface, configFile string, trustedNetworks []string) error {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var networkConfig models.TalosKubeNetworkConfig
	if _, err := getDocument(documents, models.TalosKubeNetworkConfigKind, &networkConfig); err != nil {
		return err
	}

	documents = slices.DeleteFunc(documents, func(document map[string]interface{}) bool {
		return document["kind"] == models.TalosNetworkRuleConfigKind && slices.ContainsFunc(firewallRules, func(rule firewallRule) bool {
			return document["name"] == rule.name
		})
	})

	for _, rule := range firewallRules {
		subnets := trustedNetworks
		if rule.fromPods {
			subnets = slices.Concat(trustedNetworks, networkConfig.PodSubnets)
		}

		ingress := make([]models.TalosNetworkRuleIngress, len(subnets))
		for i, subnet := range subnets {
			ingress[i] = models.TalosNetworkRuleIngress{Subnet: subnet}
		}

		documents = append(documents, map[string]interface{}{})
		networkRule := models.TalosNetworkRuleConfig{
			ApiVersion:   models.TalosConfigApiVersion,
			Kind:         models.TalosNetworkRuleConfigKind,
			Name:         rule.name,
			PortSelector: models.TalosNetworkPortSelector{Ports: rule.ports, Protocol: rule.protocol},
			Ingress:      ingress,
		}
		if err := setDocument(documents, len(documents)-1, networkRule); err != nil {
			return err
		}
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

// DownloadKubeConfig adds the cluster to kubectl's config, from the first control plane that answers. It replaces the
// cluster's earlier entry, which is how kubectl moves to the address the control planes share
func (talosService TalosService) DownloadKubeConfig(helperService interfaces.HelperServiceInterface, talosEndpoints []string) error {
	configFilePath := helperService.GetConfigFilePath(constants.TalosConfigFile)

	return onAnyControlPlane(talosEndpoints, func(controlPlane string) error {
		cmd := execCommand("talosctl", "kubeconfig", "--force", "--nodes", controlPlane, "--endpoints", controlPlane, fmt.Sprintf("--talosconfig=%s", configFilePath))
		output, err := cmd.CombinedOutput()
		logger.Debug(string(output))

		return err
	})
}

// GetTalosEndpoints returns the control planes' own addresses, which talosctl reaches the Talos API through. bbe
// records them in talosconfig as control planes join; a cluster created by an earlier bbe has none there, and its
// only control plane is the one the Kubernetes API's address in the configs leads to
func (talosService TalosService) GetTalosEndpoints(helperService interfaces.HelperServiceInterface) ([]string, error) {
	content, err := osReadFile(helperService.GetConfigFilePath(constants.TalosConfigFile))
	if err != nil {
		return nil, err
	}

	var clientConfig models.TalosClientConfig
	if err := yaml.Unmarshal(content, &clientConfig); err != nil {
		return nil, fmt.Errorf("could not read talosconfig: %w", err)
	}

	if endpoints := clientConfig.Contexts[clientConfig.Context].Endpoints; len(endpoints) > 0 {
		return endpoints, nil
	}

	controlPlaneIp, err := talosService.GetControlPlaneIp(helperService, constants.ControlplaneConfigFile)
	if err != nil {
		return nil, err
	}

	return []string{controlPlaneIp}, nil
}

// SetTalosEndpoints records the control planes' own addresses in talosconfig, so bbe and talosctl reach the cluster
// through any of them. The address they share isn't one: it only answers while etcd is healthy, and moves between
// control planes, cutting off whatever talosctl was doing
func (talosService TalosService) SetTalosEndpoints(helperService interfaces.HelperServiceInterface, talosEndpoints []string) error {
	args := append([]string{"--talosconfig", helperService.GetConfigFilePath(constants.TalosConfigFile), "config", "endpoint"}, talosEndpoints...)
	cmd := execCommand("talosctl", args...)
	output, err := cmd.CombinedOutput()
	logger.Debug(string(output))

	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// GetMembers lists the cluster's nodes, as the first control plane that answers knows them
func (talosService TalosService) GetMembers(helperService interfaces.HelperServiceInterface, talosEndpoints []string) ([]models.TalosMember, error) {
	configFilePath := helperService.GetConfigFilePath(constants.TalosConfigFile)

	var members []models.TalosMember
	err := onAnyControlPlane(talosEndpoints, func(controlPlane string) error {
		cmd := execCommand("talosctl", "get", "members", "-o", "yaml", "--nodes", controlPlane, "--endpoints", controlPlane, fmt.Sprintf("--talosconfig=%s", configFilePath))
		output, err := command.Output(cmd)
		logger.Debug(string(output))
		if err != nil {
			return err
		}

		members, err = parseMembers(output)
		return err
	})

	return members, err
}

func parseMembers(output []byte) ([]models.TalosMember, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(output))

	members := []models.TalosMember{}
	for {
		var member models.TalosMember
		err := decoder.Decode(&member)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("could not parse the cluster's members: %w", err)
		}

		if len(member.Spec.Addresses) == 0 {
			continue
		}

		members = append(members, member)
	}

	return members, nil
}

// GetNodeNetworkInterface is GetNetworkInterface for a node already in the cluster, which only answers talosctl
// with talosconfig's certificate
func (talosService TalosService) GetNodeNetworkInterface(helperService interfaces.HelperServiceInterface, talosEndpoints []string, nodeIp string) (string, error) {
	cmd := execCommand("talosctl", "get", "addresses", "--nodes", nodeIp, "--endpoints", strings.Join(talosEndpoints, ","), fmt.Sprintf("--talosconfig=%s", helperService.GetConfigFilePath(constants.TalosConfigFile)))
	output, err := command.Output(cmd)
	logger.Debug(string(output))

	if err != nil {
		return "", err
	}

	return parseNetworkInterface(string(output), nodeIp)
}

// MoveNodeToSharedIp points a node of the cluster at the Kubernetes API on the address the control planes share. A
// control plane, given the network interface to hold the address on, also takes part in holding it. Talos applies
// both without restarting the node
func (talosService TalosService) MoveNodeToSharedIp(helperService interfaces.HelperServiceInterface, talosEndpoints []string, nodeIp string, sharedIp string, networkInterface string) error {
	patches := []interface{}{
		models.TalosKubeClusterConfig{
			ApiVersion: models.TalosConfigApiVersion,
			Kind:       models.TalosKubeClusterConfigKind,
			Endpoint:   clusterEndpoint(sharedIp),
		},
	}
	if networkInterface != "" {
		patches = append(patches, models.TalosLayer2VIPConfig{
			ApiVersion: models.TalosConfigApiVersion,
			Kind:       models.TalosLayer2VIPConfigKind,
			Name:       sharedIp,
			Link:       networkInterface,
		})
	}

	args := []string{"patch", "machineconfig", "--nodes", nodeIp, "--endpoints", strings.Join(talosEndpoints, ","), fmt.Sprintf("--talosconfig=%s", helperService.GetConfigFilePath(constants.TalosConfigFile))}
	for _, patch := range patches {
		marshaled, err := yamlMarshal(patch)
		if err != nil {
			return err
		}
		args = append(args, "--patch", string(marshaled))
	}

	cmd := execCommand("talosctl", args...)
	output, err := command.Output(cmd)
	logger.Debug(string(output))

	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// GetSharedIp returns the address the control planes share for the Kubernetes API, or nothing when the cluster was
// created by an earlier bbe without one
func (talosService TalosService) GetSharedIp(helperService interfaces.HelperServiceInterface) (string, error) {
	documents, err := getParsedDocuments(helperService.GetConfigDir(), constants.ControlplaneConfigFile)
	if err != nil {
		return "", err
	}

	document, index := findDocument(documents, models.TalosLayer2VIPConfigKind)
	if index == -1 {
		return "", nil
	}

	var vipConfig models.TalosLayer2VIPConfig
	if err := decodeDocument(document, &vipConfig); err != nil {
		return "", err
	}

	return vipConfig.Name, nil
}

// ModifySharedIp has the control plane set up from the config take part in holding the address the control planes
// share, on its network interface
func (talosService TalosService) ModifySharedIp(helperService interfaces.HelperServiceInterface, sharedIp string, networkInterface string) error {
	configDir := helperService.GetConfigDir()
	configFile := constants.ControlplaneConfigFile

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var vipConfig models.TalosLayer2VIPConfig
	index, err := getOrCreateDocument(&documents, models.TalosLayer2VIPConfigKind, &vipConfig)
	if err != nil {
		return err
	}

	vipConfig.Name = sharedIp
	vipConfig.Link = networkInterface

	if err := setDocument(documents, index, vipConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

// ModifyClusterEndpoint points the nodes set up from the config at the Kubernetes API on the given address
func (talosService TalosService) ModifyClusterEndpoint(helperService interfaces.HelperServiceInterface, configFile string, ip string) error {
	configDir := helperService.GetConfigDir()

	documents, err := getParsedDocuments(configDir, configFile)
	if err != nil {
		return err
	}

	var kubeClusterConfig models.TalosKubeClusterConfig
	index, err := getDocument(documents, models.TalosKubeClusterConfigKind, &kubeClusterConfig)
	if err != nil {
		return err
	}

	kubeClusterConfig.Endpoint = clusterEndpoint(ip)

	if err := setDocument(documents, index, kubeClusterConfig); err != nil {
		return err
	}

	return writeDocuments(configDir, configFile, documents)
}

func clusterEndpoint(ip string) string {
	return fmt.Sprintf("https://%s:6443", ip)
}

// onAnyControlPlane runs action against each control plane in turn, until one succeeds
func onAnyControlPlane(talosEndpoints []string, action func(controlPlane string) error) error {
	if len(talosEndpoints) == 0 {
		return fmt.Errorf("No control plane to reach the cluster through")
	}

	var err error
	for _, controlPlane := range talosEndpoints {
		if err = action(controlPlane); err == nil {
			return nil
		}
		logger.Debug(fmt.Sprintf("Control plane %s didn't answer: %s", controlPlane, err))
	}

	return err
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
	encoded, err := yamlMarshal(source)
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
