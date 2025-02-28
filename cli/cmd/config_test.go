package cmd

import (
	"errors"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_configCommand_Succeeds_WithPreexistingLocalConfig(t *testing.T) {
	helperService := mocks.MockHelperService{}

	uiService := mocks.MockUiService{}

	configService := mocks.MockConfigService{}
	configService.On("GetBbeConfig", &helperService).Return(&models.BbeConfig{}, nil)

	err := configCommand(&helperService, &uiService, &configService)

	assert.Nil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
}

func Test_configCommand_Succeeds_WithPreexistinAwsConfig(t *testing.T) {
	helperService := mocks.MockHelperService{}

	uiService := mocks.MockUiService{}

	configService := mocks.MockConfigService{}
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Storage.Type = "aws"
	configService.On("GetBbeConfig", &helperService).Return(bbeConfig, nil)
	configService.On("SyncConfigsWithAws", &helperService, bbeConfig).Return(nil)

	err := configCommand(&helperService, &uiService, &configService)

	assert.Nil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 1)
}

func Test_configCommand_Succeeds_WithNoConfig_GeneratesLocalConfig(t *testing.T) {
	helperService := mocks.MockHelperService{}

	uiService := mocks.MockUiService{}
	uiService.On("CreateSelect", mock.Anything, []string{"Local", "AWS"}).Return("Local", nil)

	configService := mocks.MockConfigService{}
	configService.On("GetBbeConfig", &helperService).Return(&models.BbeConfig{}, errors.New("test error")).Once()
	configService.On("GenerateBbeConfig", &helperService, "local").Return(nil)
	configService.On("GetBbeConfig", &helperService).Return(&models.BbeConfig{}, nil).Once()

	err := configCommand(&helperService, &uiService, &configService)

	assert.Nil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 2)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 1)
}

func Test_configCommand_Succeeds_WithNoConfig_GeneratesAwsConfig(t *testing.T) {
	helperService := mocks.MockHelperService{}

	uiService := mocks.MockUiService{}
	uiService.On("CreateSelect", mock.Anything, []string{"Local", "AWS"}).Return("AWS", nil)

	configService := mocks.MockConfigService{}
	configService.On("GetBbeConfig", &helperService).Return(&models.BbeConfig{}, errors.New("test error")).Once()
	configService.On("GenerateBbeConfig", &helperService, "aws").Return(nil)
	configService.On("GetBbeConfig", &helperService).Return(&models.BbeConfig{}, nil).Once()

	err := configCommand(&helperService, &uiService, &configService)

	assert.Nil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 2)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 1)
}

func Test_configCommand_Fails_WithPreexistinAwsConfig_WhenFailingToSync(t *testing.T) {
	helperService := mocks.MockHelperService{}

	uiService := mocks.MockUiService{}

	configService := mocks.MockConfigService{}
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Storage.Type = "aws"
	configService.On("GetBbeConfig", &helperService).Return(bbeConfig, nil)
	configService.On("SyncConfigsWithAws", &helperService, bbeConfig).Return(errors.New("test error"))

	err := configCommand(&helperService, &uiService, &configService)

	assert.NotNil(t, err)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 1)
}

func Test_configCommand_Fails_WithNoConfig_FailsToGenerateLocalConfig(t *testing.T) {
	helperService := mocks.MockHelperService{}

	uiService := mocks.MockUiService{}
	uiService.On("CreateSelect", mock.Anything, []string{"Local", "AWS"}).Return("Local", nil)

	configService := mocks.MockConfigService{}
	configService.On("GetBbeConfig", &helperService).Return(&models.BbeConfig{}, errors.New("test error")).Once()
	configService.On("GenerateBbeConfig", &helperService, "local").Return(errors.New("test error"))
	configService.On("GetBbeConfig", &helperService).Return(&models.BbeConfig{}, nil).Once()

	err := configCommand(&helperService, &uiService, &configService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 1)
}

func Test_configCommand_Fails_WithNoConfig_FailsToGenerateAwsonfig(t *testing.T) {
	helperService := mocks.MockHelperService{}

	uiService := mocks.MockUiService{}
	uiService.On("CreateSelect", mock.Anything, []string{"Local", "AWS"}).Return("AWS", nil)

	configService := mocks.MockConfigService{}
	configService.On("GetBbeConfig", &helperService).Return(&models.BbeConfig{}, errors.New("test error")).Once()
	configService.On("GenerateBbeConfig", &helperService, "aws").Return(errors.New("test error"))
	configService.On("GetBbeConfig", &helperService).Return(&models.BbeConfig{}, nil).Once()

	err := configCommand(&helperService, &uiService, &configService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 1)
}

func Test_configCommand_Fails_WithNoConfig_FailsToRetrieveConfigAfterGeneration(t *testing.T) {
	helperService := mocks.MockHelperService{}

	uiService := mocks.MockUiService{}
	uiService.On("CreateSelect", mock.Anything, []string{"Local", "AWS"}).Return("Local", nil)

	configService := mocks.MockConfigService{}
	configService.On("GetBbeConfig", &helperService).Return(&models.BbeConfig{}, errors.New("test error"))
	configService.On("GenerateBbeConfig", &helperService, "local").Return(nil)

	err := configCommand(&helperService, &uiService, &configService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateSelect", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 2)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 1)
}
