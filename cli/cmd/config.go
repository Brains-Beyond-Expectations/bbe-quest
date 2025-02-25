package cmd

import (
	"fmt"
	"os"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/config_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helper_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/ui_service"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:     "config",
	Aliases: []string{"c"},
	Short:   "Setup your BBE-Quest configuration",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) { // coverage-ignore
		helperService := helper_service.HelperService{}
		uiService := ui_service.UiService{}
		configService := config_service.ConfigService{}

		err := configCommand(&helperService, uiService, configService)
		if err != nil {
			logger.Error("", err)
			os.Exit(1)
		}
	},
}

func configCommand(helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface, configService interfaces.ConfigServiceInterface) error {
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

	return nil
}

func getOrGenerateConfig(helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface, configService interfaces.ConfigServiceInterface) (*models.BbeConfig, error) {
	choice, err := uiService.CreateSelect("No BBE configuration file found, where would you like to store your config files?", []string{"Local", "AWS"})
	if err != nil { // coverage-ignore
		panic(err)
	}

	switch choice {
	case "Local":
		err := configService.GenerateBbeConfig(helperService, "local")
		if err != nil {
			return nil, err
		}
	case "AWS":
		err := configService.GenerateBbeConfig(helperService, "aws")
		if err != nil {
			return nil, err
		}
	}

	bbeConfig, err := configService.GetBbeConfig(helperService)
	if err != nil {
		return nil, err
	}

	return bbeConfig, nil
}

func init() {
	rootCmd.AddCommand(configCmd)
}
