package models

type BbePackage struct {
	Package           Package
	HelmChart         string
	HelmChartVersion  string
	PackageRepository BbePackageRepository
}
