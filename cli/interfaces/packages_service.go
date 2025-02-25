package interfaces

import "github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"

type PackageServiceInterface interface {
	GetAll() []models.Package
	InstallPackage(pkg models.Package, bbeConfig models.BbeConfig) error
	UninstallPackage(pkg models.Package, bbeConfig models.BbeConfig) error
}
