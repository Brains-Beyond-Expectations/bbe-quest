package talos_service

import (
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v2"
)

func Test_Ping_Succeeds_ReturnsFalseIf_NotATalosMachine(t *testing.T) {
	timesCalled := 0
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		timesCalled++
		// Initial command to check for disks will fail, ie. not a Talos machine
		return exec.Command("false")
	}

	talosService := TalosService{}
	result := talosService.Ping("127.0.0.1")

	assert.False(t, result)
	assert.Equal(t, 1, timesCalled)
}

func Test_Ping_Succeeds_ReturnsFalseIf_MachineAlreadyInitialized(t *testing.T) {
	timesCalled := 0
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		timesCalled++
		// Initial command to check for disks will succeed and second command will also succeed, ie. machine already initialized
		return exec.Command("true")
	}

	talosService := TalosService{}
	result := talosService.Ping("127.0.0.1")

	assert.False(t, result)
	assert.Equal(t, 2, timesCalled)
}

func Test_Ping_Succeeds_ReturnsTrueIf_TalosMachineFound(t *testing.T) {
	timesCalled := 0
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		// Initial command to check for disks will succeed, but second command will fail, ie. machine not initialized
		timesCalled++
		if timesCalled == 2 {
			return exec.Command("false")
		}
		return exec.Command("true")
	}

	talosService := TalosService{}
	result := talosService.Ping("127.0.0.1")

	assert.True(t, result)
	assert.Equal(t, 2, timesCalled)
}

func Test_GenerateConfig_Succeeds(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.GenerateConfig(&helperService, "127.0.0.1", "test")

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_GenerateConfig_Fails_WithConfigExistsError(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "echo already exists && exit 1")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.GenerateConfig(&helperService, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.Equal(t, constants.ConfigExistsError, err)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_GenerateConfig_Fails_WithUnexpectedError(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.GenerateConfig(&helperService, "127.0.0.1", "test")

	assert.Error(t, err)
	assert.NotEqual(t, constants.ConfigExistsError, err)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
}

func Test_JoinCluster_Succeeds(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	err := talosService.JoinCluster(&helperService, "127.0.0.1", constants.TalosConfigFile)

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
}

func Test_JoinCluster_Fails_IfTalosctlFails(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	err := talosService.JoinCluster(&helperService, "127.0.0.1", constants.TalosConfigFile)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
}

func Test_BootstrapCluster_Succeeds_WaitsForBootstrapToSucceed(t *testing.T) {
	cmdCalls := 0
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		cmdCalls++
		if cmdCalls == 1 {
			return exec.Command("exit", "1")
		}
		return exec.Command("echo")
	}
	tenSeconds = time.Nanosecond
	fiveMinutes = time.Minute * 5

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	err := talosService.BootstrapCluster(&helperService, "127.0.0.1", "test")

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
	assert.Equal(t, 2, cmdCalls)
}

func Test_BootstrapCluster_Fails_AfterTimeout(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}
	tenSeconds = time.Nanosecond
	fiveMinutes = time.Microsecond

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	err := talosService.BootstrapCluster(&helperService, "127.0.0.1", "test")

	assert.Error(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
}

func Test_VerifyNodeHealth_Succeeds_WaitsForBootstrapToSucceed(t *testing.T) {
	cmdCalls := 0
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		cmdCalls++
		if cmdCalls == 1 {
			return exec.Command("exit", "1")
		}
		return exec.Command("echo")
	}
	tenSeconds = time.Nanosecond
	fiveMinutes = time.Minute * 5

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	err := talosService.VerifyNodeHealth(&helperService, "127.0.0.1", "test")

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
	assert.Equal(t, 2, cmdCalls)
}

func Test_VerifyNodeHealth_Fails_AfterTimeout(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}
	tenSeconds = time.Nanosecond
	fiveMinutes = time.Microsecond

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	err := talosService.VerifyNodeHealth(&helperService, "127.0.0.1", "test")

	assert.Error(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
}

func Test_GetDisks_Succeeds(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "echo disk1")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("DeleteEmptyStrings", []string{"disk1", ""}).Return([]string{"disk1"})

	talosService := TalosService{}
	disks, err := talosService.GetDisks(&helperService, "127.0.0.1")

	assert.NotNil(t, disks)
	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "DeleteEmptyStrings", 1)
}

func Test_GetDisks_Fails_IfTalosCtlFails(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}

	helperService := mocks.MockHelperService{}

	talosService := TalosService{}
	disks, err := talosService.GetDisks(&helperService, "127.0.0.1")

	assert.Nil(t, disks)
	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "DeleteEmptyStrings", 0)
}

func Test_GetNetworkInterface_Succeeds(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "echo eth0")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	networkInterface, err := talosService.GetNetworkInterface(&helperService, "127.0.0.1")

	assert.NotEmpty(t, networkInterface)
	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
}

func Test_GetNetworkInterface_Fails_IfTalosctlFails(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	networkInterface, err := talosService.GetNetworkInterface(&helperService, "127.0.0.1")

	assert.Empty(t, networkInterface)
	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
}

func Test_ModifyNetworkInterface_Succeeds_PreservesUnknownKeys(t *testing.T) {
	config := &map[interface{}]interface{}{
		"foo": "bar",
		"machine": map[interface{}]interface{}{
			"network": map[interface{}]interface{}{
				"foo": "bar",
			},
		},
	}

	configYaml, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err = talosService.ModifyNetworkInterface(&helperService, constants.ControlplaneConfigFile, "eth0")

	mapResult := make(map[interface{}]interface{})
	unmarshallErr := yaml.Unmarshal(mockOs.Calls[1].Arguments[1].([]byte), &mapResult)
	if unmarshallErr != nil {
		panic(unmarshallErr)
	}

	assert.Nil(t, err)
	assert.Equal(t, "eth0", mapResult["machine"].(map[interface{}]interface{})["network"].(map[interface{}]interface{})["interfaces"].([]interface{})[0].(map[interface{}]interface{})["interface"])
	assert.Equal(t, "bar", mapResult["machine"].(map[interface{}]interface{})["network"].(map[interface{}]interface{})["foo"])
	assert.Equal(t, "bar", mapResult["foo"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifyNetworkInterface_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte{}, nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyNetworkInterface(&helperService, constants.ControlplaneConfigFile, "eth0")

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyNetworkGateway_Succeeds_PreservesUnknownKeys(t *testing.T) {
	config := &map[interface{}]interface{}{
		"foo": "bar",
		"machine": map[interface{}]interface{}{
			"network": map[interface{}]interface{}{
				"foo": "bar",
			},
		},
	}

	configYaml, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err = talosService.ModifyNetworkGateway(&helperService, constants.ControlplaneConfigFile, "127.0.0.1")

	mapResult := make(map[interface{}]interface{})
	unmarshallErr := yaml.Unmarshal(mockOs.Calls[1].Arguments[1].([]byte), &mapResult)
	if unmarshallErr != nil {
		panic(unmarshallErr)
	}

	assert.Nil(t, err)
	assert.Equal(t, "0.0.0.0/0", mapResult["machine"].(map[interface{}]interface{})["network"].(map[interface{}]interface{})["interfaces"].([]interface{})[0].(map[interface{}]interface{})["routes"].([]interface{})[0].(map[interface{}]interface{})["network"])
	assert.Equal(t, "127.0.0.1", mapResult["machine"].(map[interface{}]interface{})["network"].(map[interface{}]interface{})["interfaces"].([]interface{})[0].(map[interface{}]interface{})["routes"].([]interface{})[0].(map[interface{}]interface{})["gateway"])
	assert.Equal(t, "bar", mapResult["foo"])
	assert.Equal(t, "bar", mapResult["machine"].(map[interface{}]interface{})["network"].(map[interface{}]interface{})["foo"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifyNetworkGateway_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("invalid yaml"), nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyNetworkGateway(&helperService, constants.ControlplaneConfigFile, "127.0.0.1")

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyNetworkNodeIp_Succeeds_PreservesUnknownKeys(t *testing.T) {
	config := &map[interface{}]interface{}{
		"foo": "bar",
		"machine": map[interface{}]interface{}{
			"network": map[interface{}]interface{}{
				"foo": "bar",
			},
		},
	}
	configYaml, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err = talosService.ModifyNetworkNodeIp(&helperService, constants.ControlplaneConfigFile, "127.0.0.1")

	mapResult := make(map[interface{}]interface{})
	unmarshallErr := yaml.Unmarshal(mockOs.Calls[1].Arguments[1].([]byte), &mapResult)
	if unmarshallErr != nil {
		panic(unmarshallErr)
	}

	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1", mapResult["machine"].(map[interface{}]interface{})["network"].(map[interface{}]interface{})["interfaces"].([]interface{})[0].(map[interface{}]interface{})["addresses"].([]interface{})[0])
	assert.Equal(t, "bar", mapResult["foo"])
	assert.Equal(t, "bar", mapResult["machine"].(map[interface{}]interface{})["network"].(map[interface{}]interface{})["foo"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifyNetworkNodeIp_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("invalid yaml"), nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyNetworkNodeIp(&helperService, constants.ControlplaneConfigFile, "127.0.0.1")

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyNetworkHostname_Succeeds_PreservesUnknownKeys(t *testing.T) {
	config := &map[interface{}]interface{}{
		"foo": "bar",
		"machine": map[interface{}]interface{}{
			"network": map[interface{}]interface{}{
				"foo": "bar",
			},
		},
	}

	configYaml, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err = talosService.ModifyNetworkHostname(&helperService, constants.ControlplaneConfigFile, "test-hostname")

	mapResult := make(map[interface{}]interface{})
	unmarshallErr := yaml.Unmarshal(mockOs.Calls[1].Arguments[1].([]byte), &mapResult)
	if unmarshallErr != nil {
		panic(unmarshallErr)
	}

	assert.Nil(t, err)
	assert.Equal(t, "test-hostname", mapResult["machine"].(map[interface{}]interface{})["network"].(map[interface{}]interface{})["hostname"])
	assert.Equal(t, "bar", mapResult["foo"])
	assert.Equal(t, "bar", mapResult["machine"].(map[interface{}]interface{})["network"].(map[interface{}]interface{})["foo"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifyNetworkHostname_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("invalid yaml"), nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyNetworkHostname(&helperService, constants.ControlplaneConfigFile, "127.0.0.1")

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyConfigDisk_Succeeds_PreservesUnknownKeys(t *testing.T) {
	config := &map[interface{}]interface{}{
		"foo": "bar",
		"machine": map[interface{}]interface{}{
			"install": map[interface{}]interface{}{
				"foo": "bar",
			},
		},
	}

	configYaml, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err = talosService.ModifyConfigDisk(&helperService, constants.ControlplaneConfigFile, "test-disk")

	mapResult := make(map[interface{}]interface{})
	unmarshallErr := yaml.Unmarshal(mockOs.Calls[1].Arguments[1].([]byte), &mapResult)
	if unmarshallErr != nil {
		panic(unmarshallErr)
	}

	assert.Nil(t, err)
	assert.Equal(t, "test-disk", mapResult["machine"].(map[interface{}]interface{})["install"].(map[interface{}]interface{})["disk"])
	assert.Equal(t, "bar", mapResult["foo"])
	assert.Equal(t, "bar", mapResult["machine"].(map[interface{}]interface{})["install"].(map[interface{}]interface{})["foo"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifyConfigDisk_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("invalid yaml"), nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyConfigDisk(&helperService, constants.ControlplaneConfigFile, "test-disk")

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifySchedulingOnControlPlane_Succeeds_PreservesUnknownKeys_WithTrueValue(t *testing.T) {
	config := &map[interface{}]interface{}{
		"foo": "bar",
		"cluster": map[interface{}]interface{}{
			"foo": "bar",
		},
	}

	configYaml, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err = talosService.ModifySchedulingOnControlPlane(&helperService, true)

	mapResult := make(map[interface{}]interface{})
	unmarshallErr := yaml.Unmarshal(mockOs.Calls[1].Arguments[1].([]byte), &mapResult)
	if unmarshallErr != nil {
		panic(unmarshallErr)
	}

	assert.Nil(t, err)
	assert.True(t, mapResult["cluster"].(map[interface{}]interface{})["allowSchedulingOnControlPlanes"].(bool))
	assert.Equal(t, "bar", mapResult["foo"])
	assert.Equal(t, "bar", mapResult["cluster"].(map[interface{}]interface{})["foo"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifySchedulingOnControlPlane_Succeeds_PreservesUnknownKeys_WithFalseValue(t *testing.T) {
	config := &map[interface{}]interface{}{
		"foo": "bar",
		"cluster": map[interface{}]interface{}{
			"foo": "bar",
		},
	}

	configYaml, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err = talosService.ModifySchedulingOnControlPlane(&helperService, false)

	mapResult := make(map[interface{}]interface{})
	unmarshallErr := yaml.Unmarshal(mockOs.Calls[1].Arguments[1].([]byte), &mapResult)
	if unmarshallErr != nil {
		panic(unmarshallErr)
	}

	assert.Nil(t, err)
	assert.False(t, mapResult["cluster"].(map[interface{}]interface{})["allowSchedulingOnControlPlanes"].(bool))
	assert.Equal(t, "bar", mapResult["foo"])
	assert.Equal(t, "bar", mapResult["cluster"].(map[interface{}]interface{})["foo"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifySchedulingOnControlPlane_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("invalid yaml"), nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifySchedulingOnControlPlane(&helperService, true)

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_GetControlPlaneIp_Succeeds(t *testing.T) {
	config := &models.TalosMachineConfig{}
	config.Cluster.ControlPlane.Endpoint = "https://127.0.0.1:6443"

	yaml, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(yaml, nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	endpoint, err := talosService.GetControlPlaneIp(&helperService, constants.ControlplaneConfigFile)

	assert.Nil(t, err)
	assert.Equal(t, "127.0.0.1", endpoint)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_GetControlPlaneIp_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("invalid yaml"), nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	endpoint, err := talosService.GetControlPlaneIp(&helperService, constants.ControlplaneConfigFile)

	assert.NotNil(t, err)
	assert.Empty(t, endpoint)
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_DownloadKubeConfig_Succeeds(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	err := talosService.DownloadKubeConfig(&helperService, "127.0.0.1", "0.0.0.0")

	assert.Nil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
}

func Test_DownloadKubeConfig_Fails_IfTalosCtlFails(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	err := talosService.DownloadKubeConfig(&helperService, "127.0.0.1", "0.0.0.0")

	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
}
