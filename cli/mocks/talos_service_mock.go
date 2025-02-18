package mocks

import (
	"os/exec"

	"github.com/Brains-Beyond-Expectations/bbe-quest/interfaces"
	"github.com/stretchr/testify/mock"
)

var execCommand = exec.Command

type MockTalosService struct {
	mock.Mock
}

func (m *MockTalosService) Ping(nodeIp string) bool {
	args := m.Called(execCommand, nodeIp)

	return args.Get(0).(bool)
}

func (m *MockTalosService) GenerateConfig(helperService interfaces.HelperServiceInterface, controlPlaneIp string, clusterName string) error {
	args := m.Called(helperService, controlPlaneIp, clusterName)
	return args.Error(0)
}

func (m *MockTalosService) JoinCluster(helperService interfaces.HelperServiceInterface, nodeIp string, nodeConfigFile string) error {
	args := m.Called(helperService, nodeIp, nodeConfigFile)
	return args.Error(0)
}

func (m *MockTalosService) BootstrapCluster(helperService interfaces.HelperServiceInterface, nodeIp string, controlPlaneIp string) error {
	args := m.Called(helperService, nodeIp, controlPlaneIp)
	return args.Error(0)
}

func (m *MockTalosService) VerifyNodeHealth(helperService interfaces.HelperServiceInterface, nodeIp string, controlPlaneIp string) error {
	args := m.Called(helperService, nodeIp, controlPlaneIp)
	return args.Error(0)
}

func (m *MockTalosService) GetDisks(helperService interfaces.HelperServiceInterface, nodeIp string) ([]string, error) {
	args := m.Called(helperService, nodeIp)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockTalosService) GetNetworkInterface(helperService interfaces.HelperServiceInterface, nodeIp string) (string, error) {
	args := m.Called(helperService, nodeIp)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockTalosService) ModifyNetworkInterface(helperService interfaces.HelperServiceInterface, configFile string, networkInterfaceName string) error {
	args := m.Called(helperService, configFile, networkInterfaceName)
	return args.Error(0)
}

func (m *MockTalosService) ModifyNetworkGateway(helperService interfaces.HelperServiceInterface, configFile string, gatewayIp string) error {
	args := m.Called(helperService, configFile, gatewayIp)
	return args.Error(0)
}

func (m *MockTalosService) ModifyNetworkNodeIp(helperService interfaces.HelperServiceInterface, configFile string, nodeIp string) error {
	args := m.Called(helperService, configFile, nodeIp)
	return args.Error(0)
}

func (m *MockTalosService) ModifyNetworkHostname(helperService interfaces.HelperServiceInterface, configFile string, hostname string) error {
	args := m.Called(helperService, configFile, hostname)
	return args.Error(0)
}

func (m *MockTalosService) ModifyConfigDisk(helperService interfaces.HelperServiceInterface, configFile string, disk string) error {
	args := m.Called(helperService, configFile, disk)
	return args.Error(0)
}

func (m *MockTalosService) ModifySchedulingOnControlPlane(helperService interfaces.HelperServiceInterface, allowScheduling bool) error {
	args := m.Called(helperService, allowScheduling)
	return args.Error(0)
}

func (m *MockTalosService) GetControlPlaneIp(helperService interfaces.HelperServiceInterface, configFile string) (string, error) {
	args := m.Called(helperService, configFile)
	return args.Get(0).(string), args.Error(1)
}

func (m *MockTalosService) DownloadKubeConfig(helperService interfaces.HelperServiceInterface, nodeIp string, controlPlaneIp string) error {
	args := m.Called(helperService, nodeIp, controlPlaneIp)
	return args.Error(0)
}
