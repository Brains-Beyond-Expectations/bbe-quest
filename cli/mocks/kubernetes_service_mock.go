package mocks

import "github.com/stretchr/testify/mock"

type MockKubernetesService struct {
	mock.Mock
}

func (m *MockKubernetesService) PortForward(context string, namespace string, service string, port int) (int, func(), error) {
	args := m.Called(context, namespace, service, port)
	stop, _ := args.Get(1).(func())
	return args.Int(0), stop, args.Error(2)
}

func (m *MockKubernetesService) GetSecretValue(context string, namespace string, secret string, key string) (string, error) {
	args := m.Called(context, namespace, secret, key)
	return args.String(0), args.Error(1)
}

func (m *MockKubernetesService) SetSecretValues(context string, namespace string, secret string, values map[string]string) error {
	args := m.Called(context, namespace, secret, values)
	return args.Error(0)
}

func (m *MockKubernetesService) RemoveSecretValues(context string, namespace string, secret string, keys []string) error {
	args := m.Called(context, namespace, secret, keys)
	return args.Error(0)
}
