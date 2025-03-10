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

func (packageService PackageService) GetAll() []models.Package {
	var packageList []models.Package
	for _, pkg := range packages {
		packageList = append(packageList, pkg.Package)
	}
	return packageList
}

func (packageService PackageService) InstallPackage(pkg models.Package, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	for _, p := range packages {
		if p.Package.Name == pkg.Name {
			if !helmService.IsPackageInstalled(pkg.Name, pkg.Name, bbeConfig.Bbe.Cluster.Context) {
				response := helmService.AddRepo(p.PackageRepository.Name, p.PackageRepository.RepositoryUrl)

				if response != nil {
					return response
				}

				return helmService.InstallChart(pkg.Name, p.HelmChart, p.PackageRepository.Name, pkg.Version, pkg.Name, bbeConfig.Bbe.Cluster.Context)
			}

			return nil
		}
	}

	return fmt.Errorf("Package `%s` not found", pkg.Name)
}

func (packageService PackageService) UpgradePackage(pkg models.Package, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	for _, p := range packages {
		if p.Package.Name == pkg.Name {
			response := helmService.AddRepo(p.PackageRepository.Name, p.PackageRepository.RepositoryUrl)

			if response != nil {
				return response
			}

			return helmService.UpgradeChart(pkg.Name, p.HelmChart, p.PackageRepository.Name, p.HelmChartVersion, pkg.Name, bbeConfig.Bbe.Cluster.Context)
		}
	}

	return fmt.Errorf("Package `%s` not found", pkg.Name)
}

func (packageService PackageService) UninstallPackage(pkg models.Package, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	for _, p := range packages {
		if p.Package.Name == pkg.Name {
			return helmService.UninstallChart(pkg.Name, pkg.Name, bbeConfig.Bbe.Cluster.Context)
		}
	}

	return fmt.Errorf("Package `%s` not found", pkg.Name)
}
