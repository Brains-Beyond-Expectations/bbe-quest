package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/config_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helm_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helper_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/package_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/prerequisite_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/talos_service"
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
		talosService := talos_service.TalosService{}
		prerequisiteService := prerequisite_service.PrerequisiteService{}

		uninteractive, _ := cmd.Flags().GetBool("yes")

		err := upgradeCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService, uninteractive)
		if err != nil {
			logger.Error("", err)
			os.Exit(1)
		}
	},
}

func upgradeCommand(helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface, helmService interfaces.HelmServiceInterface, talosService interfaces.TalosServiceInterface, prerequisiteService interfaces.PrerequisiteServiceInterface, uninteractive bool) error {
	bbeConfig, err := configService.GetBbeConfig(helperService)
	if err != nil || bbeConfig.Bbe.Cluster.Name == "" {
		logger.Info("No BBE cluster found, please run 'bbe setup' to create your cluster")
		return errors.New("No BBE cluster found, please run 'bbe setup' to create your cluster")
	}

	installedPackages := bbeConfig.Bbe.Packages
	allPackages, err := packageService.GetAll()
	if err != nil {
		return err
	}

	// A newer version can need something the installed one didn't, such as a package it now relies on
	runner := &prerequisiteRunner{
		helperService:       helperService,
		uiService:           uiService,
		packageService:      packageService,
		helmService:         helmService,
		talosService:        talosService,
		prerequisiteService: prerequisiteService,
		bbeConfig:           *bbeConfig,
		allPackages:         allPackages,
		interactive:         !uninteractive,
		install: func(pkg models.ChartEntry) error {
			values, err := packageService.InstallPackage(pkg, nil, *bbeConfig, helmService, uiService)
			if err != nil {
				return err
			}
			installedPackages = append(installedPackages, models.LocalPackage{Name: pkg.Name, Version: pkg.Version, Values: values})
			return nil
		},
	}

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
						if err := runner.ensure(pkg); err != nil {
							return err
						}
						values, err := packageService.UpgradePackage(pkg, installedPackage.Values, *bbeConfig, helmService, uiService, !uninteractive)
						if err != nil {
							return err
						}
						installedPackages[i].Version = pkg.Version
						installedPackages[i].Values = values
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
