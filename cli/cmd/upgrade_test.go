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
	helperService, uiService, configService, packageService, helmService := initUpgradeCommand()

	mockSuccessfulUpgradeFlow(helperService, uiService, configService, packageService)

	err := upgradeCommand(helperService, uiService, configService, packageService, helmService, false)

	assert.Nil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 1)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 0)
	packageService.AssertNumberOfCalls(t, "UpgradeBundle", 0)
	configService.AssertCalled(t, "UpdateBbeBundles", mock.Anything, []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "2.0.0",
		},
	})
}

func Test_upgradeCommand_Fails_With_No_Cluster_name(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initUpgradeCommand()

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = ""
	bbeConfig.Bbe.Bundles = []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "1.0.0",
		},
		{
			Name:    "bundle_two",
			Version: "1.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)

	err := upgradeCommand(helperService, uiService, configService, packageService, helmService, true)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No BBE cluster found, please run 'bbe setup' to create your cluster")
}

func Test_upgradeCommand_Fails_With_Error_Getting_BBE_Config(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initUpgradeCommand()

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = ""
	bbeConfig.Bbe.Bundles = []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "1.0.0",
		},
		{
			Name:    "bundle_two",
			Version: "1.0.0",
		},
	}

	fakeError := errors.New("Fake GetBbeConfig error")
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, fakeError)

	err := upgradeCommand(helperService, uiService, configService, packageService, helmService, true)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No BBE cluster found, please run 'bbe setup' to create your cluster")
}

func Test_upgradeCommand_Fails_WhenCreateSelect(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initUpgradeCommand()

	fakeError := errors.New("Fake select error")
	uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("Yes", fakeError).Once()

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Bundles = []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "1.0.0",
		},
		{
			Name:    "bundle_two",
			Version: "1.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)

	mockSuccessfulUpgradeFlow(helperService, uiService, configService, packageService)

	err := upgradeCommand(helperService, uiService, configService, packageService, helmService, false)

	assert.Error(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 1)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 1)
	packageService.AssertNumberOfCalls(t, "UpgradeBundle", 0)

}

func Test_upgradeCommand_Succeeds_WithInteractiveUpgrade(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initUpgradeCommand()

	uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("Yes", nil).Once()
	uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("No", nil).Once()
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Bundles = []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "1.0.0",
		},
		{
			Name:    "bundle_two",
			Version: "1.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)

	mockSuccessfulUpgradeFlow(helperService, uiService, configService, packageService)

	err := upgradeCommand(helperService, uiService, configService, packageService, helmService, false)

	assert.Nil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 1)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 2)
	packageService.AssertNumberOfCalls(t, "UpgradeBundle", 1)
	configService.AssertCalled(t, "UpdateBbeBundles", mock.Anything, []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "2.0.0",
		},
		{
			Name:    "bundle_two",
			Version: "1.0.0",
		},
	})
}

func Test_upgradeCommand_Succeeds_WithNonInteractiveUpgrade(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initUpgradeCommand()

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Bundles = []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "1.0.0",
		},
		{
			Name:    "bundle_two",
			Version: "1.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)

	mockSuccessfulUpgradeFlow(helperService, uiService, configService, packageService)

	err := upgradeCommand(helperService, uiService, configService, packageService, helmService, true)

	assert.Nil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 1)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 0)
	packageService.AssertNumberOfCalls(t, "UpgradeBundle", 2)
	configService.AssertCalled(t, "UpdateBbeBundles", mock.Anything, []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "2.0.0",
		},
		{
			Name:    "bundle_two",
			Version: "3.0.0",
		},
	})
}

func Test_upgradeCommand_Fails_Prtial_UpdatesBbeConfig(t *testing.T) {
	helperService, uiService, configService, packageService, helmService := initUpgradeCommand()

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Bundles = []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "1.0.0",
		},
		{
			Name:    "bundle_two",
			Version: "1.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)

	packageService.On("UpgradeBundle", mock.Anything).Return(nil).Once()
	packageService.On("UpgradeBundle", mock.Anything).Return(errors.New("test error")).Once()

	mockSuccessfulUpgradeFlow(helperService, uiService, configService, packageService)

	err := upgradeCommand(helperService, uiService, configService, packageService, helmService, true)

	assert.NotNil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	packageService.AssertNumberOfCalls(t, "GetAllBundles", 1)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 0)
	packageService.AssertNumberOfCalls(t, "UpgradeBundle", 2)
	configService.AssertCalled(t, "UpdateBbeBundles", mock.Anything, []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "2.0.0",
		},
		{
			Name:    "bundle_two",
			Version: "1.0.0",
		},
	})
}

func initUpgradeCommand() (*mocks.MockHelperService, *mocks.MockUiService, *mocks.MockConfigService, *mocks.MockPackageService, *mocks.MockHelmService) {
	helperService := &mocks.MockHelperService{}
	uiService := &mocks.MockUiService{}
	configService := &mocks.MockConfigService{}
	packageService := &mocks.MockPackageService{}
	helmService := &mocks.MockHelmService{}

	return helperService, uiService, configService, packageService, helmService
}

func mockSuccessfulUpgradeFlow(_ *mocks.MockHelperService, uiService *mocks.MockUiService, configService *mocks.MockConfigService, packageService *mocks.MockPackageService) {
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Bundles = []models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "2.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)
	configService.On("UpdateBbeBundles", mock.Anything, mock.Anything).Return(nil)

	packageService.On("GetAllBundles").Return([]models.BbeBundle{
		{
			Name:    "bundle_one",
			Version: "2.0.0",
		},
		{
			Name:    "bundle_two",
			Version: "3.0.0",
		},
	})
	packageService.On("UpgradeBundle", mock.Anything).Return(nil)
}
