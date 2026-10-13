package models

// A list in a chart's values that `bbe configure` adds entries to and removes them from, declared in its Chart.yaml
// under the `bbe/settings` annotation
type ChartSetting struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// The dotted path of the list in the chart's values, such as `bbe.media.prowlarr.indexers`
	Values string `yaml:"values"`
	// The secret in the package's namespace that holds the entries' credentials, instead of bbe.yaml
	Secret string `yaml:"secret"`
	// Where bbe looks up what can be added
	Catalog *SettingCatalog `yaml:"catalog"`
}

type SettingCatalog struct {
	// The kind of catalog, which bbe knows how to read, such as `prowlarr-indexers`
	Type string `yaml:"type"`
	// The service in the package's namespace serving the catalog, and its port
	Service string `yaml:"service"`
	Port    int    `yaml:"port"`
	// The key in a secret in the package's namespace to read the catalog with
	ApiKey *SecretKeyRef `yaml:"apiKey"`
}

type SecretKeyRef struct {
	Secret string `yaml:"secret"`
	Key    string `yaml:"key"`
}

// An indexer Prowlarr can add, from its catalog of definitions
type IndexerDefinition struct {
	Name           string         `json:"name"`
	DefinitionName string         `json:"definitionName"`
	Description    string         `json:"description"`
	Privacy        string         `json:"privacy"`
	Protocol       string         `json:"protocol"`
	IndexerUrls    []string       `json:"indexerUrls"`
	Fields         []IndexerField `json:"fields"`
}

type IndexerField struct {
	Name          string               `json:"name"`
	Label         string               `json:"label"`
	HelpText      string               `json:"helpText"`
	Type          string               `json:"type"`
	Privacy       string               `json:"privacy"`
	Advanced      bool                 `json:"advanced"`
	Value         interface{}          `json:"value"`
	SelectOptions []IndexerFieldOption `json:"selectOptions"`
}

type IndexerFieldOption struct {
	Value interface{} `json:"value"`
	Name  string      `json:"name"`
}
