package interfaces

import "github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"

type PrerequisiteServiceInterface interface {
	Check(prerequisite models.ChartPrerequisite, cluster models.ClusterAccess) ([]string, error)
	PrepareNodes(prerequisite models.TalosPrerequisite, cluster models.ClusterAccess) error
}
