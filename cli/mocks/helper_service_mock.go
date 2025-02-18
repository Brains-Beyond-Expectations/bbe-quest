package mocks

import (
	"os/exec"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockHelperService struct {
	mock.Mock
}

func (mock *MockHelperService) CheckIfFileExists(file string) (*time.Time, bool) {
	args := mock.Mock.Called(file)

	t := args.Get(0)
	var timePtr *time.Time
	if t != nil {
		timePtr = t.(*time.Time)
	}
	return timePtr, args.Bool(1)
}

func (mock *MockHelperService) PipeCommands(commands ...*exec.Cmd) ([]byte, error) {
	args := mock.Mock.Called(commands)

	return args.Get(0).([]byte), args.Error(1)
}

func (mock *MockHelperService) DeleteEmptyStrings(s []string) []string {
	args := mock.Mock.Called(s)

	return args.Get(0).([]string)
}

func (mock *MockHelperService) GetConfigDir() string {
	args := mock.Mock.Called()

	return args.String(0)
}

func (mock *MockHelperService) GetConfigFilePath(name string) string {
	args := mock.Mock.Called(name)

	return args.String(0)
}

func (mock *MockHelperService) IsValidIp(ip string) bool {
	args := mock.Mock.Called(ip)

	return args.Get(0).(bool)
}

func (mock *MockHelperService) IsWsl() bool {
	args := mock.Mock.Called()

	return args.Get(0).(bool)
}
