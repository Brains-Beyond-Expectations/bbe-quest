package mocks

import (
	"os"

	"github.com/stretchr/testify/mock"
)

type MockOs struct {
	mock.Mock
}

func (mock *MockOs) ReadFile(filename string) ([]byte, error) {
	args := mock.Called(filename)
	return args.Get(0).([]byte), args.Error(1)
}

func (mock *MockOs) MkdirAll(path string, perm os.FileMode) error {
	args := mock.Called(path, perm)
	return args.Error(0)
}

func (mock *MockOs) WriteFile(filename string, data []byte, perm os.FileMode) error {
	args := mock.Called(filename, data, perm)
	return args.Error(0)
}

func (mock *MockOs) YamlMarshal(v interface{}) ([]byte, error) {
	args := mock.Called(v)
	return args.Get(0).([]byte), args.Error(1)
}
