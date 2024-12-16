package cmd

import (
	"fmt"
	"os"

	"github.com/nicolajv/bbe-quest/helper"
	"github.com/nicolajv/bbe-quest/services/config"
	"github.com/nicolajv/bbe-quest/services/dependencies"
	"github.com/nicolajv/bbe-quest/services/ipfinder"
	"github.com/nicolajv/bbe-quest/services/isocreator"
	"github.com/nicolajv/bbe-quest/services/talos"
	"github.com/nicolajv/bbe-quest/ui"
	"github.com/sirupsen/logrus"
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

		if bbeConfig.Bbe.Storage == "aws" {
			err := config.SyncConfigsWithAws()
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Error while syncing config with AWS")
				os.Exit(1)
			}
		}

		// Get current local path
		workingDirectory, err := os.Getwd()
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("Error while getting current working directory")
			os.Exit(1)
		}

		if !dependencies.VerifyDependencies() {
			logrus.Error("Please install missing dependencies before continuing")
			os.Exit(1)
		}

		answer, err := ui.CreateSelect("Is this the first node in your cluster?", []string{"Yes", "No"})
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("Error while creating select")
			os.Exit(1)
		}
		createControlPlane := answer == "Yes"

		configExists := config.CheckForTalosConfigs()
		if !configExists && !createControlPlane {
			logrus.Error("No config files found while trying to enroll new node in exsting cluster, please create your first node first")
			os.Exit(1)
		}

		isoCreation(workingDirectory)

		_, err = ui.CreateSelect("Please use balenaEtcher to flash the ISO to USB", []string{"Done"})
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("Error while creating select")
			os.Exit(1)
		}

		_, err = ui.CreateSelect("Please insert the USB device into your new node and boot from it", []string{"Done"})
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("Error while creating select")
			os.Exit(1)
		}

		ips, err := ipfinder.LocateDevice()
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("Error while locating device")
			os.Exit(1)
		}

		if len(ips) == 0 {
			logrus.Info("No new Talos devices found")
			os.Exit(0)
		}

		if createControlPlane {
			if len(ips) > 1 {
				logrus.Info("More than one device found, please only set up one device at a time when creating your first node")
				os.Exit(0)
			}

			if !configExists {
				clusterName, err := ui.CreateInput("Please enter what you want to name your cluster")
				if err != nil {
					logrus.WithFields(logrus.Fields{"error": err}).Error("Error while creating input")
					os.Exit(1)
				}

				err = talos.GenerateConfig(ips[0], clusterName)
				if err != nil {
					logrus.WithFields(logrus.Fields{"error": err}).Error("Error while generating config")
					os.Exit(1)
				}
			}
		}

		controlPlaneIp, err := talos.GetControlPlaneIp("controlplane.yaml")
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("Error while getting control plane IP")
			os.Exit(1)
		}

		for _, ip := range ips {
			nodeConfigFile := "worker.yaml"
			if createControlPlane {
				nodeConfigFile = "controlplane.yaml"
			}

			disks, err := talos.GetDisks(ip)
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Error while getting disks")
				os.Exit(1)
			}

			disk, err := ui.CreateSelect(fmt.Sprintf("Please select the disk to install Talos on for %s", ip), disks)
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Error while creating select")
				os.Exit(1)
			}

			err = talos.ModifyConfigDisk(nodeConfigFile, disk)
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Error while modifying config disk")
				os.Exit(1)
			}

			err = talos.JoinCluster(ip, nodeConfigFile)
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Error while joining cluster")
				os.Exit(1)
			}

			if createControlPlane {
				err := talos.BootstrapCluster(ip, controlPlaneIp)
				if err != nil {
					logrus.WithFields(logrus.Fields{"error": err}).Error("Error while bootstrapping cluster")
					os.Exit(1)
				}

				logrus.Infof("Cluster bootstrapping successfully requested at %s", ip)
			}

			err = talos.VerifyNodeHealth(ip, controlPlaneIp)
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Error while verifying node health")
				os.Exit(1)
			}

			if createControlPlane {
				err := talos.DownloadKubeConfig(ip, controlPlaneIp)
				if err != nil {
					logrus.WithFields(logrus.Fields{"error": err}).Error("Error while downloading kubeconfig")
					os.Exit(1)
				}

				logrus.Infof("Control plane node %s successfully set up", ip)
			} else {
				logrus.Infof("Worker node %s successfully set up", ip)
			}
		}
	},
}

func isoCreation(workingDirectory string) {
	isoDirectory := fmt.Sprintf("%s/_out", workingDirectory)

	createIso := true

	_, isoExists := helper.CheckIfFileExists(fmt.Sprintf("%s/metal-amd64.iso", isoDirectory))
	if isoExists {
		result, err := ui.CreateSelect("An ISO already exists, would you like to recreate it?", []string{"Yes", "No"})
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("Error while creating select")
			os.Exit(1)
		}

		if result == "No" {
			createIso = false
		}
	}

	if createIso {
		logrus.Info("Creating new ISO")
		result, err := isocreator.CreateIso(isoDirectory, []string{"intel-ucode", "gvisor", "iscsi-tools"})
		if err != nil {
			os.Exit(1)
		}
		logrus.Info(result)
	}

}

func init() {
	rootCmd.AddCommand(setupCmd)
}
