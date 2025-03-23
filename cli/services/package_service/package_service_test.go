package package_service

import (
	"errors"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_GetAll_Succeeds(t *testing.T) {
	packagesService := PackageService{}

	// Get all packages directly from the service
	result := packagesService.GetAllBundles()

	// Assert that the result contains the correct package data
	assert.Len(t, result, 1) // We have one package in the predefined array
	assert.Equal(t, "network", result[0].Name)
	assert.Equal(t, "0.1.0", result[0].Version)
	assert.Len(t, result[0].BbePackages, 2) // Package contains 2 packages
}

func Test_GetAllBundles_Succeeds(t *testing.T) {
	packagesService := PackageService{}

	// Get all bundles directly from the service
	result := packagesService.GetAllBundles()

	// Assert that the result contains the correct bundle data
	assert.Len(t, result, 1) // We have one bundle in the predefined array
	assert.Equal(t, "network", result[0].Name)
	assert.Equal(t, "0.1.0", result[0].Version)
	assert.Len(t, result[0].BbePackages, 2) // Bundle contains 2 packages
}

func Test_InstallBundle_Fails_WhenHelmInstallFails(t *testing.T) {
	mockErrorMessage := "Mock failed to install"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}
	bundle := models.BbeBundle{
		Name: "network",
		BbePackages: []models.BbePackage{
			{
				Package: models.Package{
					Name:    "ingress-nginx",
					Version: "4.12.0",
				},
			},
		},
	}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallBundle(bundle, bbeConfig, mockHelmService)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_InstallBundle_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	packagesService := PackageService{}
	bundle := models.BbeBundle{
		Name: "network",
		BbePackages: []models.BbePackage{
			{
				Package: models.Package{
					Name:    "ingress-nginx",
					Version: "4.12.0",
				},
			},
		},
	}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallBundle(bundle, bbeConfig, mockHelmService)

	assert.NoError(t, err)
}

func Test_UninstallBundle_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("UninstallChart", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	packagesService := PackageService{}
	bundle := models.BbeBundle{
		Name: "network",
		BbePackages: []models.BbePackage{
			{
				Package: models.Package{
					Name:    "ingress-nginx",
					Version: "4.12.0",
				},
			},
		},
	}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UninstallBundle(bundle, bbeConfig, mockHelmService)

	assert.NoError(t, err)
}

func Test_UpgradeBundle_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	packagesService := PackageService{}
	bundle := models.BbeBundle{
		Name: "network",
		BbePackages: []models.BbePackage{
			{
				Package: models.Package{
					Name:    "ingress-nginx",
					Version: "4.12.0",
				},
			},
		},
	}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradeBundle(bundle, bbeConfig, mockHelmService)

	assert.NoError(t, err)
}

func Test_UninstallBundle_Fails_WhenHelmUninstallFails(t *testing.T) {
	mockErrorMessage := "Mock failed to uninstall"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("UninstallChart", mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}
	bundle := models.BbeBundle{
		Name: "network",
		BbePackages: []models.BbePackage{
			{
				Package: models.Package{
					Name:    "ingress-nginx",
					Version: "4.12.0",
				},
			},
		},
	}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UninstallBundle(bundle, bbeConfig, mockHelmService)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_UpgradeBundle_Fails_WhenHelmUpgradeFails(t *testing.T) {
	mockErrorMessage := "Mock failed to upgrade"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}
	bundle := models.BbeBundle{
		Name: "network",
		BbePackages: []models.BbePackage{
			{
				Package: models.Package{
					Name:    "ingress-nginx",
					Version: "4.12.0",
				},
			},
		},
	}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradeBundle(bundle, bbeConfig, mockHelmService)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_UpgradeBundle_Fails_WhenAddRepoFails(t *testing.T) {
	mockErrorMessage := "Mock failed to add repo"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}
	bundle := models.BbeBundle{
		Name: "network",
		BbePackages: []models.BbePackage{
			{
				Package: models.Package{
					Name:    "ingress-nginx",
					Version: "4.12.0",
				},
			},
		},
	}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradeBundle(bundle, bbeConfig, mockHelmService)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_InstallBundle_Fails_WhenAddRepoFails(t *testing.T) {
	mockErrorMessage := "Mock failed to add repo"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}
	bundle := models.BbeBundle{
		Name: "network",
		BbePackages: []models.BbePackage{
			{
				Package: models.Package{
					Name:    "ingress-nginx",
					Version: "4.12.0",
				},
			},
		},
	}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallBundle(bundle, bbeConfig, mockHelmService)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}
