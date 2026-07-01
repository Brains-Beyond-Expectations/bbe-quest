package package_service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/kaptinlin/jsonschema"
	"gopkg.in/yaml.v2"
)

type PackageService struct{}

var ioReadAll = io.ReadAll
var osMkdirAll = os.MkdirAll
var osWriteFile = os.WriteFile
var yamlMarshal = yaml.Marshal

func getRemoteLibrary() (*models.LibraryEntry, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Fetch the library.yaml file from remote
	resp, err := client.Get(constants.BbeLibraryUrl)
	if err != nil {
		logger.Debug(fmt.Sprintf("Error fetching library.yaml: %v", err))
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioReadAll(resp.Body)
	if err != nil {
		logger.Debug(fmt.Sprintf("Error reading response body: %v", err))
		return nil, err
	}

	var library models.Library
	if err := yaml.Unmarshal(body, &library); err != nil {
		logger.Debug(fmt.Sprintf("Error parsing YAML: %v", err))
		return nil, err
	}

	for _, revision := range library.Library {
		if revision.MinBbeCli <= constants.Version {
			return &revision, nil
		}
	}

	return nil, fmt.Errorf("No revision found for current bbe-cli version")
}

func (packageService PackageService) GetAll() ([]models.ChartEntry, error) {
	library, err := getRemoteLibrary()
	if err != nil {
		logger.Debug(fmt.Sprintf("Error fetching library: %v", err))
		return nil, err
	}

	return library.Charts, nil
}

func (packageService PackageService) InstallPackage(chart models.ChartEntry, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface, uiService interfaces.UiServiceInterface, helperService interfaces.HelperServiceInterface) error {
	if !helmService.IsPackageInstalled(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context) {
		logger.Debug(fmt.Sprintf("Package `%s` not installed, adding helm repo...", chart.Name))
		response := helmService.AddRepo(chart.RepositoryName, chart.RepositoryUrl)
		logger.Debug(fmt.Sprintf("Helm repo added: %v", response))

		if response != nil {
			return response
		}

		valuesFile, err := ensureValuesFile(chart.Name, helperService, uiService)
		if err != nil {
			return fmt.Errorf("Failed to gather required values for `%s`: %w", chart.Name, err)
		}

		return helmService.InstallChart(chart.Name, chart.Name, chart.RepositoryName, chart.Version, chart.Name, bbeConfig.Bbe.Cluster.Context, valuesFile)
	}
	logger.Debug(fmt.Sprintf("Package `%s` already installed", chart.Name))

	return nil
}

func (packageService PackageService) UpgradePackage(chart models.ChartEntry, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface, uiService interfaces.UiServiceInterface, helperService interfaces.HelperServiceInterface) error {
	if !helmService.IsPackageInstalled(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context) {
		logger.Debug(fmt.Sprintf("Package `%s` not installed", chart.Name))
		return fmt.Errorf("Package `%s` not installed", chart.Name)
	}

	response := helmService.AddRepo(chart.RepositoryName, chart.RepositoryUrl)

	if response != nil {
		return response
	}

	valuesFile, err := ensureValuesFile(chart.Name, helperService, uiService)
	if err != nil {
		return fmt.Errorf("Failed to gather required values for `%s`: %w", chart.Name, err)
	}

	return helmService.UpgradeChart(chart.Name, chart.Name, chart.RepositoryName, chart.Version, chart.Name, bbeConfig.Bbe.Cluster.Context, valuesFile)
}

func (packageService PackageService) UninstallPackage(chart models.LocalPackage, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	if !helmService.IsPackageInstalled(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context) {
		logger.Debug(fmt.Sprintf("Package `%s` not installed", chart.Name))
		return fmt.Errorf("Package `%s` not installed", chart.Name)
	}

	return helmService.UninstallChart(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context)
}

// fetchRemoteValuesSchema fetches, compiles, and validates the values.schema.json for a chart.
// Returns the typed schema node (used for field extraction) and the compiled schema (used for final validation).
func fetchRemoteValuesSchema(packageName string) (models.RawSchemaNode, *jsonschema.Schema, error) {
	url := fmt.Sprintf(constants.BbeChartSchemaUrl, packageName)
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(url)
	if err != nil {
		return models.RawSchemaNode{}, nil, fmt.Errorf("failed to fetch schema for `%s`: %w", packageName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return models.RawSchemaNode{}, nil, fmt.Errorf("no schema found for `%s` (HTTP %d)", packageName, resp.StatusCode)
	}

	body, err := ioReadAll(resp.Body)
	if err != nil {
		return models.RawSchemaNode{}, nil, err
	}

	compiler := jsonschema.NewCompiler()
	compiled, err := compiler.Compile(body)
	if err != nil {
		return models.RawSchemaNode{}, nil, fmt.Errorf("invalid JSON Schema for `%s`: %w", packageName, err)
	}

	var rawSchema models.RawSchemaNode
	if err := json.Unmarshal(body, &rawSchema); err != nil {
		return models.RawSchemaNode{}, nil, fmt.Errorf("failed to parse schema for `%s`: %w", packageName, err)
	}

	return rawSchema, compiled, nil
}

// extractRequiredFields recursively walks a JSON schema object and returns all required
// leaf fields as dot-notation paths (e.g. "bbe.metallb.blocky.ipAddressPool").
// Leaf fields are those with type "string", "boolean", "integer", "number", or an "enum".
func extractRequiredFields(schema models.RawSchemaNode, prefix string) []models.RequiredField {
	var fields []models.RequiredField

	hasEnum := len(schema.Enum) > 0
	isLeaf := hasEnum || schema.Type == "string" || schema.Type == "boolean" ||
		schema.Type == "integer" || schema.Type == "number"

	if isLeaf {
		return []models.RequiredField{{Path: prefix, Description: schema.Description, Type: schema.Type, Enum: schema.Enum}}
	}

	if len(schema.Required) == 0 || len(schema.Properties) == 0 {
		return fields
	}

	for _, name := range schema.Required {
		prop, ok := schema.Properties[name]
		if !ok {
			continue
		}

		childPath := name
		if prefix != "" {
			childPath = prefix + "." + name
		}

		fields = append(fields, extractRequiredFields(prop, childPath)...)
	}

	return fields
}

// promptForValues iterates over the required fields, prompts the user for each value,
// and returns a nested map ready for YAML marshaling.
func promptForValues(requiredFields []models.RequiredField, uiService interfaces.UiServiceInterface) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	for _, field := range requiredFields {
		prompt := field.Path
		if field.Description != "" {
			prompt = fmt.Sprintf("%s (%s)", field.Path, field.Description)
		}

		var typedValue interface{}

		if len(field.Enum) > 0 {
			// Enum: present only valid options so the user cannot enter an invalid value.
			raw, err := uiService.CreateSelect(prompt, field.Enum)
			if err != nil {
				return nil, fmt.Errorf("failed to get value for %s: %w", field.Path, err)
			}
			typedValue = raw
		} else if field.Type == "boolean" {
			// Boolean: select between true/false — no free-text mistakes possible.
			raw, err := uiService.CreateSelect(prompt, []string{"true", "false"})
			if err != nil {
				return nil, fmt.Errorf("failed to get value for %s: %w", field.Path, err)
			}
			typedValue = raw == "true"
		} else {
			// string / integer / number: free-text with type validation; re-prompt on bad input.
			for {
				raw, err := uiService.CreateInput(prompt, "")
				if err != nil {
					return nil, fmt.Errorf("failed to get value for %s: %w", field.Path, err)
				}
				converted, err := validateAndConvertInput(field.Type, raw)
				if err != nil {
					logger.Warning(fmt.Sprintf("Invalid value for %s: %v — please try again.", field.Path, err))
					continue
				}
				typedValue = converted
				break
			}
		}

		setNestedValue(values, strings.Split(field.Path, "."), typedValue)
	}
	return values, nil
}


func setNestedValue(m map[string]interface{}, parts []string, value interface{}) {
	if len(parts) == 1 {
		m[parts[0]] = value
		return
	}

	if _, ok := m[parts[0]]; !ok {
		m[parts[0]] = make(map[string]interface{})
	}

	setNestedValue(m[parts[0]].(map[string]interface{}), parts[1:], value)
}

// validateAndConvertInput converts and validates a raw string against a JSON Schema type.
// Returns the properly typed Go value ready for YAML marshaling.
func validateAndConvertInput(fieldType, input string) (interface{}, error) {
	switch fieldType {
	case "boolean":
		if input != "true" && input != "false" {
			return nil, fmt.Errorf("expected 'true' or 'false', got %q", input)
		}
		return input == "true", nil
	case "integer":
		v, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("expected an integer, got %q", input)
		}
		return int(v), nil
	case "number":
		v, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return nil, fmt.Errorf("expected a number, got %q", input)
		}
		return v, nil
	default: // "string" or unset
		return input, nil
	}
}

// ensureValuesFile checks whether a values file for the package already exists in ~/.bbe/.
// If not, it fetches the chart's values.schema.json, prompts for each required field,
// and writes the result to ~/.bbe/<packageName>-values.yaml.
// Returns the file path, or "" if no required fields exist for the package.
func ensureValuesFile(packageName string, helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface) (string, error) {
	valuesFilePath := helperService.GetChartValuesFilePath(packageName)

	if _, exists := helperService.CheckIfFileExists(valuesFilePath); exists {
		logger.Debug(fmt.Sprintf("Re-using existing values file for `%s`: %s", packageName, valuesFilePath))
		return valuesFilePath, nil
	}

	rawSchema, compiledSchema, err := fetchRemoteValuesSchema(packageName)
	if err != nil {
		logger.Debug(fmt.Sprintf("No values schema found for `%s`, proceeding without values file: %v", packageName, err))
		return "", nil
	}

	requiredFields := extractRequiredFields(rawSchema, "")
	if len(requiredFields) == 0 {
		return "", nil
	}

	logger.Info(fmt.Sprintf("Package `%s` requires configuration. Please provide the following values:", packageName))

	values, err := promptForValues(requiredFields, uiService)
	if err != nil {
		return "", err
	}

	// Final validation: run the assembled values through the compiled JSON Schema.
	// This catches any constraint the per-field prompting may not have covered
	// (e.g. minLength, pattern, minimum/maximum).
	result := compiledSchema.Validate(values)
	if !result.IsValid() {
		details, _ := json.MarshalIndent(result.ToList(), "", "  ")
		return "", fmt.Errorf("provided values do not satisfy the schema for `%s`:\n%s", packageName, string(details))
	}

	yamlBytes, err := yamlMarshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to marshal values for `%s`: %w", packageName, err)
	}

	if err := osMkdirAll(helperService.GetChartValuesDir(), 0700); err != nil {
		return "", fmt.Errorf("failed to create chart values directory: %w", err)
	}

	if err := osWriteFile(valuesFilePath, yamlBytes, 0600); err != nil {
		return "", fmt.Errorf("failed to write values file for `%s`: %w", packageName, err)
	}

	logger.Info(fmt.Sprintf("Configuration saved to %s", valuesFilePath))
	return valuesFilePath, nil
}
