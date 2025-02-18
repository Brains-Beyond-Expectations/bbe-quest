package mocks

import (
	"github.com/stretchr/testify/mock"
)

type MockUiService struct {
	mock.Mock
}

func (mock *MockUiService) CreateSelect(title string, options []string) (string, error) {
	args := mock.Mock.Called(title, options)
	return args.Get(0).(string), args.Error(1)
}

func (mock *MockUiService) CreateInput(title string, suggestion string) (string, error) {
	args := mock.Mock.Called(title, suggestion)
	return args.Get(0).(string), args.Error(1)
}

func (mock *MockUiService) CreateMultiChoose(title string, options []string, defaultIndex []int) ([]string, error) {
	args := mock.Mock.Called(title, options, defaultIndex)
	return args.Get(0).([]string), args.Error(1)
}
