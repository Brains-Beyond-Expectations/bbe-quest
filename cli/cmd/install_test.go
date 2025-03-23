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
	configService.AssertNumberOfCalls(t, "UpdateBbeBundles", 2)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 1)
	packageService.AssertNumberOfCalls(t, "UninstallBundle", 1)
	packageService.AssertCalled(t, "UninstallBundle", models.BbeBundle{
		Name: "bundle_to_be_removed",
	})
	packageService.AssertNumberOfCalls(t, "InstallBundle", 2)
	packageService.AssertCalled(t, "InstallBundle", models.BbeBundle{
		Name: "bundle_to_be_installed",
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
	configService.AssertNumberOfCalls(t, "UpdateBbeBundles", 0)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 0)
	packageService.AssertNumberOfCalls(t, "UninstallBundle", 0)
	packageService.AssertNumberOfCalls(t, "InstallBundle", 0)
}

func Test_installCommand_Succeeds_ProceedsWhenFailingToUninstallBundles(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initInstallCommand()

	packageService.On("UninstallBundle", mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService)

	assert.Nil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbeBundles", 2)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 1)
	packageService.AssertNumberOfCalls(t, "UninstallBundle", 1)
	packageService.AssertCalled(t, "UninstallBundle", models.BbeBundle{
		Name: "bundle_to_be_removed",
	})
	packageService.AssertNumberOfCalls(t, "InstallBundle", 2)
	packageService.AssertCalled(t, "InstallBundle", models.BbeBundle{
		Name: "bundle_to_be_installed",
	})
}

func Test_installCommand_Fails_WhenFailingToUpdateBbeConfigurationOnUninstall(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initInstallCommand()

	configService.On("UpdateBbeBundles", mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbeBundles", 1)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 1)
	packageService.AssertNumberOfCalls(t, "UninstallBundle", 1)
	packageService.AssertCalled(t, "UninstallBundle", models.BbeBundle{
		Name: "bundle_to_be_removed",
	})
	packageService.AssertNumberOfCalls(t, "InstallBundle", 0)
}

func Test_installCommand_Fails_WhenFailingToInstallBundle(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initInstallCommand()

	packageService.On("InstallBundle", mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbeBundles", 1)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 1)
	packageService.AssertNumberOfCalls(t, "UninstallBundle", 1)
	packageService.AssertCalled(t, "UninstallBundle", models.BbeBundle{
		Name: "bundle_to_be_removed",
	})
	packageService.AssertNumberOfCalls(t, "InstallBundle", 1)
}

func Test_installCommand_Fails_WhenFailingToUpdateBbeConfigurationOnInstall(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initInstallCommand()

	configService.On("UpdateBbeBundles", mock.Anything, mock.Anything).Return(nil).Once()
	configService.On("UpdateBbeBundles", mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbeBundles", 2)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 1)
	packageService.AssertNumberOfCalls(t, "UninstallBundle", 1)
	packageService.AssertCalled(t, "UninstallBundle", models.BbeBundle{
		Name: "bundle_to_be_removed",
	})
	packageService.AssertNumberOfCalls(t, "InstallBundle", 2)
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
		"bundle_always_installed",
		"bundle_to_be_installed",
	}, nil)

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Bundles = []models.BbeBundle{
		{
			Name: "bundle_always_installed",
		},
		{
			Name: "bundle_to_be_removed",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)
	configService.On("UpdateBbeBundles", mock.Anything, mock.Anything).Return(nil)

	packageService.On("GetAllBundles").Return([]models.BbeBundle{
		{
			Name: "bundle_always_installed",
		},
		{
			Name: "bundle_to_be_removed",
		},
		{
			Name: "bundle_to_be_installed",
		},
		{
			Name: "bundle_to_be_ignored",
		},
	})
	packageService.On("UninstallBundle", mock.Anything).Return(nil)
	packageService.On("InstallBundle", mock.Anything).Return(nil)
}
