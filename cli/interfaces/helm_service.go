package interfaces

type HelmServiceInterface interface {
	AddRepo(repoName string, repoUrl string) error
	InstallChart(pkgName string, chartName string, repoName string, version string, namespace string, context string) error
	UpgradeChart(pkgName string, chartName string, repoName string, version string, namespace string, context string) error
	UninstallChart(pkgName string, namespace string, context string) error
	Status(pkgName string, namespace string, context string) (bool, error)
	IsPackageInstalled(pkgName string, namespace string, context string) bool
}
