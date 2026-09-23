package mocks

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/mock"
)

type MockPackageService struct {
	mock.Mock
}

func (m *MockPackageService) GetAll() ([]models.ChartEntry, error) {
	args := m.Called()
	return args.Get(0).([]models.ChartEntry), args.Error(1)
}

func (m *MockPackageService) InstallPackage(pkg models.ChartEntry, values map[string]interface{}, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface, uiService interfaces.UiServiceInterface) (map[string]interface{}, error) {
	args := m.Called(pkg, values)
	resolved, _ := args.Get(0).(map[string]interface{})
	return resolved, args.Error(1)
}

func (m *MockPackageService) UpgradePackage(pkg models.ChartEntry, values map[string]interface{}, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface, uiService interfaces.UiServiceInterface, interactive bool) (map[string]interface{}, error) {
	args := m.Called(pkg, values, interactive)
	resolved, _ := args.Get(0).(map[string]interface{})
	return resolved, args.Error(1)
}

func (m *MockPackageService) UninstallPackage(pkg models.LocalPackage, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	args := m.Called(pkg)
	return args.Error(0)
}
