package cmd

import (
	"os"

	"github.com/nicolajv/bbe-quest/services/config"
	"github.com/nicolajv/bbe-quest/ui"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"c"},
	Short:   "Setup your BBE-Quest configuration",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		bbeConfig, err := config.GetBbeConfig()
		if err != nil {
			choice, err := ui.CreateSelect("No configuration found, where would you like to store your config files?", []string{"Local", "AWS"})
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Error while creating select")
				os.Exit(1)
			}

			switch choice {
			case "Local":
				config.GenerateBbeConfig("local")
			case "AWS":
				config.GenerateBbeConfig("aws")
				if err != nil {
					logrus.WithFields(logrus.Fields{"error": err}).Error("Error while downloading config from AWS")
					os.Exit(1)
				}
			}

			bbeConfig, err = config.GetBbeConfig()
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Error while reading config file")
				os.Exit(1)
			}
		}

		if bbeConfig.Bbe.Storage == "aws" {
			err := config.SyncConfigsWithAws()
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Error while syncing config with AWS")
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
}
