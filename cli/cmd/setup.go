package cmd

import (
	"fmt"
	"os"

	"github.com/Brains-Beyond-Expectations/bbe-quest/helper"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/config"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/dependencies"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/imagecreator"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/ipfinder"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/talos"
	"github.com/Brains-Beyond-Expectations/bbe-quest/ui"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{},
	Short:   "Guides you through a BBE-Quest node setup",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		bbeConfig, err := config.GetBbeConfig()
		if err != nil {
			bbeConfig = promptForConfigStorage()
		}

		if bbeConfig.Bbe.Storage.Type == "aws" {
			err := config.SyncConfigsWithAws(bbeConfig)
			if err != nil {
				logger.Error("Error while syncing config with AWS", err)
				os.Exit(1)
			}
		}

		// Get current local path
		workingDirectory, err := os.Getwd()
		if err != nil {
			logger.Error("Error while getting current working directory", err)
			os.Exit(1)
		}

		if !dependencies.VerifyDependencies() {
			logger.Error("Error while verifying dependencies", nil)
			os.Exit(1)
		}

		answer, err := ui.CreateSelect("Is this the first node in your cluster?", []string{"Yes", "No"})
		if err != nil {
			logger.Error("Error while creating select", err)
			os.Exit(1)
		}
		createControlPlane := answer == "Yes"

		configExists := config.CheckForTalosConfigs()
		if !configExists && !createControlPlane {
			logger.Error("No config files found while trying to enroll new node in exsting cluster, please create your first node first", nil)
			os.Exit(1)
		}

		answer, err = ui.CreateSelect("What type of device are you setting up?", []string{"Intel NUC", "Raspberry Pi"})
		if err != nil {
			logger.Error("Error while creating select", err)
			os.Exit(1)
		}

		nodeType := imagecreator.IntelNuc
		if answer == "Raspberry Pi" {
			nodeType = imagecreator.RaspberryPi
		}

		imageCreation(workingDirectory, nodeType)

		firstMessage := "Please use balenaEtcher to flash the .iso to your USB device"
		secondMessage := "Please insert the USB device into your new node and boot from it"

		if nodeType.ImagerType == "rpi_generic" {
			firstMessage = "Please use balenaEtcher to flash the .xz to your SD card"
			secondMessage = "Please insert the SD card into your new node and boot from it"
		}

		_, err = ui.CreateSelect(firstMessage, []string{"Done"})
		if err != nil {
			logger.Error("Error while creating select", err)
			os.Exit(1)
		}

		_, err = ui.CreateSelect(secondMessage, []string{"Done"})
		if err != nil {
			logger.Error("Error while creating select", err)
			os.Exit(1)
		}

		ips, err := ipfinder.LocateDevice()
		if err != nil {
			logger.Error("Error while attempting to locate device", err)
			os.Exit(1)
		}

		if len(ips) == 0 {
			logger.Info("No new Talos devices found")
			os.Exit(0)
		}

		if createControlPlane {
			if len(ips) > 1 {
				logger.Info("More than one device found, please only set up one device at a time when creating your first node")
				os.Exit(0)
			}

			if !configExists {
				clusterName, err := ui.CreateInput("Please enter what you want to name your cluster")
				if err != nil {
					logger.Error("Error while creating input", err)
					os.Exit(1)
				}

				err = talos.GenerateConfig(ips[0], clusterName)
				if err != nil {
					logger.Error("Error while generating config", err)
					os.Exit(1)
				}
			}
		}

		controlPlaneIp, err := talos.GetControlPlaneIp("controlplane.yaml")
		if err != nil {
			logger.Error("Error while getting control plane IP", err)
			os.Exit(1)
		}

		for _, ip := range ips {
			nodeConfigFile := "worker.yaml"
			if createControlPlane {
				nodeConfigFile = "controlplane.yaml"
			}

			disks, err := talos.GetDisks(ip)
			if err != nil {
				logger.Error("Error while getting disks", err)
				os.Exit(1)
			}

			disk, err := ui.CreateSelect(fmt.Sprintf("Please select the disk to install Talos on for %s", ip), disks)
			if err != nil {
				logger.Error("Error while creating select", err)
				os.Exit(1)
			}

			err = talos.ModifyConfigDisk(nodeConfigFile, disk)
			if err != nil {
				logger.Error("Error while modifying config disk", err)
				os.Exit(1)
			}

			err = talos.JoinCluster(ip, nodeConfigFile)
			if err != nil {
				logger.Error("Error while joining cluster", err)
				os.Exit(1)
			}

			if createControlPlane {
				err := talos.BootstrapCluster(ip, controlPlaneIp)
				if err != nil {
					logger.Error("Error while bootstrapping cluster", err)
					os.Exit(1)
				}

				logger.Infof("Cluster bootstrapping successfully requested at %s", ip)
			}

			err = talos.VerifyNodeHealth(ip, controlPlaneIp)
			if err != nil {
				logger.Error("Error while verifying node health", err)
				os.Exit(1)
			}

			if createControlPlane {
				err := talos.DownloadKubeConfig(ip, controlPlaneIp)
				if err != nil {
					logger.Error("Error while downloading kubeconfig", err)
					os.Exit(1)
				}

				logger.Infof("Control plane node %s successfully set up", ip)
			} else {
				logger.Infof("Worker node %s successfully set up", ip)
			}
		}
	},
}

func imageCreation(workingDirectory string, nodeType imagecreator.NodeType) {
	imageDirectory := fmt.Sprintf("%s/_out", workingDirectory)
	resultFilePath := fmt.Sprintf("%s/%s", imageDirectory, nodeType.OutputFile)

	_, imageExists := helper.CheckIfFileExists(resultFilePath)
	if imageExists {
		result, err := ui.CreateSelect("An image already exists, would you like to recreate it?", []string{"Yes", "No"})
		if err != nil {
			logger.Error("Error while creating select", err)
			os.Exit(1)
		}

		if result == "No" {
			return
		}
	}

	logger.Info("Creating image")
	result, err := imagecreator.CreateImage(nodeType, imageDirectory)
	if err != nil {
		os.Exit(1)
	}
	logger.Info(result)
}

func init() {
	rootCmd.AddCommand(setupCmd)
}
