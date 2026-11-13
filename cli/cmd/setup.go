package cmd

import (
	"fmt"
	"net/netip"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/config_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/dependency_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helper_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/image_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/ipfinder_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/talos_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/ui_service"
	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/lucasepe/codename"
	"github.com/spf13/cobra"
)

var italic = color.New(color.Italic).SprintFunc()

var setupCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{},
	Short:   "Guides you through a BBE-Quest node setup",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		helperService := helper_service.HelperService{}
		dependencyService := dependency_service.DependencyService{}
		talosService := talos_service.TalosService{}
		ipFinderService := ipfinder_service.IpFinderService{}
		uiService := ui_service.UiService{}
		configService := config_service.ConfigService{}
		imageService := image_service.ImageService{}

		err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)
		if err != nil {
			logger.Error("", err)
			os.Exit(1)
		}
	},
}

func setupCommand(helperService interfaces.HelperServiceInterface, dependencyService interfaces.DependencyServiceInterface, talosService interfaces.TalosServiceInterface, ipFinderService interfaces.IpFinderServiceInterface, uiService interfaces.UiServiceInterface, configService interfaces.ConfigServiceInterface, imageService interfaces.ImageServiceInterface) error {
	rng, rngError := codename.DefaultRNG()

	spinner := spinner.New(spinner.CharSets[43], 100*time.Millisecond)

	bbeConfig, err := configService.GetBbeConfig(helperService)
	if err != nil {
		bbeConfig, err = getOrGenerateConfig(helperService, uiService, configService)
		if err != nil {
			return fmt.Errorf("Error while generating BBE config: %w", err)
		}
	}

	if bbeConfig.Bbe.Storage.Type == "aws" {
		err := configService.SyncConfigsWithAws(helperService, bbeConfig)
		if err != nil {
			return fmt.Errorf("Error while syncing config with AWS: %w", err)
		}
	}

	// Get current local path
	workingDirectory, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	if !dependencyService.VerifyDependencies() {
		return fmt.Errorf("Error while verifying dependencies")
	}

	answer, err := uiService.CreateSelect("Is this the first node in your cluster?", []string{"Yes", "No"})
	if err != nil {
		panic(err)
	}
	createCluster := answer == "Yes"

	configExists := configService.CheckForTalosConfigs(helperService)

	if createCluster && configExists {
		return fmt.Errorf("You are trying to create a control plane node, but there are already config files present, in %s", helperService.GetConfigDir())
	}

	if !configExists && !createCluster {
		return fmt.Errorf("No config files found while trying to enroll new node in existing cluster, please create your first node first")
	}

	controlPlane := createCluster
	// The control planes' own addresses, which talosctl reaches the cluster through
	var talosEndpoints []string
	// The address the control planes share for the Kubernetes API, which moves to another one when one goes down
	var sharedIp string
	if !createCluster {
		answer, err = uiService.CreateSelect(joinAsQuestion, []string{joinAsWorker, joinAsControlPlane})
		if err != nil {
			panic(err)
		}
		controlPlane = answer == joinAsControlPlane

		talosEndpoints, err = talosService.GetTalosEndpoints(helperService)
		if err != nil {
			return fmt.Errorf("Error while finding the cluster's control planes: %w", err)
		}

		if controlPlane {
			sharedIp, talosEndpoints, err = offerSharedIp(helperService, talosService, uiService, talosEndpoints)
			if err != nil {
				return err
			}
		}
	}

	answer, err = uiService.CreateSelect("What type of device are you setting up?", []string{"Bare Metal", "Single Board Computer"})
	if err != nil {
		panic(err)
	}

	var nodeType models.NodeType
	switch answer {
	case "Bare Metal":
		nodeType = image_service.IntelNuc
	case "Single Board Computer":
		nodeType = image_service.RaspberryPi
	default:
		panic("Invalid node type")
	}

	err = imageCreation(helperService, uiService, imageService, workingDirectory, nodeType)
	if err != nil {
		return fmt.Errorf("Error while downloading image: %w", err)
	}

	firstMessage := "Please use balenaEtcher to flash the .iso to your USB device"
	secondMessage := "Please insert the USB device into your new node and boot from it"

	if nodeType.ImagerType == "rpi_generic" {
		firstMessage = "Please use balenaEtcher to flash the .xz to your SD card"
		secondMessage = "Please insert the SD card into your new node and boot from it"
	}

	_, err = uiService.CreateSelect(firstMessage, []string{"Done"})
	if err != nil {
		panic(err)
	}

	_, err = uiService.CreateSelect(secondMessage, []string{"Done"})
	if err != nil {
		panic(err)
	}

	spinner.Start()
	gatewayIpSuggestion, err := ipFinderService.GetGatewayIp(helperService)
	if err != nil {
		var result string
		title := "Gateway IP not found, please enter the IP of the network you want to scan:"
		for {
			var err error
			result, err = uiService.CreateInput(title, "")
			if err != nil {
				panic(err)
			}

			if helperService.IsValidIp(result) {
				gatewayIpSuggestion = result
				break
			}
			title = "Invalid Gateway IP, please enter a valid IP:"
		}
	}

	ips, err := ipFinderService.LocateDevice(helperService, talosService, gatewayIpSuggestion)
	if err != nil {
		return fmt.Errorf("Error while attempting to locate device: %w", err)
	}
	spinner.Stop()

	logger.Infof("Found %d Talos device(s)", len(ips))

	if len(ips) > 1 {
		return fmt.Errorf("More than one node found, please make sure there is only 1 Talos node in maintenance mode.")
	}

	if len(ips) == 0 {
		return fmt.Errorf("No node found, please make sure there is only 1 Talos node in maintenance mode.")
	}

	originalIp := ips[0]

	///////////////////////////////////////////////////////////////////////////////// QUESTIONS ///////////////////////////////////////////////////////////////////////////////////////////////////////////////
	chosenIp, err := uiService.CreateInput(fmt.Sprintf("Please choose an ip for the new node %s", italic("(press enter to accept the suggested ip)")), originalIp)
	if err != nil {
		panic(err)
	}

	logger.Debug("Getting talos disks")
	disks, err := talosService.GetDisks(originalIp)
	if err != nil {
		return fmt.Errorf("Error while getting disks: %w", err)
	}

	if len(disks) == 0 {
		return fmt.Errorf("No disk Talos could be installed to was found on %s", originalIp)
	}

	diskLabels := make([]string, len(disks))
	for i, disk := range disks {
		diskLabels[i] = diskLabel(disk)
	}

	selectedDisk, err := uiService.CreateSelect(fmt.Sprintf("Please select the disk to install Talos on for %s", chosenIp), diskLabels)
	if err != nil {
		panic(err)
	}
	installDisk := disks[slices.Index(diskLabels, selectedDisk)].Spec.DevPath

	gatewayIp, err := uiService.CreateInput(fmt.Sprintf("Please choose the correct gateway ip %s", italic("(press enter to accept the suggested gateway ip)")), gatewayIpSuggestion)
	if err != nil {
		panic(err)
	}

	timeServer, err := uiService.CreateInput(fmt.Sprintf("Please choose the time server %s", italic("(press enter to accept the suggested time server)")), gatewayIp)
	if err != nil {
		panic(err)
	}

	dnsServer, err := uiService.CreateInput(fmt.Sprintf("Please choose the DNS server %s", italic("(press enter to accept the suggested dns server)")), gatewayIp)
	if err != nil {
		panic(err)
	}

	suggestedHostname := "big_brain_entropy_generator"
	if rngError == nil {
		suggestedHostname = codename.Generate(rng, 0)
	}

	hostname, err := uiService.CreateInput(fmt.Sprintf("Please select the hostname %s", italic("(press enter to accept the suggested name)")), suggestedHostname)
	if err != nil {
		panic(err)
	}

	var clusterName string
	var allowSchedulingOnControlPlanes string
	var trustedNetworks []string
	if createCluster {
		suggestedClusterName := "big_brain_entropy_holder"
		if rngError == nil {
			suggestedClusterName = codename.Generate(rng, 0)
		}

		clusterName, err = uiService.CreateInput("Please enter what you want to name your cluster", suggestedClusterName)
		if err != nil {
			panic(err)
		}

		sharedIp, err = askSharedIp(uiService, chosenIp, []string{chosenIp, gatewayIp})
		if err != nil {
			return fmt.Errorf("Error while asking for the control planes' shared address: %w", err)
		}

		// The nodes reach the Kubernetes API at the shared address, so they keep reaching it when a control plane is down
		err = talosService.GenerateConfig(helperService, sharedIp, clusterName, nodeType.TalosVersion)
		if err != nil {
			return fmt.Errorf("Error while generating config: %w", err)
		}

		trustedNetworks, err = askTrustedNetworks(helperService, ipFinderService, uiService, chosenIp)
		if err != nil {
			return fmt.Errorf("Error while asking for the trusted networks: %w", err)
		}

		talosEndpoints = []string{chosenIp}
	}

	if controlPlane {
		allowSchedulingOnControlPlanes, err = uiService.CreateSelect("Do you want to allow scheduling on the control plane? This is required if you have only one node.", []string{"Yes", "No"})
		if err != nil {
			panic(err)
		}

	}
	///////////////////////////////////////////////////////////////////////////////// QUESTIONS END ///////////////////////////////////////////////////////////////////////////////////////////////////////////////

	logger.Debug(fmt.Sprintf("Control planes: %s", strings.Join(talosEndpoints, ", ")))

	logger.Debug("Checking if talosctl matches the version the node image was built for")
	talosService.WarnIfTalosVersionMismatch(nodeType.TalosVersion)

	// A new cluster's settings for every node go into both configs, which each later node is set up from
	if trustedNetworks != nil {
		logger.Debug("Enforcing network policies, serving the control plane's metrics and setting up the firewall")
		err = talosService.ModifyNetworkPolicies(helperService)
		if err != nil {
			return fmt.Errorf("Error while enforcing network policies in file: %w", err)
		}

		err = talosService.ModifyControlPlaneMetrics(helperService)
		if err != nil {
			return fmt.Errorf("Error while storing the control plane's metrics addresses in file: %w", err)
		}

		for _, configFile := range []string{constants.ControlplaneConfigFile, constants.WorkerConfigFile} {
			err = talosService.ModifyFirewall(helperService, configFile, trustedNetworks)
			if err != nil {
				return fmt.Errorf("Error while storing the firewall in file: %w", err)
			}
		}
	}

	nodeConfigFile := constants.WorkerConfigFile
	if controlPlane {
		nodeConfigFile = constants.ControlplaneConfigFile
	}

	logger.Debug(fmt.Sprintf("Working on the config file: %s", nodeConfigFile))

	err = talosService.ModifyNetworkNodeIp(helperService, nodeConfigFile, chosenIp)
	if err != nil {
		return fmt.Errorf("Error while storing the Node IP in file: %w", err)
	}

	networkInterface, err := talosService.GetNetworkInterface(helperService, originalIp)
	if err != nil {
		return fmt.Errorf("Error while getting network interface: %w", err)
	}

	err = talosService.ModifyNetworkInterface(helperService, nodeConfigFile, networkInterface)
	if err != nil {
		return fmt.Errorf("Error while storing the network interface in file: %w", err)
	}

	if controlPlane && sharedIp != "" {
		err = talosService.ModifySharedIp(helperService, sharedIp, networkInterface)
		if err != nil {
			return fmt.Errorf("Error while storing the control planes' shared address in file: %w", err)
		}
	}

	err = talosService.ModifyNetworkGateway(helperService, nodeConfigFile, gatewayIp)
	if err != nil {
		return fmt.Errorf("Error while storing the Gateway IP in file: %w", err)
	}

	err = talosService.ModifyNetworkHostname(helperService, nodeConfigFile, hostname)
	if err != nil {
		return fmt.Errorf("Error while storing the hostname in file: %w", err)
	}

	err = talosService.ModifyTimeServer(helperService, nodeConfigFile, timeServer)
	if err != nil {
		return fmt.Errorf("Error while storing the time server in file: %w", err)
	}

	err = talosService.ModifyDnsServer(helperService, nodeConfigFile, dnsServer)
	if err != nil {
		return fmt.Errorf("Error while storing the DNS server in file: %w", err)
	}

	err = talosService.ModifyUserVolumes(helperService, nodeConfigFile, constants.UserVolumes)
	if err != nil {
		return fmt.Errorf("Error while storing the user volumes in file: %w", err)
	}

	if allowSchedulingOnControlPlanes != "" {
		scheduleOnControlPlane := allowSchedulingOnControlPlanes == "Yes"
		err = talosService.ModifySchedulingOnControlPlane(helperService, scheduleOnControlPlane)
		if err != nil {
			return fmt.Errorf("Error while storing controlplane scheduling in file: %w", err)
		}
	}

	spinner.Start()
	logger.Debug("Modifying talos config disk")
	err = talosService.ModifyConfigDisk(helperService, nodeConfigFile, installDisk)
	if err != nil {
		return fmt.Errorf("Error while modifying config disk: %w", err)
	}

	logger.Debug("Modifying talos config installer image")
	err = talosService.ModifyInstallerImage(helperService, nodeConfigFile, nodeType.InstallerImage)
	if err != nil {
		return fmt.Errorf("Error while modifying config installer image: %w", err)
	}

	logger.Debug("Joining cluster")
	err = talosService.JoinCluster(helperService, originalIp, nodeConfigFile)
	if err != nil {
		return fmt.Errorf("Error while joining cluster: %w", err)
	}

	if createCluster {
		err := talosService.BootstrapCluster(helperService, chosenIp)
		if err != nil {
			return fmt.Errorf("Error while bootstrapping cluster: %w", err)
		}

		logger.Infof("Cluster bootstrapping successfully requested at %s", chosenIp)
	}

	if controlPlane {
		if !slices.Contains(talosEndpoints, chosenIp) {
			talosEndpoints = append(talosEndpoints, chosenIp)
		}

		err = talosService.SetTalosEndpoints(helperService, talosEndpoints)
		if err != nil {
			return fmt.Errorf("Error while recording the control planes in talosconfig: %w", err)
		}
	}

	err = talosService.VerifyNodeHealth(helperService, talosEndpoints)
	if err != nil {
		return fmt.Errorf("Error while verifying node health: %w", err)
	}

	switch {
	case createCluster:
		logger.Debug("Downloading kube config")
		err := talosService.DownloadKubeConfig(helperService, talosEndpoints)
		if err != nil {
			return fmt.Errorf("Error while downloading kubeconfig: %w", err)
		}

		logger.Debug("Updating BBE cluster name")
		err = configService.UpdateBbeClusterName(helperService, clusterName)
		if err != nil {
			return fmt.Errorf("Error while updating BBE cluster name: %w", err)
		}

		logger.Infof("Control plane node %s successfully set up, with the Kubernetes API at %s", chosenIp, sharedIp)
	case controlPlane:
		logger.Infof("Control plane node %s successfully set up", chosenIp)
	default:
		logger.Infof("Worker node %s successfully set up", chosenIp)
	}
	spinner.Stop()

	if controlPlane && len(talosEndpoints)%2 == 0 {
		logger.Warning(fmt.Sprintf("The cluster has %d control planes. etcd needs more than half of them running, so it survives no more control planes going down than with %d. Add another to make the most of them", len(talosEndpoints), len(talosEndpoints)-1))
	}

	return nil
}

const joinAsQuestion = "Should this node run the control plane too? With three control planes, the cluster keeps working when one of them goes down"
const joinAsWorker = "No, it's a worker"
const joinAsControlPlane = "Yes, it's a control plane"

// Asks for the address the control planes share for the Kubernetes API, which has to be in nodeIp's network and
// not taken yet
func askSharedIp(uiService interfaces.UiServiceInterface, nodeIp string, taken []string) (string, error) {
	nodeAddress, err := netip.ParseAddr(nodeIp)
	if err != nil {
		return "", err
	}

	title := fmt.Sprintf("Which address should the control planes share for the Kubernetes API? Another control plane takes it over when one goes down. Pick a free one in %s that your router doesn't hand out", netip.PrefixFrom(nodeAddress, 24).Masked())
	for {
		answer, err := uiService.CreateInput(title, "")
		if err != nil {
			panic(err)
		}

		sharedIp, problem := parseSharedIp(answer, nodeAddress, taken)
		if problem == "" {
			return sharedIp, nil
		}

		title = fmt.Sprintf("%s. Please enter the shared address again:", problem)
	}
}

// Reads an address such as "192.168.1.250", or says what's wrong with it. Control planes hold it on the link they
// have their own address on, which bbe sets up as a /24
func parseSharedIp(answer string, nodeAddress netip.Addr, taken []string) (string, string) {
	answer = strings.TrimSpace(answer)
	address, err := netip.ParseAddr(answer)
	if err != nil {
		return "", fmt.Sprintf("%q isn't an address such as 192.168.1.250", answer)
	}

	network := netip.PrefixFrom(nodeAddress, 24).Masked()
	if !network.Contains(address) {
		return "", fmt.Sprintf("%s isn't in the node's network, %s", address, network)
	}

	if address == network.Addr() || !network.Contains(address.Next()) {
		return "", fmt.Sprintf("%s is the network's own address or its broadcast address", address)
	}

	if slices.Contains(taken, address.String()) {
		return "", fmt.Sprintf("%s is already a node's or the gateway's address", address)
	}

	return address.String(), ""
}

// Offers to move a cluster created by an earlier bbe, which reaches the Kubernetes API at its first control plane's
// own address, to an address the control planes share. Without it kubectl, bbe and the nodes lose the Kubernetes API
// while that control plane is down, however many others there are. Returns the shared address, or nothing when the
// cluster keeps its single one, and the control planes' own addresses
func offerSharedIp(helperService interfaces.HelperServiceInterface, talosService interfaces.TalosServiceInterface, uiService interfaces.UiServiceInterface, talosEndpoints []string) (string, []string, error) {
	sharedIp, err := talosService.GetSharedIp(helperService)
	if err != nil {
		return "", nil, fmt.Errorf("Error while reading the control planes' shared address: %w", err)
	}
	if sharedIp != "" {
		return sharedIp, talosEndpoints, nil
	}

	endpointIp, err := talosService.GetControlPlaneIp(helperService, constants.ControlplaneConfigFile)
	if err != nil {
		return "", nil, fmt.Errorf("Error while getting control plane IP: %w", err)
	}

	answer, err := uiService.CreateSelect(fmt.Sprintf("The cluster's Kubernetes API is at %s, a single control plane's address, so it can't be reached while that one is down. Move it to an address the control planes share? Every node is updated, without restarting", endpointIp), []string{"Yes", "No"})
	if err != nil {
		panic(err)
	}
	if answer != "Yes" {
		return "", talosEndpoints, nil
	}

	members, err := talosService.GetMembers(helperService, talosEndpoints)
	if err != nil {
		return "", nil, fmt.Errorf("Error while listing the cluster's nodes: %w", err)
	}

	var controlPlanes, workers, taken []string
	for _, member := range members {
		taken = append(taken, member.Spec.Addresses...)
		if member.Spec.MachineType == models.TalosControlPlaneMachineType {
			controlPlanes = append(controlPlanes, member.Spec.Addresses[0])
		} else {
			workers = append(workers, member.Spec.Addresses[0])
		}
	}
	if len(controlPlanes) == 0 {
		return "", nil, fmt.Errorf("No control plane found among the cluster's nodes")
	}

	sharedIp, err = askSharedIp(uiService, controlPlanes[0], taken)
	if err != nil {
		return "", nil, fmt.Errorf("Error while asking for the control planes' shared address: %w", err)
	}

	// The control planes first, which hold the address as soon as they have it, so the nodes pointed at it next
	// find the Kubernetes API there
	var networkInterface string
	for _, controlPlane := range controlPlanes {
		logger.Infof("Moving control plane %s to %s", controlPlane, sharedIp)
		networkInterface, err = talosService.GetNodeNetworkInterface(helperService, talosEndpoints, controlPlane)
		if err != nil {
			return "", nil, fmt.Errorf("Error while getting the network interface of %s: %w", controlPlane, err)
		}

		err = talosService.MoveNodeToSharedIp(helperService, talosEndpoints, controlPlane, sharedIp, networkInterface)
		if err != nil {
			return "", nil, fmt.Errorf("Error while moving control plane %s to %s: %w", controlPlane, sharedIp, err)
		}
	}

	// Workers reach the Kubernetes API through every control plane on their own (Talos's KubePrism), so one that can't
	// be updated now keeps working
	for _, worker := range workers {
		logger.Infof("Moving worker %s to %s", worker, sharedIp)
		err = talosService.MoveNodeToSharedIp(helperService, talosEndpoints, worker, sharedIp, "")
		if err != nil {
			logger.Warning(fmt.Sprintf("Couldn't move worker %s to %s, which keeps reaching the Kubernetes API through the control planes: %s. Point it at the new address later with `talosctl patch machineconfig`", worker, sharedIp, err))
		}
	}

	for _, configFile := range []string{constants.ControlplaneConfigFile, constants.WorkerConfigFile} {
		err = talosService.ModifyClusterEndpoint(helperService, configFile, sharedIp)
		if err != nil {
			return "", nil, fmt.Errorf("Error while storing the shared address in file: %w", err)
		}
	}

	// Recorded right away, so a setup stopped before the new control plane is set up doesn't offer the move again.
	// The new control plane replaces the network interface with its own
	err = talosService.ModifySharedIp(helperService, sharedIp, networkInterface)
	if err != nil {
		return "", nil, fmt.Errorf("Error while storing the control planes' shared address in file: %w", err)
	}

	err = talosService.SetTalosEndpoints(helperService, controlPlanes)
	if err != nil {
		return "", nil, fmt.Errorf("Error while recording the control planes in talosconfig: %w", err)
	}

	err = talosService.VerifyNodeHealth(helperService, controlPlanes)
	if err != nil {
		return "", nil, fmt.Errorf("Error while verifying the cluster's health on the shared address: %w", err)
	}

	err = talosService.DownloadKubeConfig(helperService, controlPlanes)
	if err != nil {
		return "", nil, fmt.Errorf("Error while downloading kubeconfig: %w", err)
	}

	logger.Infof("The Kubernetes API is now at %s, which kubectl uses too", sharedIp)

	return sharedIp, controlPlanes, nil
}

// Asks which networks may reach the Talos and Kubernetes APIs, suggesting the node's /24. They have to include the
// node, and the computer running bbe, or bbe would lock itself out of the node it's setting up
func askTrustedNetworks(helperService interfaces.HelperServiceInterface, ipFinderService interfaces.IpFinderServiceInterface, uiService interfaces.UiServiceInterface, nodeIp string) ([]string, error) {
	nodeAddress, err := netip.ParseAddr(nodeIp)
	if err != nil {
		return nil, err
	}
	suggestion := netip.PrefixFrom(nodeAddress, 24).Masked().String()

	localIp, err := ipFinderService.GetLocalIp(nodeIp)
	if err != nil {
		logger.Debug(fmt.Sprintf("Couldn't find this computer's address towards %s: %s", nodeIp, err))
	}

	title := fmt.Sprintf("Which networks may reach the Talos and Kubernetes APIs? Include every node and the computers you manage the cluster from, separated by commas %s", italic("(press enter to accept the node's network)"))
	for {
		answer, err := uiService.CreateInput(title, suggestion)
		if err != nil {
			panic(err)
		}

		networks, problem := parseTrustedNetworks(answer, nodeAddress, localIp, helperService.IsWsl())
		if problem == "" {
			return networks, nil
		}

		title = fmt.Sprintf("%s. Please enter the networks again:", problem)
	}
}

// Reads networks such as "192.168.1.0/24, 10.0.0.5/32", or says what's wrong with them
func parseTrustedNetworks(answer string, nodeAddress netip.Addr, localIp string, isWsl bool) ([]string, string) {
	prefixes := []netip.Prefix{}
	for _, field := range strings.Split(answer, ",") {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}

		prefix, err := netip.ParsePrefix(field)
		if err != nil {
			return nil, fmt.Sprintf("%s isn't a network such as 192.168.1.0/24", field)
		}
		prefixes = append(prefixes, prefix.Masked())
	}

	contains := func(address netip.Addr) bool {
		return slices.ContainsFunc(prefixes, func(prefix netip.Prefix) bool { return prefix.Contains(address) })
	}

	if len(prefixes) == 0 {
		return nil, "No network was entered"
	}

	if !contains(nodeAddress) {
		return nil, fmt.Sprintf("None of them includes the node's address, %s", nodeAddress)
	}

	if localAddress, err := netip.ParseAddr(localIp); err == nil && !contains(localAddress) {
		// WSL reaches the network through Windows, whose address the node sees instead
		if !isWsl {
			return nil, fmt.Sprintf("None of them includes this computer's address, %s, which would lock bbe out of the node", localAddress)
		}
		logger.Warning(fmt.Sprintf("This computer's address in WSL, %s, isn't in them. Make sure they include the Windows computer's address on your network, or bbe will be locked out of the node", localAddress))
	}

	networks := make([]string, len(prefixes))
	for i, prefix := range prefixes {
		networks[i] = prefix.String()
	}

	return networks, ""
}

// diskLabel describes a disk for the install prompt. The transport is included because the
// installer's own boot media - usually a USB stick - is listed alongside the real disks, and
// installing to it would overwrite the installer.
func diskLabel(disk models.TalosDisk) string {
	details := []string{disk.Spec.PrettySize}
	if disk.Spec.Model != "" {
		details = append(details, disk.Spec.Model)
	}
	if disk.Spec.Transport != "" {
		details = append(details, fmt.Sprintf("(%s)", disk.Spec.Transport))
	}

	return fmt.Sprintf("%s - %s", disk.Spec.DevPath, strings.Join(details, " "))
}

func imageCreation(helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface, imageService interfaces.ImageServiceInterface, workingDirectory string, nodeType models.NodeType) error {
	imageDirectory := fmt.Sprintf("%s/_out", workingDirectory)
	resultFilePath := fmt.Sprintf("%s/%s", imageDirectory, nodeType.OutputFile)
	spinner := spinner.New(spinner.CharSets[43], 100*time.Millisecond)

	_, imageExists := helperService.CheckIfFileExists(resultFilePath)
	if imageExists {
		result, err := uiService.CreateSelect(fmt.Sprintf("An image already exists in (%s) would you like to redownload it?", italic(resultFilePath)), []string{"Yes", "No"})
		if err != nil {
			panic(err)
		}

		if result == "No" {
			return nil
		}
	}

	logger.Debug("Creating image..")
	spinner.Start()

	result, err := imageService.CreateImage(nodeType, imageDirectory)
	if err != nil {
		return err
	}

	spinner.Stop()
	logger.Info(result)

	return nil
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
