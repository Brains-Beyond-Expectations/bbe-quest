package mocks

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/models"
	"github.com/stretchr/testify/mock"
)

type MockImageService struct {
	mock.Mock
}

func (m *MockImageService) CreateImage(nodeType models.NodeType, outputDir string) (string, error) {
	args := m.Called(nodeType, outputDir)

	return args.String(0), args.Error(1)
}
