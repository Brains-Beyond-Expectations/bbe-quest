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
	// What bbe asks for to add an entry, for a setting without a catalog to look it up in
	Fields []SettingField `yaml:"fields"`
}

// One thing bbe asks for when adding an entry, stored under its name in the entry
type SettingField struct {
	Name        string `yaml:"name"`
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
	// A regular expression the value has to match
	Pattern  string `yaml:"pattern"`
	Optional bool   `yaml:"optional"`
	// No two entries have the same value, which names the entry. A setting with fields has one
	Unique bool `yaml:"unique"`
	// The values to choose from, instead of typing one
	Options []SettingFieldOption `yaml:"options"`
	// bbe generates the value, a random password, and shows it once it's saved
	Generate bool `yaml:"generate"`
	// Kept in the setting's secret instead of bbe.yaml
	Secret bool `yaml:"secret"`
}

type SettingFieldOption struct {
	Name  string      `yaml:"name"`
	Value interface{} `yaml:"value"`
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
