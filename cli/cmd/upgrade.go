package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/config_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helm_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helper_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/package_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/ui_service"
	"github.com/spf13/cobra"
)

var upgradeCmd = &cobra.Command{
	Use:     "upgrade",
	Aliases: []string{"u"},
	Short:   "Upgrade BBE packages",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		helperService := helper_service.HelperService{}
		uiService := ui_service.UiService{}
		configService := config_service.ConfigService{}
		packageService := package_service.PackageService{}
		helmService := helm_service.HelmService{}

		uninteractive, _ := cmd.Flags().GetBool("yes")

		err := upgradeCommand(helperService, uiService, configService, packageService, helmService, uninteractive)
		if err != nil {
			logger.Error("", err)
			os.Exit(1)
		}
	},
}

func upgradeCommand(helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface, helmService interfaces.HelmServiceInterface, uninteractive bool) error {
	bbeConfig, err := configService.GetBbeConfig(helperService)
	if err != nil || bbeConfig.Bbe.Cluster.Name == "" {
		logger.Info("No BBE cluster found, please run 'bbe setup' to create your cluster")
		return errors.New("No BBE cluster found, please run 'bbe setup' to create your cluster")
	}

	installedBundles := bbeConfig.Bbe.Bundles
	allBundles := packageService.GetAllBundles()

	defer func() {
		err := configService.UpdateBbeBundles(helperService, installedBundles)
		if err != nil {
			logger.Error("Failed to update BBE bundles", err)
		}
	}()

	// Go through our currently installed bundles and see if a newer version is available
	for i, installedBundle := range installedBundles {
		logger.Info(fmt.Sprintf("Checking for newer version of bundle %s...", installedBundle.Name))
		for _, bundle := range allBundles {
			if installedBundle.Name == bundle.Name {
				if installedBundle.Version != bundle.Version {
					upgrade := uninteractive
					if !uninteractive {
						result, err := uiService.CreateSelect(fmt.Sprintf("Bundle %s has newer version %s available. Do you want to upgrade?", bundle.Name, bundle.Version), []string{"Yes", "No"})
						if err != nil {
							return err
						}

						upgrade = result == "Yes"
					}
					if upgrade {
						err := packageService.UpgradeBundle(bundle, *bbeConfig, helmService)
						if err != nil {
							return err
						}
						installedBundles[i].Version = bundle.Version
					}
				}
				break
			}
		}
	}

	logger.Info("All bundle checked")

	return nil
}

func init() {
	rootCmd.AddCommand(upgradeCmd)

	upgradeCmd.PersistentFlags().BoolP("yes", "y", false, "Automatically accept yes/no questions without input.")
}
