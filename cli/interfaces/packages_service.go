package interfaces

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
)

type PackageServiceInterface interface {
	GetAll() ([]models.ChartEntry, error)
	InstallPackage(chart models.ChartEntry, bbeConfig models.BbeConfig, helmService HelmServiceInterface, uiService UiServiceInterface, helperService HelperServiceInterface) error
	UpgradePackage(chart models.ChartEntry, bbeConfig models.BbeConfig, helmService HelmServiceInterface, uiService UiServiceInterface, helperService HelperServiceInterface) error
	UninstallPackage(chart models.LocalPackage, bbeConfig models.BbeConfig, helmService HelmServiceInterface) error
}
