package mocks

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/mock"
)

type MockPackageService struct {
	mock.Mock
}

func (m *MockPackageService) GetAll() []models.Package {
	args := m.Called()
	return args.Get(0).([]models.Package)
}

func (m *MockPackageService) InstallPackage(pkg models.Package, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	args := m.Called(pkg)
	return args.Error(0)
}

func (m *MockPackageService) UpgradePackage(pkg models.Package, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	args := m.Called(pkg)
	return args.Error(0)
}

func (m *MockPackageService) UninstallPackage(pkg models.Package, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	args := m.Called(pkg)
	return args.Error(0)
}
