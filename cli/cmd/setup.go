package cmd

import (
	"fmt"
	"os"

	"github.com/nicolajv/bbe-quest/services"
	"github.com/nicolajv/bbe-quest/ui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:     "setup",
	Aliases: []string{},
	Short:   "Guides you through the first time BBE-Quest setup",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		// Get current local path
		workingDirectory, err := os.Getwd()
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("Error while getting current working directory")
			os.Exit(1)
		}

		isoCreation(workingDirectory)
	},
}

func isoCreation(workingDirectory string) {
	isoDirectory := fmt.Sprintf("%s/_out", workingDirectory)

	createIso := true

	isoExists := services.CheckIfIsoExists(isoDirectory)
	if isoExists {
		result, err := ui.CreateModal("An ISO already exists, would you like to recreate it?", []string{"Yes", "No"})
		if err != nil {
			logrus.WithFields(logrus.Fields{"error": err}).Error("Error while creating modal")
			os.Exit(1)
		}

		if result == "No" {
			createIso = false
		}
	}

	if createIso {
		logrus.Info("Creating new ISO")
		result, err := services.CreateIso(isoDirectory, []string{"intel-ucode", "gvisor", "iscsi-tools"})
		if err != nil {
			os.Exit(1)
		}
		fmt.Println(result)
	}

}

func init() {
	rootCmd.AddCommand(setupCmd)
}
