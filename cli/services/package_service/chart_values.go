package package_service

import (
	"errors"
	"fmt"
	"maps"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
)

// Returns `values` completed with anything the chart's schemas require that isn't set yet, asking the user for it
func resolveValues(chart models.ChartEntry, helmChart *models.HelmChart, values map[string]interface{}, uiService interfaces.UiServiceInterface, interactive bool) (map[string]interface{}, error) {
	resolved := normalizeValues(values)

	missing := findMissingValues(helmChart, resolved)
	if len(missing) == 0 {
		return resolved, nil
	}

	if !interactive {
		return nil, fmt.Errorf("Package `%s` requires values that aren't set: %s. Run without --yes to enter them, or add them under `values` for the package in %s", chart.Name, describePaths(missing), constants.BbeConfigFile)
	}

	if unsupported := slices.DeleteFunc(slices.Clone(missing), canPrompt); len(unsupported) > 0 {
		return nil, fmt.Errorf("Package `%s` requires values that can't be entered here: %s. Add them under `values` for the package in %s", chart.Name, describePaths(unsupported), constants.BbeConfigFile)
	}

	logger.Info(fmt.Sprintf("Package `%s` requires %d value(s) that aren't set yet", chart.Name, len(missing)))
	if resolved == nil {
		resolved = map[string]interface{}{}
	}

	for _, required := range missing {
		value, err := promptForValue(uiService, chart.Name, required)
		if err != nil {
			return nil, err
		}
		setValue(resolved, required.Path, value)
	}

	return resolved, nil
}

// Checks the chart's schema and those of the charts it depends on, like helm does before installing
func findMissingValues(chart *models.HelmChart, values map[string]interface{}) []models.RequiredChartValue {
	var missing []models.RequiredChartValue
	seen := map[string]bool{}

	// A chart and one of its dependencies can both require the same value
	for _, required := range missingChartValues(chart, coalesceValues(chart, values), nil) {
		key := strings.Join(required.Path, ".")
		if !seen[key] {
			seen[key] = true
			missing = append(missing, required)
		}
	}

	return missing
}

func missingChartValues(chart *models.HelmChart, values map[string]interface{}, path []string) []models.RequiredChartValue {
	var missing []models.RequiredChartValue
	if chart.Schema != nil {
		missing = missingSchemaValues(*chart.Schema, values, path)
	}

	for _, dependency := range enabledDependencies(chart, values) {
		key := dependencyKey(dependency)
		dependencyValues, _ := values[key].(map[string]interface{})
		missing = append(missing, missingChartValues(dependency.Chart, dependencyValues, appendPath(path, key))...)
	}

	return missing
}

func missingSchemaValues(schema models.HelmChartSchema, values map[string]interface{}, path []string) []models.RequiredChartValue {
	var missing []models.RequiredChartValue
	for _, key := range schemaKeys(schema) {
		property := schema.Properties[key]
		value, found := values[key]
		nested, isMap := value.(map[string]interface{})
		required := slices.Contains(schema.Required, key)

		switch {
		// A required object that isn't set is missing each of the values it requires
		case isMap || (required && len(property.Required) > 0):
			missing = append(missing, missingSchemaValues(property, nested, appendPath(path, key))...)
		case required && (!found || validateValue(property, value) != nil):
			missing = append(missing, models.RequiredChartValue{Path: appendPath(path, key), Schema: property})
		}
	}

	return missing
}

// Required keys in the schema's order, then the other properties, which can still require values when they're set
func schemaKeys(schema models.HelmChartSchema) []string {
	keys := slices.Clone(schema.Required)
	for _, key := range slices.Sorted(maps.Keys(schema.Properties)) {
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}

	return keys
}

// Helm gives a chart's defaults lower priority than the values it's installed with, and the same between a chart and its dependencies
func coalesceValues(chart *models.HelmChart, values map[string]interface{}) map[string]interface{} {
	coalesced := mergeValues(normalizeValues(chart.Values), values)
	for _, dependency := range enabledDependencies(chart, coalesced) {
		key := dependencyKey(dependency)
		dependencyValues, _ := coalesced[key].(map[string]interface{})
		coalesced[key] = coalesceValues(dependency.Chart, dependencyValues)
	}

	return coalesced
}

func mergeValues(base map[string]interface{}, override map[string]interface{}) map[string]interface{} {
	merged := maps.Clone(base)
	if merged == nil {
		merged = map[string]interface{}{}
	}

	for key, value := range override {
		baseMap, baseIsMap := merged[key].(map[string]interface{})
		overrideMap, overrideIsMap := value.(map[string]interface{})
		if baseIsMap && overrideIsMap {
			merged[key] = mergeValues(baseMap, overrideMap)
		} else {
			merged[key] = value
		}
	}

	return merged
}

func enabledDependencies(chart *models.HelmChart, values map[string]interface{}) []models.HelmChartDependency {
	var enabled []models.HelmChartDependency
	for _, dependency := range chart.Dependencies {
		if dependency.Chart != nil && isDependencyEnabled(dependency.Condition, values) {
			enabled = append(enabled, dependency)
		}
	}

	return enabled
}

// A condition lists comma separated value paths, and the first one set to a boolean decides
func isDependencyEnabled(condition string, values map[string]interface{}) bool {
	for _, conditionPath := range strings.Split(condition, ",") {
		conditionPath = strings.TrimSpace(conditionPath)
		if conditionPath == "" {
			continue
		}

		if enabled, isBool := lookupValue(values, strings.Split(conditionPath, ".")).(bool); isBool {
			return enabled
		}
	}

	return true
}

func dependencyKey(dependency models.HelmChartDependency) string {
	if dependency.Alias != "" {
		return dependency.Alias
	}

	return dependency.Name
}

func validateValue(schema models.HelmChartSchema, value interface{}) error {
	types := schemaTypes(schema)
	if len(types) > 0 && !slices.ContainsFunc(types, func(typeName string) bool { return hasType(value, typeName) }) {
		return fmt.Errorf("must be %s", describeTypes(types))
	}

	if len(schema.Enum) > 0 && !slices.ContainsFunc(schema.Enum, func(option interface{}) bool { return fmt.Sprint(option) == fmt.Sprint(value) }) {
		return fmt.Errorf("must be one of: %s", strings.Join(enumOptions(schema), ", "))
	}

	text, isText := value.(string)
	if !isText {
		return nil
	}

	if schema.MinLength != nil && utf8.RuneCountInString(text) < *schema.MinLength {
		if *schema.MinLength == 1 {
			return errors.New("can't be empty")
		}
		return fmt.Errorf("must be at least %d characters", *schema.MinLength)
	}

	if schema.Pattern != "" {
		// Helm checks the pattern as well, so one bbe can't compile is left to helm
		if pattern, err := regexp.Compile(schema.Pattern); err == nil && !pattern.MatchString(text) {
			return fmt.Errorf("must match the pattern `%s`", schema.Pattern)
		}
	}

	return nil
}

func schemaTypes(schema models.HelmChartSchema) []string {
	switch typed := schema.Type.(type) {
	case string:
		return []string{typed}
	case []interface{}:
		var types []string
		for _, typeName := range typed {
			if name, isText := typeName.(string); isText {
				types = append(types, name)
			}
		}
		return types
	}

	return nil
}

func hasType(value interface{}, typeName string) bool {
	switch typeName {
	case "string":
		_, matches := value.(string)
		return matches
	case "boolean":
		_, matches := value.(bool)
		return matches
	case "integer":
		switch number := value.(type) {
		case int, int64, uint64:
			return true
		case float64:
			return number == math.Trunc(number)
		}
		return false
	case "number":
		switch value.(type) {
		case int, int64, uint64, float64:
			return true
		}
		return false
	case "object":
		_, matches := value.(map[string]interface{})
		return matches
	case "array":
		_, matches := value.([]interface{})
		return matches
	case "null":
		return value == nil
	}

	return true
}

func describeTypes(types []string) string {
	descriptions := map[string]string{
		"string":  "a string",
		"integer": "a whole number",
		"number":  "a number",
		"boolean": "true or false",
		"object":  "an object",
		"array":   "a list",
		"null":    "empty",
	}

	var described []string
	for _, typeName := range types {
		if description, found := descriptions[typeName]; found {
			described = append(described, description)
		} else {
			described = append(described, typeName)
		}
	}

	return strings.Join(described, " or ")
}

// Only scalar values can be typed into a prompt
func canPrompt(required models.RequiredChartValue) bool {
	types := schemaTypes(required.Schema)
	if len(types) == 0 || len(required.Schema.Enum) > 0 {
		return true
	}

	return slices.ContainsFunc(types, func(typeName string) bool {
		return slices.Contains([]string{"string", "integer", "number", "boolean"}, typeName)
	})
}

func promptForValue(uiService interfaces.UiServiceInterface, chartName string, required models.RequiredChartValue) (interface{}, error) {
	path := strings.Join(required.Path, ".")
	title := fmt.Sprintf("%s requires `%s`", chartName, path)
	if required.Schema.Description != "" {
		title = fmt.Sprintf("%s: %s (`%s`)", chartName, required.Schema.Description, path)
	}

	options := enumOptions(required.Schema)
	if len(options) == 0 && slices.Equal(schemaTypes(required.Schema), []string{"boolean"}) {
		options = []string{"true", "false"}
	}

	for {
		var input string
		var err error
		if len(options) > 0 {
			input, err = uiService.CreateSelect(title, options)
		} else {
			input, err = uiService.CreateInput(title, suggestValue(required.Schema))
		}
		if err != nil {
			return nil, err
		}

		value, err := parseValue(required.Schema, input)
		if err == nil {
			return value, nil
		}
		logger.Warning(fmt.Sprintf("`%s` %v, please try again", path, err))
	}
}

func enumOptions(schema models.HelmChartSchema) []string {
	var options []string
	for _, option := range schema.Enum {
		options = append(options, fmt.Sprint(option))
	}

	return options
}

// Helm doesn't apply a schema's default or examples, but they make good suggestions
func suggestValue(schema models.HelmChartSchema) string {
	for _, suggestion := range append([]interface{}{schema.Default}, schema.Examples...) {
		if suggestion != nil && validateValue(schema, suggestion) == nil {
			return fmt.Sprint(suggestion)
		}
	}

	return ""
}

func parseValue(schema models.HelmChartSchema, input string) (interface{}, error) {
	value, err := convertInput(schemaTypes(schema), strings.TrimSpace(input))
	if err != nil {
		return nil, err
	}

	if err := validateValue(schema, value); err != nil {
		return nil, err
	}

	return value, nil
}

func convertInput(types []string, input string) (interface{}, error) {
	if len(types) == 0 || slices.Contains(types, "string") {
		return input, nil
	}

	for _, typeName := range types {
		switch typeName {
		case "integer":
			if number, err := strconv.Atoi(input); err == nil {
				return number, nil
			}
		case "number":
			if number, err := strconv.ParseFloat(input, 64); err == nil {
				return number, nil
			}
		case "boolean":
			if boolean, err := strconv.ParseBool(input); err == nil {
				return boolean, nil
			}
		}
	}

	return nil, fmt.Errorf("must be %s", describeTypes(types))
}

// bbe.yaml is read with yaml.v2, which decodes nested maps with interface{} keys
func normalizeValues(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}

	normalized := make(map[string]interface{}, len(values))
	for key, value := range values {
		normalized[key] = normalizeValue(value)
	}

	return normalized
}

func normalizeValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		return normalizeValues(typed)
	case map[interface{}]interface{}:
		normalized := make(map[string]interface{}, len(typed))
		for key, nested := range typed {
			normalized[fmt.Sprint(key)] = normalizeValue(nested)
		}
		return normalized
	case []interface{}:
		normalized := make([]interface{}, len(typed))
		for i, nested := range typed {
			normalized[i] = normalizeValue(nested)
		}
		return normalized
	}

	return value
}

func lookupValue(values map[string]interface{}, path []string) interface{} {
	var current interface{} = values
	for _, key := range path {
		nested, isMap := current.(map[string]interface{})
		if !isMap {
			return nil
		}
		current = nested[key]
	}

	return current
}

func setValue(values map[string]interface{}, path []string, value interface{}) {
	for _, key := range path[:len(path)-1] {
		nested, isMap := values[key].(map[string]interface{})
		if !isMap {
			nested = map[string]interface{}{}
			values[key] = nested
		}
		values = nested
	}
	values[path[len(path)-1]] = value
}

func appendPath(path []string, key string) []string {
	return append(slices.Clone(path), key)
}

func describePaths(values []models.RequiredChartValue) string {
	var paths []string
	for _, value := range values {
		paths = append(paths, fmt.Sprintf("`%s`", strings.Join(value.Path, ".")))
	}

	return strings.Join(paths, ", ")
}
