package models

// The parts of a packaged helm chart bbe reads to find the values it requires
type HelmChart struct {
	Name         string
	Values       map[string]interface{}
	Schema       *HelmChartSchema
	Dependencies []HelmChartDependency
}

type HelmChartMetadata struct {
	Name         string                `yaml:"name"`
	Dependencies []HelmChartDependency `yaml:"dependencies"`
}

type HelmChartDependency struct {
	Name      string     `yaml:"name"`
	Alias     string     `yaml:"alias"`
	Condition string     `yaml:"condition"`
	Chart     *HelmChart `yaml:"-"`
}

// The subset of JSON schema bbe understands, see https://json-schema.org/understanding-json-schema/reference
type HelmChartSchema struct {
	Type        interface{}                `json:"type"` // A type name, or a list of them
	Description string                     `json:"description"`
	Required    []string                   `json:"required"`
	Properties  map[string]HelmChartSchema `json:"properties"`
	Enum        []interface{}              `json:"enum"`
	MinLength   *int                       `json:"minLength"`
	Pattern     string                     `json:"pattern"`
	Default     interface{}                `json:"default"`
	Examples    []interface{}              `json:"examples"`
}

// A value required by a chart's schema that isn't set yet
type RequiredChartValue struct {
	Path   []string
	Schema HelmChartSchema
}
