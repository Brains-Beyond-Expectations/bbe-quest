package cmd

import (
	"os"
	"slices"

	"github.com/Brains-Beyond-Expectations/bbe-quest/services/config"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/packages"
	"github.com/Brains-Beyond-Expectations/bbe-quest/ui"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install",
	Aliases: []string{"v"},
	Short:   "Install BBE packages",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		bbeConfig, err := config.GetBbeConfig()
		if err != nil || bbeConfig.Bbe.Cluster.Name == "" {
			logger.Info("No BBE configuration file found, please run 'bbe setup' to create your cluster")
			os.Exit(0)
		}

		selectedIndexes := []int{}
		packageList := []string{}

		allPackages := packages.GetAll()
		for packageIndex, pkg := range allPackages {
			packageList = append(packageList, pkg.Name)
			for _, installedPkg := range bbeConfig.Bbe.Packages.Package {
				if installedPkg.Name == pkg.Name {
					selectedIndexes = append(selectedIndexes, packageIndex)
				}
			}
		}

		chosenPackages, err := ui.CreateMultiChoose("Select packages to install", packageList, selectedIndexes)
		if err != nil {
			logger.Error("Failed to select packages to install", err)
			os.Exit(1)
		}

		installedPackages := []*config.Package{}
		uninstalledPackages := []*config.Package{}

		for _, pkg := range allPackages {
			found := false
			for _, chosenPackage := range chosenPackages {
				if pkg.Name == chosenPackage {
					found = true
					installedPackages = append(installedPackages, pkg)
				}
			}

			if !found {
				uninstalledPackages = append(uninstalledPackages, pkg)
			}
		}

		updatedBbeConfig := &config.BbeConfig{
			Bbe: config.Bbe{
				Packages: bbeConfig.Bbe.Packages,
			},
		}

		for _, pkg := range uninstalledPackages {
			for i, existingPkg := range updatedBbeConfig.Bbe.Packages.Package {
				if existingPkg.Name == pkg.Name {
					packages.UninstallPackage(pkg)
					updatedBbeConfig.Bbe.Packages.Package = slices.Delete(updatedBbeConfig.Bbe.Packages.Package, i, i+1)
					break
				}
			}
			_, err := config.UpdateBbeConfig(updatedBbeConfig)
			if err != nil {
				logger.Error("Failed to update BBE configuration", err)
				os.Exit(1)
			}
		}

		for _, pkg := range installedPackages {
			err := packages.InstallPackage(pkg)
			if err != nil {
				logger.Error("Failed to install package", err)
				os.Exit(1)
			}

			found := false
			for i, existingPkg := range updatedBbeConfig.Bbe.Packages.Package {
				if existingPkg.Name == pkg.Name {
					updatedBbeConfig.Bbe.Packages.Package[i] = pkg
					found = true
					break
				}
			}
			if !found {
				updatedBbeConfig.Bbe.Packages.Package = append(updatedBbeConfig.Bbe.Packages.Package, pkg)
			}
			_, err = config.UpdateBbeConfig(updatedBbeConfig)
			if err != nil {
				logger.Error("Failed to update BBE configuration", err)
				os.Exit(1)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
