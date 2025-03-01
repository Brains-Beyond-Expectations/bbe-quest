package interfaces

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
)

type PackageServiceInterface interface {
	GetAll() []models.Package
	InstallPackage(pkg models.Package, bbeConfig models.BbeConfig, helmService HelmServiceInterface) error
	UpgradePackage(pkg models.Package, bbeConfig models.BbeConfig, helmService HelmServiceInterface) error
	UninstallPackage(pkg models.Package, bbeConfig models.BbeConfig, helmService HelmServiceInterface) error
}
