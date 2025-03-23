package mocks

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/mock"
)

type MockPackageService struct {
	mock.Mock
}

func (m *MockPackageService) GetAllBundles() []models.BbeBundle {
	args := m.Called()
	return args.Get(0).([]models.BbeBundle)
}

func (m *MockPackageService) InstallBundle(bundle models.BbeBundle, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	args := m.Called(bundle)
	return args.Error(0)
}

func (m *MockPackageService) UpgradeBundle(bundle models.BbeBundle, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	args := m.Called(bundle)
	return args.Error(0)
}

func (m *MockPackageService) UninstallBundle(bundle models.BbeBundle, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	args := m.Called(bundle)
	return args.Error(0)
}
