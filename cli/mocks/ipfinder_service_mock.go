package mocks

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/interfaces"
	"github.com/stretchr/testify/mock"
)

type MockIpFinderService struct {
	mock.Mock
}

func (mock *MockIpFinderService) LocateDevice(helperService interfaces.HelperServiceInterface, talosService interfaces.TalosServiceInterface, ip string) ([]string, error) {
	args := mock.Called(helperService, talosService, ip)
	return args.Get(0).([]string), args.Error(1)
}

func (mock *MockIpFinderService) GetGatewayIp(helperService interfaces.HelperServiceInterface) (string, error) {
	args := mock.Called(helperService)
	return args.String(0), args.Error(1)
}
