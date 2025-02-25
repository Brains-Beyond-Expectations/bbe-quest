package cmd

import (
	"fmt"
	"os"
	"slices"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/config_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helper_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/package_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/ui_service"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install",
	Aliases: []string{"i"},
	Short:   "Install BBE packages",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) { // coverage-ignore
		helperService := helper_service.HelperService{}
		uiService := ui_service.UiService{}
		configService := config_service.ConfigService{}
		packageService := package_service.PackageService{}

		err := installCommand(helperService, uiService, configService, packageService)
		if err != nil {
			logger.Error("", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func installCommand(helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface) error {
	bbeConfig, err := configService.GetBbeConfig(helperService)
	if err != nil || bbeConfig.Bbe.Cluster.Name == "" {
		logger.Info("No BBE cluster found, please run 'bbe setup' to create your cluster")
		return nil
	}

	allPackages := packageService.GetAll()

	selectedIndexes, packageList := buildPackageIndex(allPackages, *bbeConfig)

	chosenPackages, err := uiService.CreateMultiChoose("Select packages to install", packageList, selectedIndexes)
	if err != nil { // coverage-ignore
		panic(err)
	}

	packagesToInstall, packagesToUninstall := diffPackages(allPackages, chosenPackages)

	updatedBbeConfig := *bbeConfig
	updatedBbeConfig.Bbe.Packages = bbeConfig.Bbe.Packages

	err = uninstallPackages(helperService, configService, packageService, updatedBbeConfig, packagesToUninstall)
	if err != nil {
		return fmt.Errorf("Failed to uninstall packages: %w", err)
	}

	err = installPackages(helperService, configService, packageService, updatedBbeConfig, packagesToInstall)
	if err != nil {
		return fmt.Errorf("Failed to install packages: %w", err)
	}

	return nil
}

func buildPackageIndex(allPackages []models.Package, bbeConfig models.BbeConfig) (selectedIndexes []int, packageList []string) {
	for packageIndex, pkg := range allPackages {
		packageList = append(packageList, pkg.Name)
		for _, installedPkg := range bbeConfig.Bbe.Packages {
			if installedPkg.Name == pkg.Name {
				selectedIndexes = append(selectedIndexes, packageIndex)
			}
		}
	}

	return selectedIndexes, packageList
}

func diffPackages(allPackages []models.Package, chosenPackages []string) (packagesToInstall []models.Package, packagesToUninstall []models.Package) {
	for _, pkg := range allPackages {
		found := false
		for _, chosenPackage := range chosenPackages {
			if pkg.Name == chosenPackage {
				found = true
				packagesToInstall = append(packagesToInstall, pkg)
			}
		}

		if !found {
			packagesToUninstall = append(packagesToUninstall, pkg)
		}
	}

	return packagesToInstall, packagesToUninstall
}

func uninstallPackages(helperService interfaces.HelperServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface, updatedBbeConfig models.BbeConfig, uninstalledPackages []models.Package) error {
	for _, pkg := range uninstalledPackages {
		for i, existingPkg := range updatedBbeConfig.Bbe.Packages {
			if existingPkg.Name == pkg.Name {
				err := packageService.UninstallPackage(pkg, updatedBbeConfig)
				if err != nil {
					logger.Error("Failed to uninstall package", err)
					continue
				}
				updatedBbeConfig.Bbe.Packages = slices.Delete(updatedBbeConfig.Bbe.Packages, i, i+1)
				break
			}
		}
	}
	err := configService.UpdateBbePackages(helperService, updatedBbeConfig.Bbe.Packages)
	if err != nil {
		return fmt.Errorf("Failed to update BBE configuration: %w", err)
	}

	return nil
}

func installPackages(helperService interfaces.HelperServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface, updatedBbeConfig models.BbeConfig, installedPackages []models.Package) error {
	for _, pkg := range installedPackages {
		err := packageService.InstallPackage(pkg, updatedBbeConfig)
		if err != nil {
			return fmt.Errorf("Failed to install package: %w", err)
		}

		found := false
		for i, existingPkg := range updatedBbeConfig.Bbe.Packages {
			if existingPkg.Name == pkg.Name {
				updatedBbeConfig.Bbe.Packages[i] = pkg
				found = true
				break
			}
		}
		if !found {
			updatedBbeConfig.Bbe.Packages = append(updatedBbeConfig.Bbe.Packages, pkg)
		}
	}
	err := configService.UpdateBbePackages(helperService, updatedBbeConfig.Bbe.Packages)
	if err != nil {
		return fmt.Errorf("Failed to update BBE configuration: %w", err)
	}

	return nil
}
