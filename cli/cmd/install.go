package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/config_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helm_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helper_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/package_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/ui_service"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install",
	Aliases: []string{"i"},
	Short:   "Install BBE bundles",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		helperService := helper_service.HelperService{}
		uiService := ui_service.UiService{}
		configService := config_service.ConfigService{}
		packageService := package_service.PackageService{}
		helmService := helm_service.HelmService{}

		err := installCommand(helperService, uiService, configService, packageService, helmService)
		if err != nil {
			logger.Error("", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func installCommand(helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface, helmService interfaces.HelmServiceInterface) error {
	bbeConfig, err := configService.GetBbeConfig(helperService)
	if err != nil || bbeConfig.Bbe.Cluster.Name == "" {
		logger.Info("No BBE cluster found, please run 'bbe setup' to create your cluster")
		return nil
	}

	allBundles := packageService.GetAllBundles()

	selectedIndexes, bundleList := buildBundleIndex(allBundles, *bbeConfig)

	chosenBundles, err := uiService.CreateMultiChoose("Select bundles to install", bundleList, selectedIndexes)
	if err != nil {
		panic(err)
	}

	bundlesToInstall, bundlesToUninstall := diffBundles(allBundles, chosenBundles)

	updatedBbeConfig := *bbeConfig
	updatedBbeConfig.Bbe.Bundles = bbeConfig.Bbe.Bundles

	err = uninstallBundles(helperService, configService, packageService, helmService, updatedBbeConfig, bundlesToUninstall)
	if err != nil {
		return fmt.Errorf("Failed to uninstall bundles: %w", err)
	}

	err = installBundles(helperService, configService, packageService, helmService, updatedBbeConfig, bundlesToInstall)
	if err != nil {
		return fmt.Errorf("Failed to install bundles: %w", err)
	}

	return nil
}

func buildBundleIndex(allBundles []models.BbeBundle, bbeConfig models.BbeConfig) (selectedIndexes []int, bundleList []string) {
	for bundleIndex, bundle := range allBundles {
		bundleList = append(bundleList, bundle.Name)
		for _, installedBundle := range bbeConfig.Bbe.Bundles {
			if installedBundle.Name == bundle.Name {
				selectedIndexes = append(selectedIndexes, bundleIndex)
			}
		}
	}

	return selectedIndexes, bundleList
}

func diffBundles(allBundles []models.BbeBundle, chosenBundles []string) (bundlesToInstall []models.BbeBundle, bundlesToUninstall []models.BbeBundle) {
	for _, pkg := range allBundles {
		found := false
		for _, chosenBundle := range chosenBundles {
			if pkg.Name == chosenBundle {
				found = true
				bundlesToInstall = append(bundlesToInstall, pkg)
			}
		}

		if !found {
			bundlesToUninstall = append(bundlesToUninstall, pkg)
		}
	}

	return bundlesToInstall, bundlesToUninstall
}

func uninstallBundles(helperService interfaces.HelperServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface, helmService interfaces.HelmServiceInterface, updatedBbeConfig models.BbeConfig, uninstalledBundles []models.BbeBundle) error {
	for _, bundle := range uninstalledBundles {
		for i, existingPkg := range updatedBbeConfig.Bbe.Bundles {
			if existingPkg.Name == bundle.Name {
				err := packageService.UninstallBundle(bundle, updatedBbeConfig, helmService)
				if err != nil {
					logger.Error("Failed to uninstall bundle", err)
					continue
				}
				updatedBbeConfig.Bbe.Bundles = slices.Delete(updatedBbeConfig.Bbe.Bundles, i, i+1)
				break
			}
		}
	}
	err := configService.UpdateBbeBundles(helperService, updatedBbeConfig.Bbe.Bundles)
	if err != nil {
		return fmt.Errorf("Failed to update BBE configuration: %w", err)
	}

	return nil
}

func installBundles(helperService interfaces.HelperServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface, helmService interfaces.HelmServiceInterface, updatedBbeConfig models.BbeConfig, installedBundles []models.BbeBundle) error {
	for _, bundle := range installedBundles {
		err := packageService.InstallBundle(bundle, updatedBbeConfig, helmService)
		if err != nil {
			return fmt.Errorf("Failed to install bundle: %w", err)
		}

		found := false
		for i, existingBundle := range updatedBbeConfig.Bbe.Bundles {
			if existingBundle.Name == bundle.Name {
				updatedBbeConfig.Bbe.Bundles[i] = bundle
				found = true
				break
			}
		}
		if !found {
			updatedBbeConfig.Bbe.Bundles = append(updatedBbeConfig.Bbe.Bundles, bundle)
		}
	}
	err := configService.UpdateBbeBundles(helperService, updatedBbeConfig.Bbe.Bundles)
	if err != nil {
		return fmt.Errorf("Failed to update BBE configuration: %w", err)
	}

	return nil
}
