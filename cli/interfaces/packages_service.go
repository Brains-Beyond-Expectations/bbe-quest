package interfaces

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
)

type PackageServiceInterface interface {
	GetAll() ([]models.ChartEntry, error)
	InstallPackage(chart models.ChartEntry, values map[string]interface{}, bbeConfig models.BbeConfig, helmService HelmServiceInterface, uiService UiServiceInterface) (map[string]interface{}, error)
	UpgradePackage(chart models.ChartEntry, values map[string]interface{}, bbeConfig models.BbeConfig, helmService HelmServiceInterface, uiService UiServiceInterface, interactive bool) (map[string]interface{}, error)
	UninstallPackage(chart models.LocalPackage, bbeConfig models.BbeConfig, helmService HelmServiceInterface) error
}
