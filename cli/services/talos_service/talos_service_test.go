package talos_service

import (
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v2"
)

// buildMultiDocYaml marshals each doc and joins them the same way writeDocuments does,
// to build fixtures matching Talos's multi-document config format.
func buildMultiDocYaml(t *testing.T, docs ...map[interface{}]interface{}) []byte {
	t.Helper()

	parts := make([]string, 0, len(docs))
	for _, doc := range docs {
		marshaled, err := yaml.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		parts = append(parts, string(marshaled))
	}

	return []byte(strings.Join(parts, "---\n"))
}

// decodeWrittenDocuments splits a writeDocuments-produced byte slice back into its
// individual documents for assertions.
func decodeWrittenDocuments(t *testing.T, raw []byte) []map[interface{}]interface{} {
	t.Helper()

	parts := strings.Split(string(raw), "---\n")
	documents := make([]map[interface{}]interface{}, 0, len(parts))
	for _, part := range parts {
		var document map[interface{}]interface{}
		if err := yaml.Unmarshal([]byte(part), &document); err != nil {
			t.Fatal(err)
		}
		documents = append(documents, document)
	}

	return documents
}

// assertNoLeakedModelKeys guards against the typed models serialising their own internals
// into the config: the inlined Unmapped fields must flatten away, at every nesting level.
func assertNoLeakedModelKeys(t *testing.T, value interface{}) {
	t.Helper()

	switch typed := value.(type) {
	case map[interface{}]interface{}:
		for key, nested := range typed {
			assert.NotEqual(t, "unmapped", key)
			assertNoLeakedModelKeys(t, nested)
		}
	case []interface{}:
		for _, nested := range typed {
			assertNoLeakedModelKeys(t, nested)
		}
	}
}

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

func Test_WarnIfTalosVersionMismatch_DoesNotPanic_WhenVersionsMatch(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "echo 'Client:' && echo 'Talos v1.14.1'")
	}

	talosService := TalosService{}
	assert.NotPanics(t, func() {
		talosService.WarnIfTalosVersionMismatch("v1.14.1")
	})
}

func Test_WarnIfTalosVersionMismatch_DoesNotPanic_WhenVersionsDiffer(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "echo 'Client:' && echo 'Talos v1.14.1'")
	}

	talosService := TalosService{}
	assert.NotPanics(t, func() {
		talosService.WarnIfTalosVersionMismatch("v1.9.0")
	})
}

func Test_WarnIfTalosVersionMismatch_DoesNotPanic_WhenLocalVersionFails(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}

	talosService := TalosService{}
	assert.NotPanics(t, func() {
		talosService.WarnIfTalosVersionMismatch("v1.14.1")
	})
}

func Test_GetLocalVersion_Succeeds(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "echo 'Client:' && echo 'Talos v1.14.1'")
	}

	talosService := TalosService{}
	version, err := talosService.GetLocalVersion()

	assert.Nil(t, err)
	assert.Equal(t, "v1.14.1", version)
}

func Test_GetLocalVersion_Fails_IfTalosctlFails(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}

	talosService := TalosService{}
	version, err := talosService.GetLocalVersion()

	assert.NotNil(t, err)
	assert.Empty(t, version)
}

func Test_GetLocalVersion_Fails_IfOutputUnparsable(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("bash", "-c", "echo 'garbage output'")
	}

	talosService := TalosService{}
	version, err := talosService.GetLocalVersion()

	assert.NotNil(t, err)
	assert.Empty(t, version)
}

func Test_GenerateConfig_Succeeds(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.GenerateConfig(&helperService, "127.0.0.1", "test", "v1.14.1")

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
	err := talosService.GenerateConfig(&helperService, "127.0.0.1", "test", "v1.14.1")

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
	err := talosService.GenerateConfig(&helperService, "127.0.0.1", "test", "v1.14.1")

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

// trimmed from real "talosctl get disks -o yaml" output: one document per disk, including the
// read-only loop devices backing the running system and the USB stick the installer booted from
const disksOutput = `node: 192.168.1.50
metadata:
    namespace: runtime
    type: Disks.block.talos.dev
    id: loop0
spec:
    dev_path: /dev/loop0
    pretty_size: 4.1 kB
    readonly: true
    cdrom: false
---
node: 192.168.1.50
metadata:
    namespace: runtime
    type: Disks.block.talos.dev
    id: nvme0n1
spec:
    dev_path: /dev/nvme0n1
    pretty_size: 256 GB
    readonly: false
    cdrom: false
    model: PC300 NVMe SK hynix 256GB
    transport: nvme
---
node: 192.168.1.50
metadata:
    namespace: runtime
    type: Disks.block.talos.dev
    id: sda
spec:
    dev_path: /dev/sda
    pretty_size: 8.1 GB
    readonly: false
    cdrom: false
    model: ProductCode
    transport: usb
---
node: 192.168.1.50
metadata:
    namespace: runtime
    type: Disks.block.talos.dev
    id: sr0
spec:
    dev_path: /dev/sr0
    pretty_size: 1.1 GB
    readonly: false
    cdrom: true
`

func Test_GetDisks_Succeeds_ReturnsOnlyInstallableDisks(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", disksOutput)
	}

	talosService := TalosService{}
	disks, err := talosService.GetDisks("192.168.1.50")

	assert.Nil(t, err)
	// the read-only loop device and the CD-ROM are not installable
	assert.Len(t, disks, 2)
	assert.Equal(t, "/dev/nvme0n1", disks[0].Spec.DevPath)
	assert.Equal(t, "256 GB", disks[0].Spec.PrettySize)
	assert.Equal(t, "PC300 NVMe SK hynix 256GB", disks[0].Spec.Model)
	assert.Equal(t, "nvme", disks[0].Spec.Transport)
	assert.Equal(t, "/dev/sda", disks[1].Spec.DevPath)
	assert.Equal(t, "usb", disks[1].Spec.Transport)
}

func Test_GetDisks_Succeeds_WithNoInstallableDisks(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", "spec:\n    dev_path: /dev/loop0\n    readonly: true\n")
	}

	talosService := TalosService{}
	disks, err := talosService.GetDisks("192.168.1.50")

	assert.Nil(t, err)
	assert.Empty(t, disks)
}

func Test_GetDisks_Fails_IfOutputUnparsable(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", "spec: [not a map")
	}

	talosService := TalosService{}
	disks, err := talosService.GetDisks("192.168.1.50")

	assert.NotNil(t, err)
	assert.Nil(t, disks)
}

func Test_GetDisks_Fails_IfTalosCtlFails(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}

	talosService := TalosService{}
	disks, err := talosService.GetDisks("192.168.1.50")

	assert.Nil(t, disks)
	assert.NotNil(t, err)
}

// realistic "talosctl get addresses" output: talosctl repeats the node IP in the NODE
// column of every row, and each link appears more than once (IPv4 plus link-local IPv6).
const addressesOutput = `NODE            NAMESPACE   TYPE            ID                                      VERSION   ADDRESS                                 LINK
192.168.1.50    network     AddressStatus   enp57s0u1u1/192.168.1.50/24             1         192.168.1.50/24                         enp57s0u1u1
192.168.1.50    network     AddressStatus   enp57s0u1u1/fe80::1e69:7aff:fe2d:1/64   2         fe80::1e69:7aff:fe2d:1/64               enp57s0u1u1
192.168.1.50    network     AddressStatus   lo/127.0.0.1/8                          1         127.0.0.1/8                             lo
192.168.1.50    network     AddressStatus   lo/::1/128                              1         ::1/128                                 lo
`

func Test_GetNetworkInterface_Succeeds(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", addressesOutput)
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	networkInterface, err := talosService.GetNetworkInterface(&helperService, "192.168.1.50")

	assert.Nil(t, err)
	assert.Equal(t, "enp57s0u1u1", networkInterface)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
}

// the old implementation matched whole lines, so the node IP in the NODE column matched
// every row and it returned every link joined by newlines - which Talos then silently
// ignored, leaving the node with no address at all
func Test_GetNetworkInterface_Succeeds_ReturnsSingleLinkNotEveryLink(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", addressesOutput)
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	networkInterface, err := talosService.GetNetworkInterface(&helperService, "192.168.1.50")

	assert.Nil(t, err)
	assert.NotContains(t, networkInterface, "\n")
	assert.NotContains(t, networkInterface, "lo")
}

func Test_GetNetworkInterface_Fails_IfNodeIpNotPresent(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", addressesOutput)
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	networkInterface, err := talosService.GetNetworkInterface(&helperService, "10.0.0.1")

	assert.NotNil(t, err)
	assert.Empty(t, networkInterface)
}

// a node IP that is a string prefix of another address on the machine must not match it
func Test_ParseNetworkInterface_DoesNotMatchLongerAddress(t *testing.T) {
	output := `NODE           NAMESPACE   TYPE            ID                            VERSION   ADDRESS            LINK
192.168.1.16   network     AddressStatus   enp0s31f6/192.168.1.161/24    1         192.168.1.161/24   enp0s31f6
192.168.1.16   network     AddressStatus   eth1/192.168.1.16/24          1         192.168.1.16/24    eth1
`

	networkInterface, err := parseNetworkInterface(output, "192.168.1.16")

	assert.Nil(t, err)
	assert.Equal(t, "eth1", networkInterface)
}

func Test_GetNetworkInterface_Fails_IfTalosctlFails(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("exit", "1")
	}

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigFilePath", constants.TalosConfigFile).Return("test")

	talosService := TalosService{}
	networkInterface, err := talosService.GetNetworkInterface(&helperService, "192.168.1.50")

	assert.Empty(t, networkInterface)
	assert.NotNil(t, err)
	helperService.AssertNumberOfCalls(t, "GetConfigFilePath", 1)
}

func Test_ModifyNetworkNodeIp_Succeeds_CreatesLinkConfigWhenMissing(t *testing.T) {
	configYaml := buildMultiDocYaml(t, map[interface{}]interface{}{"foo": "bar"})

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyNetworkNodeIp(&helperService, constants.ControlplaneConfigFile, "192.168.1.161")

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))
	for _, document := range documents {
		assertNoLeakedModelKeys(t, document)
	}

	assert.Nil(t, err)
	assert.Equal(t, "bar", documents[0]["foo"])
	assert.Equal(t, models.TalosLinkConfigKind, documents[1]["kind"])
	assert.Equal(t, models.TalosConfigApiVersion, documents[1]["apiVersion"])
	assert.Equal(t, "192.168.1.161/24", documents[1]["addresses"].([]interface{})[0].(map[interface{}]interface{})["address"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
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
	err := talosService.ModifyNetworkNodeIp(&helperService, constants.ControlplaneConfigFile, "192.168.1.161")

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyNetworkInterface_Succeeds_PreservesUnknownKeys(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{"foo": "bar"},
		map[interface{}]interface{}{"kind": models.TalosLinkConfigKind, "mtu": 1500},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyNetworkInterface(&helperService, constants.ControlplaneConfigFile, "enp57s0u1u1")

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))
	for _, document := range documents {
		assertNoLeakedModelKeys(t, document)
	}

	assert.Nil(t, err)
	assert.Equal(t, "bar", documents[0]["foo"])
	assert.Equal(t, "enp57s0u1u1", documents[1]["name"])
	assert.Equal(t, 1500, documents[1]["mtu"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifyNetworkInterface_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("invalid yaml"), nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyNetworkInterface(&helperService, constants.ControlplaneConfigFile, "eth0")

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyNetworkGateway_Succeeds_PreservesUnknownKeys(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{"foo": "bar"},
		map[interface{}]interface{}{"kind": models.TalosLinkConfigKind, "name": "eth0"},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyNetworkGateway(&helperService, constants.ControlplaneConfigFile, "192.168.1.1")

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))
	for _, document := range documents {
		assertNoLeakedModelKeys(t, document)
	}

	assert.Nil(t, err)
	assert.Equal(t, "bar", documents[0]["foo"])
	assert.Equal(t, "eth0", documents[1]["name"])
	assert.Equal(t, "192.168.1.1", documents[1]["routes"].([]interface{})[0].(map[interface{}]interface{})["gateway"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
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
	err := talosService.ModifyNetworkGateway(&helperService, constants.ControlplaneConfigFile, "192.168.1.1")

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyNetworkHostname_Succeeds_PreservesUnknownKeys(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{"foo": "bar"},
		map[interface{}]interface{}{"kind": models.TalosHostnameConfigKind, "auto": "stable"},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyNetworkHostname(&helperService, constants.ControlplaneConfigFile, "test-hostname")

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))
	for _, document := range documents {
		assertNoLeakedModelKeys(t, document)
	}

	assert.Nil(t, err)
	assert.Equal(t, "bar", documents[0]["foo"])
	assert.Equal(t, "test-hostname", documents[1]["hostname"])
	assert.NotContains(t, documents[1], "auto")
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
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
	err := talosService.ModifyNetworkHostname(&helperService, constants.ControlplaneConfigFile, "test-hostname")

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyNetworkHostname_Fails_IfHostnameConfigNotFound(t *testing.T) {
	configYaml := buildMultiDocYaml(t, map[interface{}]interface{}{"foo": "bar"})

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyNetworkHostname(&helperService, constants.ControlplaneConfigFile, "test-hostname")

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 0)

	osReadFile = os.ReadFile
}

func Test_ModifyConfigDisk_Succeeds_PreservesUnknownKeys(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{"foo": "bar"},
		map[interface{}]interface{}{
			"kind": models.TalosUnattendedInstallConfigKind,
			"provisioning": map[interface{}]interface{}{
				"wipe": true,
				"diskSelector": map[interface{}]interface{}{
					"match": `disk.dev_path == "/dev/sda"`,
				},
			},
		},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyConfigDisk(&helperService, constants.ControlplaneConfigFile, "/dev/sdb")

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))
	for _, document := range documents {
		assertNoLeakedModelKeys(t, document)
	}
	provisioning := documents[1]["provisioning"].(map[interface{}]interface{})
	diskSelector := provisioning["diskSelector"].(map[interface{}]interface{})

	assert.Nil(t, err)
	assert.Equal(t, "bar", documents[0]["foo"])
	assert.Equal(t, `disk.dev_path == "/dev/sdb"`, diskSelector["match"])
	assert.Equal(t, true, provisioning["wipe"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
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
	err := talosService.ModifyConfigDisk(&helperService, constants.ControlplaneConfigFile, "/dev/sdb")

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyConfigDisk_Fails_IfUnattendedInstallConfigNotFound(t *testing.T) {
	configYaml := buildMultiDocYaml(t, map[interface{}]interface{}{"foo": "bar"})

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyConfigDisk(&helperService, constants.ControlplaneConfigFile, "/dev/sdb")

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 0)

	osReadFile = os.ReadFile
}

func Test_ModifySchedulingOnControlPlane_Succeeds_RemovesTaint_WithTrueValue(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{"foo": "bar"},
		map[interface{}]interface{}{
			"kind":   models.TalosKubeNodeConfigKind,
			"labels": map[interface{}]interface{}{"foo": "bar"},
			"taints": map[interface{}]interface{}{models.TalosControlPlaneTaint: "NoSchedule"},
		},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifySchedulingOnControlPlane(&helperService, true)

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))
	for _, document := range documents {
		assertNoLeakedModelKeys(t, document)
	}

	assert.Nil(t, err)
	assert.Equal(t, "bar", documents[0]["foo"])
	assert.Empty(t, documents[1]["taints"])
	assert.Equal(t, "bar", documents[1]["labels"].(map[interface{}]interface{})["foo"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifySchedulingOnControlPlane_Succeeds_RestoresTaint_WithFalseValue(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{"foo": "bar"},
		map[interface{}]interface{}{"kind": models.TalosKubeNodeConfigKind, "taints": map[interface{}]interface{}{}},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifySchedulingOnControlPlane(&helperService, false)

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))

	assert.Nil(t, err)
	assert.Equal(t, "NoSchedule", documents[1]["taints"].(map[interface{}]interface{})[models.TalosControlPlaneTaint])
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
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifySchedulingOnControlPlane_Fails_IfKubeNodeConfigNotFound(t *testing.T) {
	configYaml := buildMultiDocYaml(t, map[interface{}]interface{}{"foo": "bar"})

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifySchedulingOnControlPlane(&helperService, true)

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 0)

	osReadFile = os.ReadFile
}

func Test_ModifyDnsServer_Succeeds_PreservesUnknownKeys(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{"version": "v1alpha1"},
		map[interface{}]interface{}{
			"kind": models.TalosResolverConfigKind,
			"hostDNS": map[interface{}]interface{}{
				"enabled": true,
			},
		},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyDnsServer(&helperService, constants.ControlplaneConfigFile, "192.168.1.1")

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))
	for _, document := range documents {
		assertNoLeakedModelKeys(t, document)
	}

	assert.Nil(t, err)
	assert.Equal(t, "v1alpha1", documents[0]["version"])
	assert.Equal(t, "192.168.1.1", documents[1]["nameservers"].([]interface{})[0].(map[interface{}]interface{})["address"])
	// the caching resolver settings must not be clobbered by setting nameservers
	assert.Equal(t, true, documents[1]["hostDNS"].(map[interface{}]interface{})["enabled"])
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifyDnsServer_Succeeds_CreatesResolverConfigWhenMissing(t *testing.T) {
	configYaml := buildMultiDocYaml(t, map[interface{}]interface{}{"version": "v1alpha1"})

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyDnsServer(&helperService, constants.ControlplaneConfigFile, "192.168.1.1")

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))

	assert.Nil(t, err)
	assert.Len(t, documents, 2)
	assert.Equal(t, models.TalosResolverConfigKind, documents[1]["kind"])
	assert.Equal(t, "192.168.1.1", documents[1]["nameservers"].([]interface{})[0].(map[interface{}]interface{})["address"])

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifyDnsServer_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("invalid yaml"), nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyDnsServer(&helperService, constants.ControlplaneConfigFile, "192.168.1.1")

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyTimeServer_Succeeds_PreservesUnknownKeys(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{
			"version": "v1alpha1",
			"machine": map[interface{}]interface{}{
				"type":  "controlplane",
				"token": "secret-token",
			},
		},
		map[interface{}]interface{}{"kind": models.TalosHostnameConfigKind, "auto": "stable"},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyTimeServer(&helperService, constants.ControlplaneConfigFile, "192.168.1.1")

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))
	for _, document := range documents {
		assertNoLeakedModelKeys(t, document)
	}
	machine := documents[0]["machine"].(map[interface{}]interface{})

	assert.Nil(t, err)
	assert.Equal(t, "192.168.1.1", machine["time"].(map[interface{}]interface{})["servers"].([]interface{})[0])
	// the machine's identity must survive being round-tripped through the model
	assert.Equal(t, "secret-token", machine["token"])
	assert.Equal(t, "controlplane", machine["type"])
	assert.Equal(t, "v1alpha1", documents[0]["version"])
	assert.Equal(t, "stable", documents[1]["auto"])
	helperService.AssertNumberOfCalls(t, "GetConfigDir", 1)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 1)

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifyTimeServer_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("invalid yaml"), nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyTimeServer(&helperService, constants.ControlplaneConfigFile, "192.168.1.1")

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "ReadFile", 1)

	osReadFile = os.ReadFile
}

func Test_ModifyTimeServer_Fails_IfPrimaryDocumentMissing(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{"kind": models.TalosHostnameConfigKind, "auto": "stable"},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyTimeServer(&helperService, constants.ControlplaneConfigFile, "192.168.1.1")

	assert.NotNil(t, err)
	mockOs.AssertNumberOfCalls(t, "WriteFile", 0)

	osReadFile = os.ReadFile
}

func Test_FindPrimaryDocument_ReturnsDocumentWithoutKind(t *testing.T) {
	documents := []map[string]interface{}{
		{"kind": models.TalosHostnameConfigKind},
		{"version": "v1alpha1"},
	}

	document, index := findPrimaryDocument(documents)

	assert.Equal(t, 1, index)
	assert.Equal(t, "v1alpha1", document["version"])
}

func Test_FindPrimaryDocument_ReturnsNegativeIndex_WhenNotFound(t *testing.T) {
	documents := []map[string]interface{}{
		{"kind": models.TalosHostnameConfigKind},
	}

	document, index := findPrimaryDocument(documents)

	assert.Equal(t, -1, index)
	assert.Nil(t, document)
}

func Test_GetControlPlaneIp_Succeeds(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{"foo": "bar"},
		map[interface{}]interface{}{"kind": "KubeClusterConfig", "endpoint": "https://127.0.0.1:6443"},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
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

func Test_GetControlPlaneIp_Fails_IfKubeClusterConfigNotFound(t *testing.T) {
	configYaml := buildMultiDocYaml(t, map[interface{}]interface{}{"foo": "bar"})

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	endpoint, err := talosService.GetControlPlaneIp(&helperService, constants.ControlplaneConfigFile)

	assert.NotNil(t, err)
	assert.Empty(t, endpoint)

	osReadFile = os.ReadFile
}

func Test_GetParsedDocuments_Succeeds_ReturnsAllDocumentsInOrder(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{"foo": "bar"},
		map[interface{}]interface{}{"kind": "HostnameConfig", "hostname": "test"},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	osReadFile = mockOs.ReadFile

	documents, err := getParsedDocuments("test", constants.ControlplaneConfigFile)

	assert.Nil(t, err)
	assert.Len(t, documents, 2)
	assert.Equal(t, "bar", documents[0]["foo"])
	assert.Equal(t, "HostnameConfig", documents[1]["kind"])

	osReadFile = os.ReadFile
}

func Test_GetParsedDocuments_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("foo: [invalid"), nil)
	osReadFile = mockOs.ReadFile

	documents, err := getParsedDocuments("test", constants.ControlplaneConfigFile)

	assert.NotNil(t, err)
	assert.Nil(t, documents)

	osReadFile = os.ReadFile
}

func Test_WriteDocuments_Succeeds_JoinsDocumentsWithSeparator(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osWriteFile = mockOs.WriteFile

	documents := []map[string]interface{}{
		{"foo": "bar"},
		{"kind": "HostnameConfig", "hostname": "test"},
	}

	err := writeDocuments("test", constants.ControlplaneConfigFile, documents)

	assert.Nil(t, err)
	written := string(mockOs.Calls[0].Arguments[1].([]byte))
	assert.Contains(t, written, "foo: bar")
	assert.Contains(t, written, "---\n")
	assert.Contains(t, written, "kind: HostnameConfig")

	osWriteFile = os.WriteFile
}

func Test_FindDocument_ReturnsMatchingKind(t *testing.T) {
	documents := []map[string]interface{}{
		{"version": "v1alpha1"},
		{"kind": models.TalosHostnameConfigKind, "hostname": "test"},
	}

	document, index := findDocument(documents, models.TalosHostnameConfigKind)

	assert.Equal(t, 1, index)
	assert.Equal(t, "test", document["hostname"])
}

func Test_FindDocument_ReturnsNegativeIndex_WhenNotFound(t *testing.T) {
	documents := []map[string]interface{}{
		{"version": "v1alpha1"},
	}

	document, index := findDocument(documents, models.TalosHostnameConfigKind)

	assert.Equal(t, -1, index)
	assert.Nil(t, document)
}

func Test_GetDocument_DecodesIntoTarget(t *testing.T) {
	documents := []map[string]interface{}{
		{"version": "v1alpha1"},
		{"kind": models.TalosHostnameConfigKind, "hostname": "test", "foo": "bar"},
	}

	var hostnameConfig models.TalosHostnameConfig
	index, err := getDocument(documents, models.TalosHostnameConfigKind, &hostnameConfig)

	assert.Nil(t, err)
	assert.Equal(t, 1, index)
	assert.Equal(t, "test", hostnameConfig.Hostname)
	assert.Equal(t, "bar", hostnameConfig.Unmapped["foo"])
}

func Test_GetDocument_Fails_WhenDocumentMissing(t *testing.T) {
	documents := []map[string]interface{}{
		{"version": "v1alpha1"},
	}

	var hostnameConfig models.TalosHostnameConfig
	index, err := getDocument(documents, models.TalosHostnameConfigKind, &hostnameConfig)

	assert.NotNil(t, err)
	assert.Equal(t, -1, index)
}

func Test_GetOrCreateDocument_ReturnsExisting_WhenFound(t *testing.T) {
	documents := []map[string]interface{}{
		{"kind": models.TalosLinkConfigKind, "name": "eth0"},
	}

	var linkConfig models.TalosLinkConfig
	index, err := getOrCreateDocument(&documents, models.TalosLinkConfigKind, &linkConfig)

	assert.Nil(t, err)
	assert.Equal(t, 0, index)
	assert.Equal(t, "eth0", linkConfig.Name)
	assert.Len(t, documents, 1)
}

func Test_GetOrCreateDocument_AppendsNew_WhenNotFound(t *testing.T) {
	documents := []map[string]interface{}{
		{"version": "v1alpha1"},
	}

	var linkConfig models.TalosLinkConfig
	index, err := getOrCreateDocument(&documents, models.TalosLinkConfigKind, &linkConfig)

	assert.Nil(t, err)
	assert.Equal(t, 1, index)
	assert.Equal(t, models.TalosLinkConfigKind, linkConfig.Kind)
	assert.Equal(t, models.TalosConfigApiVersion, linkConfig.ApiVersion)
	assert.Len(t, documents, 2)
}

func Test_SetDocument_EncodesOverDocumentAtIndex(t *testing.T) {
	documents := []map[string]interface{}{
		{"version": "v1alpha1"},
		{"kind": models.TalosHostnameConfigKind, "auto": "stable"},
	}

	hostnameConfig := models.TalosHostnameConfig{
		Kind:     models.TalosHostnameConfigKind,
		Hostname: "test",
		Unmapped: map[string]interface{}{"foo": "bar"},
	}

	err := setDocument(documents, 1, hostnameConfig)

	assert.Nil(t, err)
	assert.Equal(t, "v1alpha1", documents[0]["version"])
	assert.Equal(t, "test", documents[1]["hostname"])
	assert.Equal(t, "bar", documents[1]["foo"])
	assert.NotContains(t, documents[1], "auto")
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

func Test_ModifyUserVolumes_Succeeds_AddingOnlyMissingVolumes(t *testing.T) {
	configYaml := buildMultiDocYaml(t,
		map[interface{}]interface{}{
			"version": "v1alpha1",
			"machine": map[interface{}]interface{}{"token": "secret-token"},
		},
		map[interface{}]interface{}{"apiVersion": "v1alpha1", "kind": models.TalosUserVolumeConfigKind, "name": "other", "volumeType": "directory"},
		map[interface{}]interface{}{"kind": models.TalosHostnameConfigKind, "auto": "stable"},
	)

	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return(configYaml, nil)
	mockOs.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	osReadFile = mockOs.ReadFile
	osWriteFile = mockOs.WriteFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyUserVolumes(&helperService, constants.WorkerConfigFile, []string{"longhorn", "other"})

	documents := decodeWrittenDocuments(t, mockOs.Calls[1].Arguments[1].([]byte))
	for _, document := range documents {
		assertNoLeakedModelKeys(t, document)
	}

	assert.Nil(t, err)
	assert.Len(t, documents, 4)
	assert.Equal(t, "secret-token", documents[0]["machine"].(map[interface{}]interface{})["token"])
	assert.Equal(t, "other", documents[1]["name"])
	assert.Equal(t, "stable", documents[2]["auto"])
	assert.Equal(t, map[interface{}]interface{}{"apiVersion": "v1alpha1", "kind": "UserVolumeConfig", "name": "longhorn", "volumeType": "directory"}, documents[3])

	osReadFile = os.ReadFile
	osWriteFile = os.WriteFile
}

func Test_ModifyUserVolumes_Fails_IfConfigNotValid(t *testing.T) {
	mockOs := mocks.MockOs{}
	mockOs.On("ReadFile", mock.Anything).Return([]byte("invalid yaml"), nil)
	osReadFile = mockOs.ReadFile

	helperService := mocks.MockHelperService{}
	helperService.On("GetConfigDir").Return("test")

	talosService := TalosService{}
	err := talosService.ModifyUserVolumes(&helperService, constants.WorkerConfigFile, []string{"longhorn"})

	assert.NotNil(t, err)

	osReadFile = os.ReadFile
}
