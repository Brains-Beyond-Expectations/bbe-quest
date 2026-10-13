package mocks

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/mock"
)

type MockCatalogService struct {
	mock.Mock
}

func (m *MockCatalogService) IndexerDefinitions(url string, apiKey string) ([]models.IndexerDefinition, error) {
	args := m.Called(url, apiKey)
	definitions, _ := args.Get(0).([]models.IndexerDefinition)
	return definitions, args.Error(1)
}
