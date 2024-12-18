package cmd

import (
	"os"

	"github.com/nicolajv/bbe-quest/services/config"
	"github.com/nicolajv/bbe-quest/services/logger"
	"github.com/nicolajv/bbe-quest/ui"
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
			bbeConfig = promptForConfigStorage()
		}

		if bbeConfig.Bbe.Storage.Type == "aws" {
			err := config.SyncConfigsWithAws(bbeConfig)
			if err != nil {
				logger.Error("Error while syncing config with AWS", err)
				os.Exit(1)
			}
		}
	},
}

func promptForConfigStorage() *config.BbeConfig {
	choice, err := ui.CreateSelect("No BBE configuration file found, where would you like to store your config files?", []string{"Local", "AWS"})
	if err != nil {
		logger.Error("Error while creating select", err)
		os.Exit(1)
	}

	switch choice {
	case "Local":
		err := config.GenerateBbeConfig("local")
		if err != nil {
			logger.Error("Error while generating BBE config", err)
			os.Exit(1)
		}
	case "AWS":
		err := config.GenerateBbeConfig("aws")
		if err != nil {
			logger.Error("Error while generating BBE config", err)
			os.Exit(1)
		}
	}

	bbeConfig, err := config.GetBbeConfig()
	if err != nil {
		logger.Error("Error while getting BBE config", err)
		os.Exit(1)
	}

	return bbeConfig
}

func init() {
	rootCmd.AddCommand(configCmd)
}
