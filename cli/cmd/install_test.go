package cmd

import (
	"errors"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_installCommand_Succeeds(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initInstallCommand()

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService)

	assert.Nil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 2)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	packageService.AssertNumberOfCalls(t, "UninstallPackage", 1)
	packageService.AssertCalled(t, "UninstallPackage", models.LocalPackage{
		Name:    "package_to_be_removed",
		Version: "2.0.0",
	})
	packageService.AssertNumberOfCalls(t, "InstallPackage", 2)
	packageService.AssertCalled(t, "InstallPackage", models.ChartEntry{
		Name:    "package_to_be_installed",
		Version: "3.0.0",
	})
}

func Test_installCommand_Fails_WithNoCluster(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initInstallCommand()

	configService.On("GetBbeConfig", mock.Anything).Return(&models.BbeConfig{}, errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService)

	assert.Nil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 0)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 0)
	packageService.AssertNumberOfCalls(t, "GetAll", 0)
	packageService.AssertNumberOfCalls(t, "UninstallPackage", 0)
	packageService.AssertNumberOfCalls(t, "InstallPackage", 0)
}

func Test_installCommand_Succeeds_ProceedsWhenFailingToUninstallPackages(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initInstallCommand()

	packageService.On("UninstallPackage", mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService)

	assert.Nil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 2)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	packageService.AssertNumberOfCalls(t, "UninstallPackage", 1)
	packageService.AssertCalled(t, "UninstallPackage", models.LocalPackage{
		Name:    "package_to_be_removed",
		Version: "2.0.0",
	})
	packageService.AssertNumberOfCalls(t, "InstallPackage", 2)
	packageService.AssertCalled(t, "InstallPackage", models.ChartEntry{
		Name:    "package_always_installed",
		Version: "1.0.0",
	})
}

func Test_installCommand_Fails_WhenFailingToUpdateBbeConfigurationOnUninstall(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initInstallCommand()

	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 1)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	packageService.AssertNumberOfCalls(t, "UninstallPackage", 1)
	packageService.AssertCalled(t, "UninstallPackage", models.LocalPackage{
		Name:    "package_to_be_removed",
		Version: "2.0.0",
	})
	packageService.AssertNumberOfCalls(t, "InstallPackage", 0)
}

func Test_installCommand_Fails_WhenFailingToInstallPackage(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initInstallCommand()

	packageService.On("InstallPackage", mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 1)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	packageService.AssertNumberOfCalls(t, "UninstallPackage", 1)
	packageService.AssertCalled(t, "UninstallPackage", models.LocalPackage{
		Name:    "package_to_be_removed",
		Version: "2.0.0",
	})
	packageService.AssertNumberOfCalls(t, "InstallPackage", 1)
}

func Test_installCommand_Fails_WhenFailingToUpdateBbeConfigurationOnInstall(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initInstallCommand()

	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(nil).Once()
	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 2)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	packageService.AssertNumberOfCalls(t, "UninstallPackage", 1)
	packageService.AssertCalled(t, "UninstallPackage", models.LocalPackage{
		Name:    "package_to_be_removed",
		Version: "2.0.0",
	})
	packageService.AssertNumberOfCalls(t, "InstallPackage", 2)
}

func initInstallCommand() (*mocks.MockHelperService, *mocks.MockUiService, *mocks.MockConfigService, *mocks.MockPackageService, *mocks.MockHelmService) {
	helperService := &mocks.MockHelperService{}
	uiService := &mocks.MockUiService{}
	configService := &mocks.MockConfigService{}
	packageService := &mocks.MockPackageService{}
	helmService := &mocks.MockHelmService{}

	return helperService, uiService, configService, packageService, helmService
}

func mockSuccessfulInstallFlow(_ *mocks.MockHelperService, uiService *mocks.MockUiService, configService *mocks.MockConfigService, packageService *mocks.MockPackageService) {
	uiService.On("CreateMultiChoose", mock.Anything, mock.Anything, mock.Anything).Return([]string{
		"package_always_installed",
		"package_to_be_installed",
	}, nil)

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Packages = []models.LocalPackage{
		{
			Name:    "package_always_installed",
			Version: "1.0.0",
		},
		{
			Name:    "package_to_be_removed",
			Version: "2.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)
	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(nil)

	packageService.On("GetAll").Return([]models.ChartEntry{
		{
			Name:    "package_always_installed",
			Version: "1.0.0",
		},
		{
			Name:    "package_to_be_removed",
			Version: "2.0.0",
		},
		{
			Name:    "package_to_be_installed",
			Version: "3.0.0",
		},
		{
			Name:    "package_to_be_ignored",
			Version: "4.0.0",
		},
	}, nil)
	packageService.On("UninstallPackage", mock.Anything).Return(nil)
	packageService.On("InstallPackage", mock.Anything).Return(nil)
}
