package interfaces

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
)

type PackageServiceInterface interface {
	GetAllBundles() []models.BbeBundle
	InstallBundle(bundle models.BbeBundle, bbeConfig models.BbeConfig, helmService HelmServiceInterface) error
	UninstallBundle(bundle models.BbeBundle, bbeConfig models.BbeConfig, helmService HelmServiceInterface) error
	UpgradeBundle(bundle models.BbeBundle, bbeConfig models.BbeConfig, helmService HelmServiceInterface) error
}
