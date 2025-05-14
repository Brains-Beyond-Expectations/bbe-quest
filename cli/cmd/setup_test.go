package cmd

import (
	"errors"
	"testing"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_setupCommand_Succeeds_WithControlPlane_RaspberryPi(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 1)
}

func Test_setupCommand_Succeeds_WithWorkerNode_RaspberryPi(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	uiService.On("CreateSelect", "Is this the first node in your cluster?", mock.Anything).Return("No", nil)
	configService.On("CheckForTalosConfigs", helperService).Return(true)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, false)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Succeeds_WithControlPlane_IntelNUC(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	uiService.On("CreateSelect", "What type of device are you setting up?", mock.Anything).Return("Intel NUC", nil)
	uiService.On("CreateSelect", "Please use balenaEtcher to flash the .iso to your USB device", mock.Anything).Return("Done", nil)
	uiService.On("CreateSelect", "Please insert the USB device into your new node and boot from it", mock.Anything).Return("Done", nil)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 1)
}

func Test_setupCommand_Succeeds_WithWorkerNode_IntelNUC(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	uiService.On("CreateSelect", "Is this the first node in your cluster?", mock.Anything).Return("No", nil)
	configService.On("CheckForTalosConfigs", helperService).Return(true)

	uiService.On("CreateSelect", "What type of device are you setting up?", mock.Anything).Return("Intel NUC", nil)
	uiService.On("CreateSelect", "Please use balenaEtcher to flash the .iso to your USB device", mock.Anything).Return("Done", nil)
	uiService.On("CreateSelect", "Please insert the USB device into your new node and boot from it", mock.Anything).Return("Done", nil)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, false)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Succeeds_WithIpNotFoundFallback(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	ipFinderService.On("GetGatewayIp", helperService).Return("", errors.New("test error"))
	uiService.On("CreateInput", "Gateway IP not found, please enter the IP of the network you want to scan:", mock.Anything).Return("test", nil)
	helperService.On("IsValidIp", "test").Return(false)
	uiService.On("CreateInput", "Invalid Gateway IP, please enter a valid IP:", mock.Anything).Return(gatewayIp, nil)
	helperService.On("IsValidIp", gatewayIp).Return(true)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 2)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 1)
}

func Test_setupCommand_Succeeds_WithPreexistingImage(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	now := time.Now()
	helperService.On("CheckIfFileExists", mock.Anything).Return(&now, true)
	uiService.On("CreateSelect", "An image already exists, would you like to redownload it?", mock.Anything).Return("No", nil)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 0)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 1)
}

func Test_setupCommand_Succeeds_GeneratesLocalConfig(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	configService.On("GetBbeConfig", mock.Anything).Return(&models.BbeConfig{}, errors.New("test error")).Once()
	uiService.On("CreateSelect", "No BBE configuration file found, where would you like to store your config files?", mock.Anything).Return("Local", nil)
	configService.On("GenerateBbeConfig", helperService, "local").Return(nil)
	configService.On("GetBbeConfig", mock.Anything).Return(&models.BbeConfig{}, nil)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 1)
	configService.AssertCalled(t, "GenerateBbeConfig", helperService, "local")
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 1)
}

func Test_setupCommand_Succeeds_GeneratesAwsConfig(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	configService.On("GetBbeConfig", mock.Anything).Return(&models.BbeConfig{}, errors.New("test error")).Once()
	uiService.On("CreateSelect", "No BBE configuration file found, where would you like to store your config files?", mock.Anything).Return("AWS", nil)
	configService.On("GenerateBbeConfig", helperService, "aws").Return(nil)
	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Storage.Type = "aws"
	configService.On("GetBbeConfig", mock.Anything).Return(&bbeConfig, nil)
	configService.On("SyncConfigsWithAws", helperService, &bbeConfig).Return(nil)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 1)
	configService.AssertCalled(t, "GenerateBbeConfig", helperService, "aws")
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 1)
}

func Test_setupCommand_Fails_WithNoNodeFound(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	ipFinderService.On("LocateDevice", helperService, talosService, gatewayIp).Return([]string{}, nil)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 0)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails_WithMoreThanOneNodeFound(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	ipFinderService.On("LocateDevice", helperService, talosService, gatewayIp).Return([]string{gatewayIp, nodeIp, chosenIp}, nil)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 0)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails_WhenFailingToGenerateBbeConfig(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	configService.On("GetBbeConfig", mock.Anything).Return(&models.BbeConfig{}, errors.New("test error")).Once()
	uiService.On("CreateSelect", "No BBE configuration file found, where would you like to store your config files?", mock.Anything).Return("Local", nil)
	configService.On("GenerateBbeConfig", helperService, "local").Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 0)
	talosService.AssertNumberOfCalls(t, "GetDisks", 0)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 1)
	configService.AssertCalled(t, "GenerateBbeConfig", helperService, "local")
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails_WhenFailingToSyncConfigsWithAws(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	configService.On("GetBbeConfig", mock.Anything).Return(&models.BbeConfig{}, errors.New("test error")).Once()
	uiService.On("CreateSelect", "No BBE configuration file found, where would you like to store your config files?", mock.Anything).Return("AWS", nil)
	configService.On("GenerateBbeConfig", helperService, "aws").Return(nil)
	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Storage.Type = "aws"
	configService.On("GetBbeConfig", mock.Anything).Return(&bbeConfig, nil)
	configService.On("SyncConfigsWithAws", helperService, &bbeConfig).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 0)
	talosService.AssertNumberOfCalls(t, "GetDisks", 0)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 1)
	configService.AssertCalled(t, "GenerateBbeConfig", helperService, "aws")
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails_WhenDependenciesMissing(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	dependencyService.On("VerifyDependencies").Return(false)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 0)
	talosService.AssertNumberOfCalls(t, "GetDisks", 0)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails_WhenEnrollingIntoExistingClusterWithMissingConfigs(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	uiService.On("CreateSelect", "Is this the first node in your cluster?", mock.Anything).Return("No", nil)
	configService.On("CheckForTalosConfigs", helperService).Return(false)

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 0)
	talosService.AssertNumberOfCalls(t, "GetDisks", 0)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToDownloadImage(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	imageService.On("CreateImage", mock.Anything, mock.Anything).Return("imagePath", errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	ipFinderService.AssertNumberOfCalls(t, "LocateDevice", 0)
	talosService.AssertNumberOfCalls(t, "GetDisks", 0)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenNoDevicesAreFound(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	ipFinderService.On("LocateDevice", helperService, talosService, gatewayIp).Return([]string{}, errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	ipFinderService.AssertNumberOfCalls(t, "LocateDevice", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 0)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenNoDisksAreFound(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("GetDisks", helperService, nodeIp).Return([]string{}, errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToGenerateTalosConfig(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("GenerateConfig", helperService, chosenIp, "talos-cluster").Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToGetControlPlaneIp(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("GetControlPlaneIp", helperService, constants.ControlplaneConfigFile).Return("", errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToModifyNetworkNodeIp(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("ModifyNetworkNodeIp", helperService, mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToGetNetworkInterface(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("GetNetworkInterface", helperService, nodeIp).Return("", errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToModifyNetworkInterface(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("ModifyNetworkInterface", helperService, mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToModifyNetworkGateway(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("ModifyNetworkGateway", helperService, mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToModifyNetworkHostname(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("ModifyNetworkHostname", helperService, mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToModifySchedulingOnControlPlane(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("ModifySchedulingOnControlPlane", helperService, mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToModifyConfigDisk(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("ModifyConfigDisk", helperService, mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToJoinCluster(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("JoinCluster", helperService, nodeIp, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToBootstrapCluster(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("BootstrapCluster", helperService, chosenIp, chosenIp).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToVerifyNodeHealth(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("VerifyNodeHealth", helperService, chosenIp, chosenIp).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToDownloadKubeConfig(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	talosService.On("DownloadKubeConfig", helperService, chosenIp, chosenIp).Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 0)
}

func Test_setupCommand_Fails__WhenFailingToUpdateBbeClusterName(t *testing.T) {
	helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp := initSetupTests()

	configService.On("UpdateBbeClusterName", helperService, "talos-cluster").Return(errors.New("test error"))

	mockSuccessfulSetupFlow(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService, gatewayIp, nodeIp, chosenIp, true)

	err := setupCommand(helperService, dependencyService, talosService, ipFinderService, uiService, configService, imageService)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "IsValidIp", 0)
	imageService.AssertNumberOfCalls(t, "CreateImage", 1)
	talosService.AssertNumberOfCalls(t, "GetDisks", 1)
	configService.AssertNumberOfCalls(t, "GenerateBbeConfig", 0)
	configService.AssertNumberOfCalls(t, "SyncConfigsWithAws", 0)
	configService.AssertNumberOfCalls(t, "UpdateBbeClusterName", 1)
}

func initSetupTests() (*mocks.MockHelperService, *mocks.MockDependencyService, *mocks.MockTalosService, *mocks.MockIpFinderService, *mocks.MockUiService, *mocks.MockConfigService, *mocks.MockImageService, string, string, string) {
	helperService := mocks.MockHelperService{}
	dependencyService := mocks.MockDependencyService{}
	talosService := mocks.MockTalosService{}
	ipFinderService := mocks.MockIpFinderService{}
	uiService := mocks.MockUiService{}
	configService := mocks.MockConfigService{}
	imageService := mocks.MockImageService{}

	gatewayIp := "127.0.0.1"
	nodeIp := "1.2.3.4"
	chosenIp := "5.6.7.8"

	return &helperService, &dependencyService, &talosService, &ipFinderService, &uiService, &configService, &imageService, gatewayIp, nodeIp, chosenIp
}

func mockSuccessfulSetupFlow(helperService *mocks.MockHelperService, dependencyService *mocks.MockDependencyService, talosService *mocks.MockTalosService, ipFinderService *mocks.MockIpFinderService, uiService *mocks.MockUiService, configService *mocks.MockConfigService, imageService *mocks.MockImageService, gatewayIp, nodeIp, chosenIp string, isControlPlane bool) {
	nodeTypeConfigFile := constants.ControlplaneConfigFile
	if !isControlPlane {
		nodeTypeConfigFile = constants.WorkerConfigFile
	}

	configService.On("GetBbeConfig", mock.Anything).Return(&models.BbeConfig{}, nil)
	dependencyService.On("VerifyDependencies").Return(true)
	uiService.On("CreateSelect", "Is this the first node in your cluster?", mock.Anything).Return("Yes", nil)
	configService.On("CheckForTalosConfigs", helperService).Return(false)
	uiService.On("CreateSelect", "What type of device are you setting up?", mock.Anything).Return("Raspberry Pi 4 (or older)", nil)
	now := time.Now()
	helperService.On("CheckIfFileExists", mock.Anything).Return(&now, false)
	imageService.On("CreateImage", mock.Anything, mock.Anything).Return("imagePath", nil)
	uiService.On("CreateSelect", "Please use balenaEtcher to flash the .xz to your SD card", mock.Anything).Return("Done", nil)
	uiService.On("CreateSelect", "Please insert the SD card into your new node and boot from it", mock.Anything).Return("Done", nil)
	ipFinderService.On("GetGatewayIp", helperService).Return(gatewayIp, nil)
	ipFinderService.On("LocateDevice", helperService, talosService, gatewayIp).Return([]string{nodeIp}, nil)
	uiService.On("CreateInput", "Please choose an ip for the new node", nodeIp).Return(chosenIp, nil)
	talosService.On("GetDisks", helperService, nodeIp).Return([]string{"node", "namespace", "sda"}, nil)
	uiService.On("CreateSelect", "Please select the disk to install Talos on for 5.6.7.8", mock.Anything).Return("node namespace sda", nil)
	uiService.On("CreateInput", "Please choose the correct gateway ip", gatewayIp).Return(gatewayIp, nil)
	uiService.On("CreateInput", "Please select the hostname", mock.Anything).Return("talos-node", nil)
	uiService.On("CreateInput", "Please enter what you want to name your cluster", mock.Anything).Return("talos-cluster", nil)
	uiService.On("CreateSelect", "Do you want to allow scheduling on the control plane? This is required if you have only one node.", mock.Anything).Return("Yes", nil)
	talosService.On("GenerateConfig", helperService, chosenIp, "talos-cluster").Return(nil)
	talosService.On("GetControlPlaneIp", helperService, constants.ControlplaneConfigFile).Return(chosenIp, nil)
	talosService.On("ModifyNetworkNodeIp", helperService, nodeTypeConfigFile, chosenIp).Return(nil)
	talosService.On("GetNetworkInterface", helperService, nodeIp).Return("eth0", nil)
	talosService.On("ModifyNetworkInterface", helperService, nodeTypeConfigFile, "eth0").Return(nil)
	talosService.On("ModifyNetworkGateway", helperService, nodeTypeConfigFile, gatewayIp).Return(nil)
	talosService.On("ModifyNetworkHostname", helperService, nodeTypeConfigFile, "talos-node").Return(nil)
	talosService.On("ModifySchedulingOnControlPlane", helperService, true).Return(nil)
	talosService.On("ModifyConfigDisk", helperService, nodeTypeConfigFile, "/dev/sda").Return(nil)
	talosService.On("JoinCluster", helperService, nodeIp, nodeTypeConfigFile).Return(nil)
	talosService.On("BootstrapCluster", helperService, chosenIp, chosenIp).Return(nil)
	talosService.On("VerifyNodeHealth", helperService, chosenIp, chosenIp).Return(nil)
	talosService.On("DownloadKubeConfig", helperService, chosenIp, chosenIp).Return(nil)
	configService.On("UpdateBbeClusterName", helperService, "talos-cluster").Return(nil)
}
