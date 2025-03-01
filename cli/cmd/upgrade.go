package cmd

import (
	"fmt"
	"os"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/config_service"
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

		uninteractive, _ := cmd.Flags().GetBool("yes")

		err := upgradeCommand(helperService, uiService, configService, packageService, uninteractive)
		if err != nil {
			logger.Error("", err)
			os.Exit(1)
		}
	},
}

func upgradeCommand(helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface, uninteractive bool) error {
	bbeConfig, err := configService.GetBbeConfig(helperService)
	if err != nil || bbeConfig.Bbe.Cluster.Name == "" {
		logger.Info("No BBE cluster found, please run 'bbe setup' to create your cluster")
		return nil
	}

	installedPackages := bbeConfig.Bbe.Packages
	allPackages := packageService.GetAll()

	defer func() {
		err := configService.UpdateBbePackages(helperService, installedPackages)
		if err != nil {
			logger.Error("Failed to update BBE packages", err)
		}
	}()

	// Go through our currently installed packages and see if a newer version is available
	for i, installedPackage := range installedPackages {
		logger.Info(fmt.Sprintf("Checking for newer version of package %s...", installedPackage.Name))
		for _, pkg := range allPackages {
			if installedPackage.Name == pkg.Name {
				if installedPackage.Version != pkg.Version {
					upgrade := uninteractive
					if !uninteractive {
						result, err := uiService.CreateSelect(fmt.Sprintf("Package %s has newer version %s available. Do you want to upgrade?", pkg.Name, pkg.Version), []string{"Yes", "No"})
						if err != nil {
							return err
						}

						upgrade = result == "Yes"
					}
					if upgrade {
						err := packageService.UpgradePackage(pkg, *bbeConfig)
						if err != nil {
							return err
						}
						installedPackages[i].Version = pkg.Version
					}
				}
				break
			}
		}
	}

	logger.Info("All packages checked")

	return nil
}

func init() {
	rootCmd.AddCommand(upgradeCmd)

	upgradeCmd.PersistentFlags().BoolP("yes", "y", false, "Automatically accept yes/no questions without input.")
}
