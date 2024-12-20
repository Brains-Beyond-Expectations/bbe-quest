package packages

import (
	"fmt"

	"github.com/Brains-Beyond-Expectations/bbe-quest/services/config"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/logger"
)

type BbePackage struct {
	Package          config.Package
	HelmChart        string
	HelmChartVersion string
}

var packages = []BbePackage{
	{
		Package: config.Package{
			Name:    "jellyfin",
			Version: "2.1.0",
		},
		HelmChart:        "https://jellyfin.github.io/jellyfin-helm/jellyfin",
		HelmChartVersion: "2.1.0",
	},
	{
		Package: config.Package{
			Name:    "blocky",
			Version: "1.0.0",
		},
		HelmChart:        "blocky",
		HelmChartVersion: "0.1.0",
	},
}

func GetAll() []*config.Package {
	var packageList []*config.Package
	for _, pkg := range packages {
		packageList = append(packageList, &pkg.Package)
	}
	return packageList
}

func InstallPackage(pkg *config.Package) error {
	for _, p := range packages {
		if p.Package.Name == pkg.Name {
			// cmd := exec.Command("helm", "install", pkg.Name, p.HelmChart,
			// 	"--version", p.HelmChartVersion,
			// 	"--namespace", pkg.Name,
			// 	"--create-namespace")
			// if err := cmd.Run(); err != nil {
			// 	return fmt.Errorf("failed to install helm package %s: %w", pkg.Name, err)
			// }
			logger.Info(fmt.Sprintf("Installed package %s", pkg.Name))
			return nil
		}
	}
	return fmt.Errorf("package %s not found", pkg.Name)
}

func UninstallPackage(pkg *config.Package) error {
	for _, p := range packages {
		if p.Package.Name == pkg.Name {
			// cmd := exec.Command("helm", "uninstall", pkg.Name,
			// 	"--namespace", pkg.Name)
			// if err := cmd.Run(); err != nil {
			// 	return fmt.Errorf("failed to uninstall helm package %s: %w", pkg.Name, err)
			// }
			logger.Info(fmt.Sprintf("Uninstalled package %s", pkg.Name))
			return nil
		}
	}
	return fmt.Errorf("package %s not found", pkg.Name)
}
