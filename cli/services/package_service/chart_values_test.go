package package_service

import (
	"errors"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v2"
)

var networkingChart = models.ChartEntry{Name: "bbe-networking", Version: "0.3.0", RepositoryUrl: "https://example.com/charts"}

func Test_findMissingValues_Succeeds_FindingNestedRequiredValues(t *testing.T) {
	chart := chartRequiring("bbe", "metallb", "ipAddressPool")
	chart.Values = map[string]interface{}{"bbe": map[string]interface{}{"metallb": map[string]interface{}{"ipAddressPool": nil}}}

	missing := findMissingValues(chart, nil)

	assert.Equal(t, [][]string{{"bbe", "metallb", "ipAddressPool"}}, missingPaths(missing))
	assert.Equal(t, "A required value", missing[0].Schema.Description)
}

func Test_findMissingValues_Succeeds_WhenDefaultsOrValuesProvideThem(t *testing.T) {
	chart := chartRequiring("service", "ip")
	chart.Values = map[string]interface{}{"service": map[string]interface{}{"ip": "192.168.1.240"}}
	assert.Empty(t, findMissingValues(chart, nil))

	chart.Values = map[string]interface{}{"service": map[string]interface{}{"ip": ""}}
	assert.Empty(t, findMissingValues(chart, map[string]interface{}{"service": map[string]interface{}{"ip": "192.168.1.240"}}))
}

func Test_findMissingValues_Succeeds_CheckingDependencies(t *testing.T) {
	dependency := chartRequiring("password")
	chart := &models.HelmChart{Dependencies: []models.HelmChartDependency{{Name: "database", Alias: "db", Chart: dependency}}}

	assert.Equal(t, [][]string{{"db", "password"}}, missingPaths(findMissingValues(chart, nil)))

	// The parent chart's defaults and the given values both override the dependency's defaults
	chart.Values = map[string]interface{}{"db": map[string]interface{}{"password": "from-parent"}}
	assert.Empty(t, findMissingValues(chart, nil))

	chart.Values = nil
	dependency.Values = map[string]interface{}{"password": ""}
	assert.Empty(t, findMissingValues(chart, map[string]interface{}{"db": map[string]interface{}{"password": "given"}}))
}

func Test_findMissingValues_Succeeds_SkippingDisabledDependencies(t *testing.T) {
	chart := &models.HelmChart{
		Values:       map[string]interface{}{"database": map[string]interface{}{"enabled": false}},
		Dependencies: []models.HelmChartDependency{{Name: "database", Condition: "database.enabled", Chart: chartRequiring("password")}},
	}
	assert.Empty(t, findMissingValues(chart, nil))

	enabled := map[string]interface{}{"database": map[string]interface{}{"enabled": true}}
	assert.Equal(t, [][]string{{"database", "password"}}, missingPaths(findMissingValues(chart, enabled)))
}

func Test_findMissingValues_Succeeds_ReportingAValueRequiredTwiceOnce(t *testing.T) {
	chart := chartRequiring("database", "password")
	chart.Dependencies = []models.HelmChartDependency{{Name: "database", Chart: chartRequiring("password")}}

	assert.Equal(t, [][]string{{"database", "password"}}, missingPaths(findMissingValues(chart, nil)))
}

func Test_findMissingValues_Succeeds_CheckingOptionalObjectsThatAreSet(t *testing.T) {
	persistence := chartRequiring("persistence", "size").Schema.Properties["persistence"]
	chart := &models.HelmChart{Schema: &models.HelmChartSchema{Properties: map[string]models.HelmChartSchema{"persistence": persistence}}}

	assert.Empty(t, findMissingValues(chart, nil))
	assert.Equal(t, [][]string{{"persistence", "size"}}, missingPaths(findMissingValues(chart, map[string]interface{}{"persistence": map[string]interface{}{}})))
}

func Test_findMissingValues_Succeeds_ReportingValuesThatBreakTheSchema(t *testing.T) {
	minLength := 3
	chart := &models.HelmChart{
		Values: map[string]interface{}{"replicas": "two", "mode": "fast", "name": "ab", "tag": "latest", "nullable": nil, "anything": 1},
		Schema: &models.HelmChartSchema{
			Required: []string{"replicas", "mode", "name", "tag", "nullable", "anything", "absent"},
			Properties: map[string]models.HelmChartSchema{
				"replicas": {Type: "integer"},
				"mode":     {Enum: []interface{}{"slow", "safe"}},
				"name":     {Type: "string", MinLength: &minLength},
				"tag":      {Type: "string", Pattern: `^v\d+$`},
				"nullable": {Type: []interface{}{"string", "null"}},
				"anything": {},
			},
		},
	}

	assert.Equal(t, [][]string{{"replicas"}, {"mode"}, {"name"}, {"tag"}, {"absent"}}, missingPaths(findMissingValues(chart, nil)))
}

func Test_resolveValues_Succeeds_AskingForEachMissingValue(t *testing.T) {
	chart := chartRequiring("service", "ip")
	chart.Schema.Required = append(chart.Schema.Required, "replicas", "mode", "debug")
	chart.Schema.Properties["replicas"] = models.HelmChartSchema{Type: "integer", Default: 2.0}
	chart.Schema.Properties["mode"] = models.HelmChartSchema{Enum: []interface{}{"slow", "safe"}}
	chart.Schema.Properties["debug"] = models.HelmChartSchema{Type: "boolean"}
	mockUiService := &mocks.MockUiService{}
	mockUiService.On("CreateInput", "bbe-networking: A required value (`service.ip`)", "").Return("192.168.1.240", nil)
	mockUiService.On("CreateInput", "bbe-networking requires `replicas`", "2").Return("3", nil)
	mockUiService.On("CreateSelect", "bbe-networking requires `mode`", []string{"slow", "safe"}).Return("safe", nil)
	mockUiService.On("CreateSelect", "bbe-networking requires `debug`", []string{"true", "false"}).Return("false", nil)

	values, err := resolveValues(networkingChart, chart, nil, mockUiService, true)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"service":  map[string]interface{}{"ip": "192.168.1.240"},
		"replicas": 3,
		"mode":     "safe",
		"debug":    false,
	}, values)
}

func Test_resolveValues_Succeeds_AskingAgainForAnInvalidValue(t *testing.T) {
	mockUiService := &mocks.MockUiService{}
	mockUiService.On("CreateInput", mock.Anything, mock.Anything).Return("  ", nil).Once()
	mockUiService.On("CreateInput", mock.Anything, mock.Anything).Return(" 192.168.1.240 ", nil).Once()

	values, err := resolveValues(networkingChart, chartRequiring("ip"), nil, mockUiService, true)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"ip": "192.168.1.240"}, values)
	mockUiService.AssertNumberOfCalls(t, "CreateInput", 2)
}

func Test_resolveValues_Succeeds_ReusingValuesStoredInBbeConfig(t *testing.T) {
	mockUiService := &mocks.MockUiService{}

	// Round trip the values through yaml.v2 like bbe.yaml
	var pkg models.LocalPackage
	content, _ := yaml.Marshal(models.LocalPackage{Name: "bbe-networking", Values: map[string]interface{}{"service": map[string]interface{}{"ip": "192.168.1.240"}}})
	yaml.Unmarshal(content, &pkg)

	values, err := resolveValues(networkingChart, chartRequiring("service", "ip"), pkg.Values, mockUiService, false)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"service": map[string]interface{}{"ip": "192.168.1.240"}}, values)
	mockUiService.AssertNotCalled(t, "CreateInput", mock.Anything, mock.Anything)
}

func Test_resolveValues_Fails_WhenAValueCannotBeEntered(t *testing.T) {
	chart := &models.HelmChart{Schema: &models.HelmChartSchema{
		Required:   []string{"hosts"},
		Properties: map[string]models.HelmChartSchema{"hosts": {Type: "array"}},
	}}

	values, err := resolveValues(networkingChart, chart, nil, &mocks.MockUiService{}, true)

	assert.Nil(t, values)
	assert.EqualError(t, err, "Package `bbe-networking` requires values that can't be entered here: `hosts`. Add them under `values` for the package in bbe.yaml")
}

func Test_resolveValues_Fails_WhenThePromptFails(t *testing.T) {
	mockUiService := &mocks.MockUiService{}
	mockUiService.On("CreateInput", mock.Anything, mock.Anything).Return("", errors.New("Mock prompt cancelled"))

	values, err := resolveValues(networkingChart, chartRequiring("ip"), nil, mockUiService, true)

	assert.Nil(t, values)
	assert.EqualError(t, err, "Mock prompt cancelled")
}

func Test_parseValue_Succeeds_ConvertingToTheSchemaType(t *testing.T) {
	cases := []struct {
		schema   models.HelmChartSchema
		input    string
		expected interface{}
	}{
		{models.HelmChartSchema{}, " text ", "text"},
		{models.HelmChartSchema{Type: []interface{}{"integer", "string"}}, "3", "3"},
		{models.HelmChartSchema{Type: "integer"}, "3", 3},
		{models.HelmChartSchema{Type: "number"}, "1.5", 1.5},
		{models.HelmChartSchema{Type: "boolean"}, "true", true},
		{models.HelmChartSchema{Type: []interface{}{"boolean", "integer"}}, "4", 4},
	}

	for _, testCase := range cases {
		value, err := parseValue(testCase.schema, testCase.input)
		assert.NoError(t, err)
		assert.Equal(t, testCase.expected, value)
	}
}

func Test_parseValue_Fails_WhenTheInputBreaksTheSchema(t *testing.T) {
	minLength := 1
	longer := 3
	cases := []struct {
		schema   models.HelmChartSchema
		input    string
		expected string
	}{
		{models.HelmChartSchema{Type: "integer"}, "three", "must be a whole number"},
		{models.HelmChartSchema{Type: []interface{}{"number", "boolean"}}, "yes please", "must be a number or true or false"},
		{models.HelmChartSchema{Type: "string", MinLength: &minLength}, "", "can't be empty"},
		{models.HelmChartSchema{Type: "string", MinLength: &longer}, "ab", "must be at least 3 characters"},
		{models.HelmChartSchema{Type: "string", Pattern: `^\d+\.\d+\.\d+\.\d+$`}, "localhost", "must match the pattern `^\\d+\\.\\d+\\.\\d+\\.\\d+$`"},
		{models.HelmChartSchema{Enum: []interface{}{"slow", 2.0}}, "fast", "must be one of: slow, 2"},
		{models.HelmChartSchema{Type: "custom"}, "anything", "must be custom"},
	}

	for _, testCase := range cases {
		value, err := parseValue(testCase.schema, testCase.input)
		assert.Nil(t, value)
		assert.EqualError(t, err, testCase.expected)
	}
}

func Test_validateValue_Succeeds_WithSchemasBbeCannotFullyCheck(t *testing.T) {
	assert.NoError(t, validateValue(models.HelmChartSchema{Type: "string", Pattern: "(?<=x)y"}, "text"))
	assert.NoError(t, validateValue(models.HelmChartSchema{Type: "custom"}, "text"))
	assert.NoError(t, validateValue(models.HelmChartSchema{Enum: []interface{}{1.0}}, 1))
	assert.EqualError(t, validateValue(models.HelmChartSchema{Type: "custom-type", Enum: []interface{}{"a"}}, "b"), "must be one of: a")
	assert.EqualError(t, validateValue(models.HelmChartSchema{Type: []interface{}{"object", "array", "null", 5}}, "text"), "must be an object or a list or empty")
}

func Test_hasType_Succeeds_MatchingDecodedValues(t *testing.T) {
	assert.True(t, hasType(int64(1), "integer"))
	assert.True(t, hasType(uint64(1), "number"))
	assert.True(t, hasType(2.0, "integer"))
	assert.False(t, hasType(2.5, "integer"))
	assert.False(t, hasType("2", "integer"))
	assert.False(t, hasType("2", "number"))
	assert.True(t, hasType(map[string]interface{}{}, "object"))
	assert.True(t, hasType([]interface{}{}, "array"))
	assert.True(t, hasType(nil, "null"))
	assert.True(t, hasType("anything", "unknown"))
}

func Test_suggestValue_Succeeds_UsingAValidDefaultOrExample(t *testing.T) {
	assert.Equal(t, "", suggestValue(models.HelmChartSchema{}))
	assert.Equal(t, "8080", suggestValue(models.HelmChartSchema{Type: "integer", Default: 8080.0}))
	assert.Equal(t, "192.168.1.240", suggestValue(models.HelmChartSchema{Type: "string", Default: 5.0, Examples: []interface{}{"192.168.1.240"}}))
}

func Test_isDependencyEnabled_Succeeds_UsingTheFirstBooleanCondition(t *testing.T) {
	values := map[string]interface{}{"first": map[string]interface{}{"enabled": "yes"}, "second": map[string]interface{}{"enabled": false}}

	assert.True(t, isDependencyEnabled("", values))
	assert.True(t, isDependencyEnabled("missing.enabled", values))
	assert.False(t, isDependencyEnabled("first.enabled, ,second.enabled", values))
	assert.True(t, isDependencyEnabled("first.enabled.deeper", values))
}

func Test_setValue_Succeeds_ReplacingValuesThatAreNotMaps(t *testing.T) {
	values := map[string]interface{}{"service": "not a map"}

	setValue(values, []string{"service", "ip"}, "192.168.1.240")

	assert.Equal(t, map[string]interface{}{"service": map[string]interface{}{"ip": "192.168.1.240"}}, values)
}

func Test_normalizeValues_Succeeds_ConvertingNestedMaps(t *testing.T) {
	values := map[string]interface{}{
		"hosts": []interface{}{map[interface{}]interface{}{"name": "a", 1: "b"}},
	}

	assert.Equal(t, map[string]interface{}{
		"hosts": []interface{}{map[string]interface{}{"name": "a", "1": "b"}},
	}, normalizeValues(values))
}

// A chart whose schema requires a non-empty string at `path`
func chartRequiring(path ...string) *models.HelmChart {
	minLength := 1
	schema := models.HelmChartSchema{Type: "string", MinLength: &minLength, Description: "A required value"}
	for i := len(path) - 1; i >= 0; i-- {
		schema = models.HelmChartSchema{
			Type:       "object",
			Required:   []string{path[i]},
			Properties: map[string]models.HelmChartSchema{path[i]: schema},
		}
	}

	return &models.HelmChart{Name: "chart", Schema: &schema}
}

func missingPaths(missing []models.RequiredChartValue) [][]string {
	var paths [][]string
	for _, required := range missing {
		paths = append(paths, required.Path)
	}

	return paths
}
