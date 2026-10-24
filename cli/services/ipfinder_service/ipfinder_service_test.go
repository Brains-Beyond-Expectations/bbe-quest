package ipfinder_service

import (
	"errors"
	"net"
	"os"
	"os/exec"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_Locate_Device_Successfully(t *testing.T) {
	ipFinderService := IpFinderService{}

	helperMock := new(mocks.MockHelperService)
	talosMock := new(mocks.MockTalosService)

	inputIp := "192.168.1.1"
	helperMock.On("DeleteEmptyStrings", mock.Anything).Return([]string{inputIp, "justARandomIp"})
	talosMock.On("Ping", mock.Anything, mock.Anything).Return(true)

	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", `something\nelse\n`)
	}

	ips, err := ipFinderService.LocateDevice(helperMock, talosMock, inputIp)

	assert.NoError(t, err)
	assert.Equal(t, inputIp, ips[0])
}

func Test_Locate_Device_Not_Found(t *testing.T) {
	ipFinderService := IpFinderService{}

	helperMock := new(mocks.MockHelperService)
	talosMock := new(mocks.MockTalosService)

	inputIp := "192.168.1.1"
	helperMock.On("DeleteEmptyStrings", mock.Anything).Return([]string{inputIp, "justARandomIp"})
	talosMock.On("Ping", mock.Anything, mock.Anything).Return(false)

	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", `something\nelse\n`)
	}

	ips, err := ipFinderService.LocateDevice(helperMock, talosMock, inputIp)

	assert.NoError(t, err)
	assert.Equal(t, 0, len(ips))
}
func Test_Locate_Device_Failed(t *testing.T) {
	ipFinderService := IpFinderService{}

	helperMock := new(mocks.MockHelperService)
	talosMock := new(mocks.MockTalosService)

	inputIp := "192.168.1.1"

	execCommand = func(_ string, _ ...string) *exec.Cmd {
		cmd := exec.Command("echo", "mocked response")
		cmd.Stderr = os.Stderr
		return cmd
	}

	ips, err := ipFinderService.LocateDevice(helperMock, talosMock, inputIp)

	assert.Error(t, err)
	assert.Equal(t, []string([]string(nil)), ips)
}

func Test_GetIp_Wsl(t *testing.T) {
	ipFinderService := IpFinderService{}

	helperMock := new(mocks.MockHelperService)
	helperMock.On("IsWsl").Return(true)

	mockedIpResult := "1.1.1.1"
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", mockedIpResult)
	}

	ip, err := ipFinderService.GetGatewayIp(helperMock)

	assert.NoError(t, err)
	assert.Equal(t, mockedIpResult, ip)
}

func Test_GetGatewayIp_Non_WSL(t *testing.T) {
	ipFinderService := IpFinderService{}

	helperMock := new(mocks.MockHelperService)
	helperMock.On("IsWsl").Return(false)

	mockedIpResult := "1.1.1.1"
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", mockedIpResult)
	}

	ip, err := ipFinderService.GetGatewayIp(helperMock)

	assert.NoError(t, err)
	assert.Equal(t, mockedIpResult, ip)
}

func Test_GetLocalIp_Succeeds(t *testing.T) {
	ip, err := IpFinderService{}.GetLocalIp("127.0.0.1")

	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1", ip)
}

func Test_GetLocalIp_Fails_WhenNoRouteIsFound(t *testing.T) {
	netDial = func(_ string, _ string) (net.Conn, error) {
		return nil, errors.New("network is unreachable")
	}
	defer func() { netDial = net.Dial }()

	ip, err := IpFinderService{}.GetLocalIp("192.168.1.161")

	assert.EqualError(t, err, "network is unreachable")
	assert.Equal(t, "", ip)
}
