package interfaces

import "github.com/Brains-Beyond-Expectations/bbe-quest/models"

type PackageServiceInterface interface {
	GetAll() []models.Package
	InstallPackage(pkg models.Package, bbeConfig models.BbeConfig) error
	UpgradePackage(pkg models.Package, bbeConfig models.BbeConfig) error
	UninstallPackage(pkg models.Package, bbeConfig models.BbeConfig) error
}
