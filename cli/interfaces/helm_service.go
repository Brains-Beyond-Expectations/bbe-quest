package interfaces

import "github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"

type HelmServiceInterface interface {
	AddRepo(repoName string, repoUrl string) error
	InstallChart(pkgName string, chartName string, repoName string, version string, namespace string, context string, values map[string]interface{}) error
	UpgradeChart(pkgName string, chartName string, repoName string, version string, namespace string, context string, values map[string]interface{}) error
	UninstallChart(pkgName string, namespace string, context string) error
	IsPackageInstalled(pkgName string, namespace string, context string) bool
	PullChart(repositoryUrl string, chartName string, version string) (*models.HelmChart, error)
}
