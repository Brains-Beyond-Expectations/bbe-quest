package package_service

import (
	"fmt"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
)

type PackageService struct{}

var bbeRepository = models.BbePackageRepository{
	Name:          "bbe",
	RepositoryUrl: "https://brains-beyond-expectations.github.io/bbe-charts",
}

func (packageService PackageService) GetAllBundles() []models.BbeBundle {
	return packageService.getLibrary()
}

func (packageService PackageService) getLibrary() []models.BbeBundle {
	var packages = []models.BbePackage{
		{
			Package: models.Package{
				Name:    "blocky",
				Version: "0.1.3",
			},
			HelmChart:         "blocky",
			HelmChartVersion:  "0.1.3",
			PackageRepository: bbeRepository,
		},
		{
			Package: models.Package{
				Name:    "ingress-nginx",
				Version: "4.12.0",
			},
			HelmChart:        "ingress-nginx",
			HelmChartVersion: "4.12.0",
			PackageRepository: models.BbePackageRepository{
				Name:          "ingress-nginx",
				RepositoryUrl: "https://kubernetes.github.io/ingress-nginx",
			},
		},
	}

	var bundles = []models.BbeBundle{
		{
			Name:        "network",
			BbePackages: packages,
			Version:     "0.1.0",
		},
	}

	return bundles
}

func (packageService PackageService) InstallBundle(bundle models.BbeBundle, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	for _, pkg := range bundle.BbePackages {
		err := packageService.installPackage(pkg, bbeConfig, helmService)
		if err != nil {
			return err
		}
	}

	return nil
}

func (packageService PackageService) UninstallBundle(bundle models.BbeBundle, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	for _, pkg := range bundle.BbePackages {
		err := packageService.uninstallPackage(pkg, bbeConfig, helmService)
		if err != nil {
			return err
		}
	}

	return nil
}

func (packageService PackageService) UpgradeBundle(bundle models.BbeBundle, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	for _, pkg := range bundle.BbePackages {
		err := packageService.upgradePackage(pkg, bbeConfig, helmService)
		if err != nil {
			return err
		}
	}

	return nil
}

func (packageService PackageService) installPackage(pkg models.BbePackage, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	if !helmService.IsPackageInstalled(pkg.Package.Name, pkg.Package.Name, bbeConfig.Bbe.Cluster.Context) {
		response := helmService.AddRepo(pkg.PackageRepository.Name, pkg.PackageRepository.RepositoryUrl)

		if response != nil {
			return response
		}

		return helmService.InstallChart(pkg.Package.Name, pkg.HelmChart, pkg.HelmChart, pkg.HelmChartVersion, pkg.Package.Name, bbeConfig.Bbe.Cluster.Context)
	}

	return fmt.Errorf("Package `%s` is already installed", pkg.Package.Name)
}

func (packageService PackageService) upgradePackage(pkg models.BbePackage, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	if !helmService.IsPackageInstalled(pkg.Package.Name, pkg.Package.Name, bbeConfig.Bbe.Cluster.Context) {
		return fmt.Errorf("Package `%s` is not installed", pkg.Package.Name)
	}

	return helmService.UpgradeChart(pkg.Package.Name, pkg.HelmChart, pkg.HelmChart, pkg.HelmChartVersion, pkg.Package.Name, bbeConfig.Bbe.Cluster.Context)

}

func (packageService PackageService) uninstallPackage(pkg models.BbePackage, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	return helmService.UninstallChart(pkg.Package.Name, pkg.Package.Name, bbeConfig.Bbe.Cluster.Context)
}
