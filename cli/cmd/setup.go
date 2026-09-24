package cmd

import (
	"fmt"
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
	createControlPlane := answer == "Yes"

	configExists := configService.CheckForTalosConfigs(helperService)

	if createControlPlane && configExists {
		return fmt.Errorf("You are trying to create a control plane node, but there are already config files present, in %s", helperService.GetConfigDir())
	}

	if !configExists && !createControlPlane {
		return fmt.Errorf("No config files found while trying to enroll new node in existing cluster, please create your first node first")
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
	if createControlPlane {
		if !configExists {
			suggestedClusterName := "big_brain_entropy_holder"
			if rngError == nil {
				suggestedClusterName = codename.Generate(rng, 0)
			}

			clusterName, err = uiService.CreateInput("Please enter what you want to name your cluster", suggestedClusterName)
			if err != nil {
				panic(err)
			}

			err = talosService.GenerateConfig(helperService, chosenIp, clusterName, nodeType.TalosVersion)
			if err != nil {
				return fmt.Errorf("Error while generating config: %w", err)
			}
		}

		allowSchedulingOnControlPlanes, err = uiService.CreateSelect("Do you want to allow scheduling on the control plane? This is required if you have only one node.", []string{"Yes", "No"})
		if err != nil {
			panic(err)
		}

	}
	///////////////////////////////////////////////////////////////////////////////// QUESTIONS END ///////////////////////////////////////////////////////////////////////////////////////////////////////////////

	controlPlaneIp, err := talosService.GetControlPlaneIp(helperService, constants.ControlplaneConfigFile)
	if err != nil {
		return fmt.Errorf("Error while getting control plane IP: %w", err)
	}
	logger.Debug(fmt.Sprintf("Control plane IP: %s", controlPlaneIp))

	logger.Debug("Checking if talosctl matches the version the node image was built for")
	talosService.WarnIfTalosVersionMismatch(nodeType.TalosVersion)

	nodeConfigFile := constants.WorkerConfigFile
	if createControlPlane {
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

	logger.Debug("Joining cluster")
	err = talosService.JoinCluster(helperService, originalIp, nodeConfigFile)
	if err != nil {
		return fmt.Errorf("Error while joining cluster: %w", err)
	}

	if createControlPlane {
		err := talosService.BootstrapCluster(helperService, chosenIp, controlPlaneIp)
		if err != nil {
			return fmt.Errorf("Error while bootstrapping cluster: %w", err)
		}

		logger.Infof("Cluster bootstrapping successfully requested at %s", chosenIp)
	}

	err = talosService.VerifyNodeHealth(helperService, chosenIp, controlPlaneIp)
	if err != nil {
		return fmt.Errorf("Error while verifying node health: %w", err)
	}

	if createControlPlane {
		logger.Debug("Downloading kube config")
		err := talosService.DownloadKubeConfig(helperService, chosenIp, controlPlaneIp)
		if err != nil {
			return fmt.Errorf("Error while downloading kubeconfig: %w", err)
		}

		logger.Debug("Updating BBE cluster name")
		err = configService.UpdateBbeClusterName(helperService, clusterName)
		if err != nil {
			return fmt.Errorf("Error while updating BBE cluster name: %w", err)
		}

		logger.Infof("Control plane node %s successfully set up", chosenIp)
	} else {
		logger.Infof("Worker node %s successfully set up", chosenIp)
	}
	spinner.Stop()

	return nil
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
