package catalog_service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

type CatalogService struct{}

// The indexers Prowlarr can add, read from its API at url
func (catalogService CatalogService) IndexerDefinitions(url string, apiKey string) ([]models.IndexerDefinition, error) {
	request, err := http.NewRequest(http.MethodGet, url+"/api/v1/indexer/schema", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("X-Api-Key", apiKey)
	request.Header.Set("Accept", "application/json")

	response, err := httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("Failed to read Prowlarr's indexers: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 500))
		return nil, fmt.Errorf("Failed to read Prowlarr's indexers: %s %s", response.Status, body)
	}

	var definitions []models.IndexerDefinition
	if err := json.NewDecoder(response.Body).Decode(&definitions); err != nil {
		return nil, fmt.Errorf("Failed to read Prowlarr's indexers: %w", err)
	}

	return definitions, nil
}
