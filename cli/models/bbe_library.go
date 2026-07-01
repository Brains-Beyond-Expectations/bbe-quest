package models

type Library struct {
	Library []LibraryEntry `mapstructure:"library"`
}

type LibraryEntry struct {
	MinBbeCli    string       `mapstructure:"min-bbe-cli" yaml:"min-bbe-cli"`
	ListRevision int          `mapstructure:"list-revision" yaml:"list-revision"`
	Charts       []ChartEntry `mapstructure:"charts" yaml:"charts"`
}

type ChartEntry struct {
	Name           string `mapstructure:"name" yaml:"name"`
	Version        string `mapstructure:"version" yaml:"version"`
	RepositoryUrl  string `mapstructure:"repositoryUrl" yaml:"repositoryUrl"`
	RepositoryName string `mapstructure:"repositoryName" yaml:"repositoryName"`
}

// RequiredField is a required leaf field from a chart's values.schema.json.
type RequiredField struct {
	Path        string
	Description string
	// Type is the JSON Schema type: "string", "boolean", "integer", or "number".
	Type string
	// Enum holds the allowed values when the field has an "enum" constraint.
	Enum []string
}

// RawSchemaNode is a typed representation of a JSON Schema object node used
// to walk a chart's values.schema.json and extract required leaf fields.
type RawSchemaNode struct {
	Type        string                   `json:"type"`
	Description string                   `json:"description"`
	Enum        []string                 `json:"enum"`
	Required    []string                 `json:"required"`
	Properties  map[string]RawSchemaNode `json:"properties"`
}
