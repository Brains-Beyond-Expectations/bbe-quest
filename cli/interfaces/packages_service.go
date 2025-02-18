package interfaces

import "github.com/Brains-Beyond-Expectations/bbe-quest/models"

type PackageServiceInterface interface {
	GetAll() []models.Package
	InstallPackage(pkg models.Package) error
	UninstallPackage(pkg models.Package) error
}
