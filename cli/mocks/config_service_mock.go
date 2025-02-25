package mocks

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/mock"
)

type MockConfigService struct {
	mock.Mock
}

func (m *MockConfigService) GetBbeConfig(helperService interfaces.HelperServiceInterface) (*models.BbeConfig, error) {
	args := m.Called(helperService)
	return args.Get(0).(*models.BbeConfig), args.Error(1)
}

func (m *MockConfigService) GenerateBbeConfig(helperService interfaces.HelperServiceInterface, storage string) error {
	args := m.Called(helperService, storage)
	return args.Error(0)
}

func (m *MockConfigService) UpdateBbeClusterName(helperService interfaces.HelperServiceInterface, clusterName string) error {
	args := m.Called(helperService, clusterName)
	return args.Error(0)
}

func (m *MockConfigService) UpdateBbeStorageType(helperService interfaces.HelperServiceInterface, storageType string) error {
	args := m.Called(helperService, storageType)
	return args.Error(0)
}

func (m *MockConfigService) UpdateBbeAwsBucketName(helperService interfaces.HelperServiceInterface, bucketName string) error {
	args := m.Called(helperService, bucketName)
	return args.Error(0)
}

func (m *MockConfigService) UpdateBbePackages(helperService interfaces.HelperServiceInterface, packages []models.Package) error {
	args := m.Called(helperService, packages)
	return args.Error(0)
}

func (m *MockConfigService) CheckForTalosConfigs(helperService interfaces.HelperServiceInterface) bool {
	args := m.Called(helperService)
	return args.Bool(0)
}

func (m *MockConfigService) SyncConfigsWithAws(helperService interfaces.HelperServiceInterface, bbeConfig *models.BbeConfig) error {
	args := m.Called(helperService, bbeConfig)
	return args.Error(0)
}
