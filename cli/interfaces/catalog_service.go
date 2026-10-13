package interfaces

import "github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"

type CatalogServiceInterface interface {
	IndexerDefinitions(url string, apiKey string) ([]models.IndexerDefinition, error)
}
