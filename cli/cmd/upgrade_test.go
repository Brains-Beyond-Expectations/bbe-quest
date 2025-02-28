package cmd

import (
	"errors"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_upgradeCommand_Succeeds_WithNothingToDo(t *testing.T) {
	helperService, uiService, configService, packageService := initUpgradeCommand()

	mockSuccessfulUpgradeFlow(helperService, uiService, configService, packageService)

	err := upgradeCommand(helperService, uiService, configService, packageService, false)

	assert.Nil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 0)
	packageService.AssertNumberOfCalls(t, "UpgradePackage", 0)
	configService.AssertCalled(t, "UpdateBbePackages", mock.Anything, []models.Package{
		{
			Name:    "package_one",
			Version: "2.0.0",
		},
	})
}

func Test_upgradeCommand_Succeeds_WithInteractiveUpgrade(t *testing.T) {
	helperService, uiService, configService, packageService := initUpgradeCommand()

	uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("Yes", nil).Once()
	uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("No", nil).Once()
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Packages = []models.Package{
		{
			Name:    "package_one",
			Version: "1.0.0",
		},
		{
			Name:    "package_two",
			Version: "1.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)

	mockSuccessfulUpgradeFlow(helperService, uiService, configService, packageService)

	err := upgradeCommand(helperService, uiService, configService, packageService, false)

	assert.Nil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 2)
	packageService.AssertNumberOfCalls(t, "UpgradePackage", 1)
	configService.AssertCalled(t, "UpdateBbePackages", mock.Anything, []models.Package{
		{
			Name:    "package_one",
			Version: "2.0.0",
		},
		{
			Name:    "package_two",
			Version: "1.0.0",
		},
	})
}

func Test_upgradeCommand_Succeeds_WithNonInteractiveUpgrade(t *testing.T) {
	helperService, uiService, configService, packageService := initUpgradeCommand()

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Packages = []models.Package{
		{
			Name:    "package_one",
			Version: "1.0.0",
		},
		{
			Name:    "package_two",
			Version: "1.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)

	mockSuccessfulUpgradeFlow(helperService, uiService, configService, packageService)

	err := upgradeCommand(helperService, uiService, configService, packageService, true)

	assert.Nil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 0)
	packageService.AssertNumberOfCalls(t, "UpgradePackage", 2)
	configService.AssertCalled(t, "UpdateBbePackages", mock.Anything, []models.Package{
		{
			Name:    "package_one",
			Version: "2.0.0",
		},
		{
			Name:    "package_two",
			Version: "3.0.0",
		},
	})
}

func Test_upgradeCommand_Fails_Prtial_UpdatesBbeConfig(t *testing.T) {
	helperService, uiService, configService, packageService := initUpgradeCommand()

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Packages = []models.Package{
		{
			Name:    "package_one",
			Version: "1.0.0",
		},
		{
			Name:    "package_two",
			Version: "1.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)

	packageService.On("UpgradePackage", mock.Anything).Return(nil).Once()
	packageService.On("UpgradePackage", mock.Anything).Return(errors.New("test error")).Once()

	mockSuccessfulUpgradeFlow(helperService, uiService, configService, packageService)

	err := upgradeCommand(helperService, uiService, configService, packageService, true)

	assert.NotNil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 0)
	packageService.AssertNumberOfCalls(t, "UpgradePackage", 2)
	configService.AssertCalled(t, "UpdateBbePackages", mock.Anything, []models.Package{
		{
			Name:    "package_one",
			Version: "2.0.0",
		},
		{
			Name:    "package_two",
			Version: "1.0.0",
		},
	})
}

func initUpgradeCommand() (*mocks.MockHelperService, *mocks.MockUiService, *mocks.MockConfigService, *mocks.MockPackageService) {
	helperService := &mocks.MockHelperService{}
	uiService := &mocks.MockUiService{}
	configService := &mocks.MockConfigService{}
	packageService := &mocks.MockPackageService{}

	return helperService, uiService, configService, packageService
}

func mockSuccessfulUpgradeFlow(_ *mocks.MockHelperService, uiService *mocks.MockUiService, configService *mocks.MockConfigService, packageService *mocks.MockPackageService) {
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Packages = []models.Package{
		{
			Name:    "package_one",
			Version: "2.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)
	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(nil)

	packageService.On("GetAll").Return([]models.Package{
		{
			Name:    "package_one",
			Version: "2.0.0",
		},
		{
			Name:    "package_two",
			Version: "3.0.0",
		},
	})
	packageService.On("UpgradePackage", mock.Anything).Return(nil)
}
