package package_service

import (
	"fmt"
	"os/exec"

	"github.com/Brains-Beyond-Expectations/bbe-quest/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/models"
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
}

var execCommand = exec.Command

type PackagesServiceInterface interface {
	GetAll() []models.Package
	InstallPackage(pkg models.Package) error
	UninstallPackage(pkg models.Package) error
}

type PackagesService struct{}

func (packageService PackageService) GetAll() []models.Package {
	var packageList []models.Package
	for _, pkg := range packages {
		packageList = append(packageList, pkg.Package)
	}
	return packageList
}

func (packageService PackageService) InstallPackage(pkg models.Package) error {
	for _, p := range packages {
		if p.Package.Name == pkg.Name {
			cmd := execCommand("helm", "repo", "add", p.PackageRepository.Name, p.PackageRepository.RepositoryUrl)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to add helm repository %s: %w", p.PackageRepository.Name, err)
			}

			cmd = execCommand("helm", "install", pkg.Name, fmt.Sprintf("%s/%s", p.PackageRepository.Name, p.HelmChart),
				"--version", p.HelmChartVersion,
				"--namespace", pkg.Name,
				"--create-namespace")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to install helm package %s: %w", pkg.Name, err)
			}
			logger.Info(fmt.Sprintf("Installed package %s", pkg.Name))
			return nil
		}
	}
	return fmt.Errorf("package %s not found", pkg.Name)
}

func (packageService PackageService) UninstallPackage(pkg models.Package) error {
	for _, p := range packages {
		if p.Package.Name == pkg.Name {
			cmd := execCommand("helm", "uninstall", pkg.Name,
				"--namespace", pkg.Name)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to uninstall helm package %s: %w", pkg.Name, err)
			}
			logger.Info(fmt.Sprintf("Uninstalled package %s", pkg.Name))
			return nil
		}
	}
	return fmt.Errorf("package %s not found", pkg.Name)
}
