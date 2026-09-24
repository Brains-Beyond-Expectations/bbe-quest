package mocks

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/mock"
)

type MockPrerequisiteService struct {
	mock.Mock
}

func (m *MockPrerequisiteService) Check(prerequisite models.ChartPrerequisite, cluster models.ClusterAccess) ([]string, error) {
	args := m.Called(prerequisite, cluster)
	missing, _ := args.Get(0).([]string)
	return missing, args.Error(1)
}

func (m *MockPrerequisiteService) PrepareNodes(prerequisite models.TalosPrerequisite, cluster models.ClusterAccess) error {
	args := m.Called(prerequisite, cluster)
	return args.Error(0)
}
