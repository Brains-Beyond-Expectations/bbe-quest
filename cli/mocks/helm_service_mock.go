package mocks

import (
	"github.com/stretchr/testify/mock"
)

type MockHelmService struct {
	mock.Mock
}

func (m *MockHelmService) AddRepo(repoName string, repoUrl string) error {
	args := m.Called(repoName, repoUrl)
	return args.Error(0)
}

func (m *MockHelmService) InstallChart(pkgName string, chartName string, repoName string, version string, namespace string, context string) error {
	args := m.Called(pkgName, chartName, repoName, version, namespace, context)
	return args.Error(0)
}

func (m *MockHelmService) UpgradeChart(pkgName string, chartName string, repoName string, version string, namespace string, context string) error {
	args := m.Called(pkgName, chartName, repoName, version, namespace, context)
	return args.Error(0)
}

func (m *MockHelmService) UninstallChart(pkgName string, namespace string, context string) error {
	args := m.Called(pkgName, namespace, context)
	return args.Error(0)
}

func (m *MockHelmService) Status(pkgName string, namespace string, context string) (bool, error) {
	args := m.Called(pkgName, namespace, context)
	return args.Bool(0), args.Error(1)
}

func (m *MockHelmService) IsPackageInstalled(pkgName string, namespace string, context string) bool {
	args := m.Called(pkgName, namespace, context)
	return args.Bool(0)
}
