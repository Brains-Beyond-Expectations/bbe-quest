package catalog_service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
)

func Test_IndexerDefinitions_Succeeds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/indexer/schema" || r.Header.Get("X-Api-Key") != "key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`[{"name": "LinuxTracker", "definitionName": "linuxtracker", "privacy": "public", "protocol": "torrent",
			"indexerUrls": ["https://linuxtracker.org/"],
			"fields": [{"name": "baseUrl", "label": "Base Url", "type": "select", "privacy": "normal", "advanced": false},
				{"name": "sort", "label": "Sort", "type": "select", "value": 0, "selectOptions": [{"value": 0, "name": "created"}]}]}]`))
	}))
	defer server.Close()

	definitions, err := CatalogService{}.IndexerDefinitions(server.URL, "key")

	assert.NoError(t, err)
	assert.Equal(t, []models.IndexerDefinition{{
		Name: "LinuxTracker", DefinitionName: "linuxtracker", Privacy: "public", Protocol: "torrent",
		IndexerUrls: []string{"https://linuxtracker.org/"},
		Fields: []models.IndexerField{
			{Name: "baseUrl", Label: "Base Url", Type: "select", Privacy: "normal"},
			{Name: "sort", Label: "Sort", Type: "select", Value: float64(0), SelectOptions: []models.IndexerFieldOption{{Value: float64(0), Name: "created"}}},
		},
	}}, definitions)
}

func Test_IndexerDefinitions_Fails(t *testing.T) {
	cases := map[string]http.HandlerFunc{
		"401 Unauthorized":  func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusUnauthorized) },
		"invalid character": func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("not json")) },
	}

	for expected, handler := range cases {
		server := httptest.NewServer(handler)

		definitions, err := CatalogService{}.IndexerDefinitions(server.URL, "key")

		assert.Nil(t, definitions, expected)
		assert.ErrorContains(t, err, expected, expected)
		server.Close()
	}
}

func Test_IndexerDefinitions_Fails_WhenProwlarrCannotBeReached(t *testing.T) {
	definitions, err := CatalogService{}.IndexerDefinitions("http://127.0.0.1:1", "key")

	assert.Nil(t, definitions)
	assert.ErrorContains(t, err, "Failed to read Prowlarr's indexers")
}

func Test_IndexerDefinitions_Fails_WithAnInvalidUrl(t *testing.T) {
	definitions, err := CatalogService{}.IndexerDefinitions("http://127.0.0.1:1\x7f", "key")

	assert.Nil(t, definitions)
	assert.ErrorContains(t, err, "invalid control character in URL")
}
