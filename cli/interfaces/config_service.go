package interfaces

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
)

type ConfigServiceInterface interface {
	GetBbeConfig(helperService HelperServiceInterface) (*models.BbeConfig, error)
	GenerateBbeConfig(helperService HelperServiceInterface, storage string) error
	UpdateBbeClusterName(helperService HelperServiceInterface, clusterName string) error
	UpdateBbeStorageType(helperService HelperServiceInterface, storageType string) error
	UpdateBbeAwsBucketName(helperService HelperServiceInterface, bucketName string) error
	UpdateBbePackages(helperService HelperServiceInterface, packages []models.Package) error
	CheckForTalosConfigs(helperService HelperServiceInterface) bool
	SyncConfigsWithAws(helperService HelperServiceInterface, bbeConfig *models.BbeConfig) error
}
