package package_service

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gopkg.in/yaml.v2"
)

func Test_GetAll_Succeeds(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(`library:
  - minBbeCli: "0.0.1"
    charts:
      - name: "blocky"
        version: "0.1.3"
        repositoryName: "blocky"
        repositoryUrl: "https://k8s-at-home.com/charts/"
      - name: "ingress-nginx"
        version: "4.12.0"
        repositoryName: "ingress-nginx"
        repositoryUrl: "https://kubernetes.github.io/ingress-nginx"`))
	}))
	defer ts.Close()

	// Override the BbeLibraryUrl constant to point to our test server
	originalUrl := constants.BbeLibraryUrl
	constants.BbeLibraryUrl = ts.URL
	defer func() { constants.BbeLibraryUrl = originalUrl }()

	packagesService := PackageService{}

	// Get all packages directly from the service
	result, err := packagesService.GetAll()
	if err != nil {
		t.Fatalf("Failed to get all packages: %v", err)
	}

	// Assert that the result contains the correct package data
	assert.Len(t, result, 2) // We have two packages in the mock library.yaml
	assert.Equal(t, "blocky", result[0].Name)
	assert.Equal(t, "0.1.3", result[0].Version)
	assert.Equal(t, "ingress-nginx", result[1].Name)
	assert.Equal(t, "4.12.0", result[1].Version)
}

func Test_GetAll_Fails(t *testing.T) {
	// Create a test server that returns our mock library.yaml
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return our mock library.yaml content
		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(`Not today, not today`))
	}))
	defer ts.Close()

	// Override the BbeLibraryUrl constant to point to our test server
	originalUrl := constants.BbeLibraryUrl
	constants.BbeLibraryUrl = ts.URL
	defer func() { constants.BbeLibraryUrl = originalUrl }()

	packagesService := PackageService{}

	// Get all packages directly from the service
	result, err := packagesService.GetAll()

	// Assert that the result contains the correct package data
	assert.Empty(t, result)
	assert.Error(t, err)
}

func Test_InstallPackage_Fails_WhenHelmRepositoryNotFound(t *testing.T) {
	mockErrorMessage := "Mock failed to add"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	mockUiService := &mocks.MockUiService{}
	mockHelperService := &mocks.MockHelperService{}
	err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService, mockUiService, mockHelperService)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_InstallPackage_Fails_WhenEnsureValuesFileFails(t *testing.T) {
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["host"],
		"properties": {
			"host": { "type": "string" }
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(schemaJSON))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("GetChartValuesFilePath", mock.Anything).Return("/tmp/test-values.yaml")
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return((*time.Time)(nil), false)

	mockUiService := &mocks.MockUiService{}
	mockUiService.On("CreateInput", mock.Anything, mock.Anything).Return("", errors.New("input cancelled"))

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := PackageService{}.InstallPackage(models.ChartEntry{Name: "test-chart", Version: "0.1.0"}, bbeConfig, mockHelmService, mockUiService, mockHelperService)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to gather required values")
}

func Test_InstallPackage_Fails_WhenHelmInstallFails(t *testing.T) {
	mockErrorMessage := "Mock failed to install"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}
	mockUiService := &mocks.MockUiService{}
	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("GetChartValuesFilePath", mock.Anything).Return("/mock/.bbe/chart-values/ingress-nginx-values.yaml")
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return(&now, true)

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService, mockUiService, mockHelperService)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_InstallPackage_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	packagesService := PackageService{}
	mockUiService := &mocks.MockUiService{}
	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("GetChartValuesFilePath", mock.Anything).Return("/mock/.bbe/chart-values/ingress-nginx-values.yaml")
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return(&now, true)

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService, mockUiService, mockHelperService)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_InstallPackage_Skips_Already_Installed_And_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"

	mockUiService := &mocks.MockUiService{}
	mockHelperService := &mocks.MockHelperService{}
	err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService, mockUiService, mockHelperService)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_UpgradePackage_Fails_WhenEnsureValuesFileFails(t *testing.T) {
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["host"],
		"properties": {
			"host": { "type": "string" }
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(schemaJSON))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)

	mockHelperService := &mocks.MockHelperService{}
	mockHelperService.On("GetChartValuesFilePath", mock.Anything).Return("/tmp/test-values.yaml")
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return((*time.Time)(nil), false)

	mockUiService := &mocks.MockUiService{}
	mockUiService.On("CreateInput", mock.Anything, mock.Anything).Return("", errors.New("input cancelled"))

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := PackageService{}.UpgradePackage(models.ChartEntry{Name: "test-chart", Version: "0.1.0"}, bbeConfig, mockHelmService, mockUiService, mockHelperService)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to gather required values")
}

func Test_UpgradePackage_Fails_WhenHelmRepositoryNotFound(t *testing.T) {
	mockErrorMessage := "Mockfailed adding repo"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	mockUiService := &mocks.MockUiService{}
	mockHelperService := &mocks.MockHelperService{}
	err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService, mockUiService, mockHelperService)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_UpgradePackage_Fails_WhenHelmUpgradeFails(t *testing.T) {
	mockErrorMessage := "Mockfailed upgrading repo"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}
	mockUiService := &mocks.MockUiService{}
	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("GetChartValuesFilePath", mock.Anything).Return("/mock/.bbe/chart-values/ingress-nginx-values.yaml")
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return(&now, true)

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService, mockUiService, mockHelperService)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_UpgradePackage_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	packagesService := PackageService{}
	mockUiService := &mocks.MockUiService{}
	mockHelperService := &mocks.MockHelperService{}
	now := time.Now()
	mockHelperService.On("GetChartValuesFilePath", mock.Anything).Return("/mock/.bbe/chart-values/ingress-nginx-values.yaml")
	mockHelperService.On("CheckIfFileExists", mock.Anything).Return(&now, true)

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService, mockUiService, mockHelperService)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_UpgradePackage_Fails_WhenPackageNotInstalled(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	mockUiService := &mocks.MockUiService{}
	mockHelperService := &mocks.MockHelperService{}
	err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService, mockUiService, mockHelperService)

	// Assert an error occurred
	assert.Error(t, err)
}

func Test_UninstallPackage_Fails_WhenHelmFails(t *testing.T) {
	mockErrorMessage := "Mockfailed uninstalling repo"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("UninstallChart", mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UninstallPackage(models.LocalPackage{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_UninstallPackage_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("UninstallChart", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UninstallPackage(models.LocalPackage{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_UninstallPackage_Fails_WhenPackageNotInstalled(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UninstallPackage(models.LocalPackage{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService)

	// Assert an error occurred
	assert.Error(t, err)
}

func Test_getRemoteLibrary_Fails_WhenInvalidProtocol(t *testing.T) {

	// Override the BbeLibraryUrl constant to point to our test server
	originalUrl := constants.BbeLibraryUrl
	constants.BbeLibraryUrl = "thisIsNotReal"
	defer func() { constants.BbeLibraryUrl = originalUrl }()

	mockErrorMessage := "unsupported protocol scheme"

	result, err := getRemoteLibrary()

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_getRemoteLibrary_Fails_WhenIOReadFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(`test`))
	}))
	defer ts.Close()

	// Override the BbeLibraryUrl constant to point to our test server
	originalUrl := constants.BbeLibraryUrl
	constants.BbeLibraryUrl = ts.URL
	defer func() { constants.BbeLibraryUrl = originalUrl }()

	ioReadAll = func(r io.Reader) ([]byte, error) {
		return nil, errors.New("Mock failed to read")
	}

	defer func() { ioReadAll = io.ReadAll }()

	result, err := getRemoteLibrary()

	assert.Error(t, err)
	assert.Nil(t, result)
}

func Test_getRemoteLibrary_Fails_WhenNoFileContent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(``))
	}))
	defer ts.Close()

	// Override the BbeLibraryUrl constant to point to our test server
	originalUrl := constants.BbeLibraryUrl
	constants.BbeLibraryUrl = ts.URL
	defer func() { constants.BbeLibraryUrl = originalUrl }()

	result, err := getRemoteLibrary()

	assert.Error(t, err)
	assert.Nil(t, result)
}

func Test_validateAndConvertInput_String(t *testing.T) {
	v, err := validateAndConvertInput("string", "hello")
	assert.NoError(t, err)
	assert.Equal(t, "hello", v)
}

func Test_validateAndConvertInput_StringUntyped(t *testing.T) {
	v, err := validateAndConvertInput("", "anything")
	assert.NoError(t, err)
	assert.Equal(t, "anything", v)
}

func Test_validateAndConvertInput_BooleanTrue(t *testing.T) {
	v, err := validateAndConvertInput("boolean", "true")
	assert.NoError(t, err)
	assert.Equal(t, true, v)
}

func Test_validateAndConvertInput_BooleanFalse(t *testing.T) {
	v, err := validateAndConvertInput("boolean", "false")
	assert.NoError(t, err)
	assert.Equal(t, false, v)
}

func Test_validateAndConvertInput_Boolean_Invalid(t *testing.T) {
	v, err := validateAndConvertInput("boolean", "yes")
	assert.Error(t, err)
	assert.Nil(t, v)
}

func Test_validateAndConvertInput_Integer(t *testing.T) {
	v, err := validateAndConvertInput("integer", "42")
	assert.NoError(t, err)
	assert.Equal(t, 42, v)
}

func Test_validateAndConvertInput_Integer_Invalid(t *testing.T) {
	v, err := validateAndConvertInput("integer", "3.14")
	assert.Error(t, err)
	assert.Nil(t, v)
}

func Test_validateAndConvertInput_Number(t *testing.T) {
	v, err := validateAndConvertInput("number", "3.14")
	assert.NoError(t, err)
	assert.Equal(t, 3.14, v)
}

func Test_validateAndConvertInput_Number_Invalid(t *testing.T) {
	v, err := validateAndConvertInput("number", "notanumber")
	assert.Error(t, err)
	assert.Nil(t, v)
}

func Test_extractRequiredFields_String(t *testing.T) {
	schema := models.RawSchemaNode{
		Type:     "object",
		Required: []string{"host"},
		Properties: map[string]models.RawSchemaNode{
			"host": {Type: "string", Description: "The hostname"},
		},
	}
	fields := extractRequiredFields(schema, "")
	assert.Len(t, fields, 1)
	assert.Equal(t, "host", fields[0].Path)
	assert.Equal(t, "string", fields[0].Type)
	assert.Equal(t, "The hostname", fields[0].Description)
	assert.Empty(t, fields[0].Enum)
}

func Test_extractRequiredFields_Boolean(t *testing.T) {
	schema := models.RawSchemaNode{
		Type:     "object",
		Required: []string{"enabled"},
		Properties: map[string]models.RawSchemaNode{
			"enabled": {Type: "boolean"},
		},
	}
	fields := extractRequiredFields(schema, "")
	assert.Len(t, fields, 1)
	assert.Equal(t, "enabled", fields[0].Path)
	assert.Equal(t, "boolean", fields[0].Type)
}

func Test_extractRequiredFields_Integer(t *testing.T) {
	schema := models.RawSchemaNode{
		Type:     "object",
		Required: []string{"port"},
		Properties: map[string]models.RawSchemaNode{
			"port": {Type: "integer"},
		},
	}
	fields := extractRequiredFields(schema, "")
	assert.Len(t, fields, 1)
	assert.Equal(t, "port", fields[0].Path)
	assert.Equal(t, "integer", fields[0].Type)
}

func Test_extractRequiredFields_Enum(t *testing.T) {
	schema := models.RawSchemaNode{
		Type:     "object",
		Required: []string{"mode"},
		Properties: map[string]models.RawSchemaNode{
			"mode": {Type: "string", Enum: []string{"alpha", "beta", "stable"}},
		},
	}
	fields := extractRequiredFields(schema, "")
	assert.Len(t, fields, 1)
	assert.Equal(t, "mode", fields[0].Path)
	assert.Equal(t, []string{"alpha", "beta", "stable"}, fields[0].Enum)
}

func Test_extractRequiredFields_Nested(t *testing.T) {
	schema := models.RawSchemaNode{
		Type:     "object",
		Required: []string{"bbe"},
		Properties: map[string]models.RawSchemaNode{
			"bbe": {
				Type:     "object",
				Required: []string{"ip"},
				Properties: map[string]models.RawSchemaNode{
					"ip": {Type: "string"},
				},
			},
		},
	}
	fields := extractRequiredFields(schema, "")
	assert.Len(t, fields, 1)
	assert.Equal(t, "bbe.ip", fields[0].Path)
	assert.Equal(t, "string", fields[0].Type)
}

func Test_extractRequiredFields_SkipsMissingProperty(t *testing.T) {
	// "ghost" is in required but absent from properties — the branch should be skipped.
	schema := models.RawSchemaNode{
		Type:     "object",
		Required: []string{"ghost", "host"},
		Properties: map[string]models.RawSchemaNode{
			"host": {Type: "string"},
		},
	}
	fields := extractRequiredFields(schema, "")
	assert.Len(t, fields, 1)
	assert.Equal(t, "host", fields[0].Path)
}

func Test_ensureValuesFile_FailsSchemaValidation_WhenValuesViolateConstraints(t *testing.T) {
	// Schema requires `host` as a string with minLength:5.
	// We'll mock the user returning a 2-char value, which should fail final schema validation.
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["host"],
		"properties": {
			"host": {
				"type": "string",
				"minLength": 5
			}
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(schemaJSON))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	mockHelper := &mocks.MockHelperService{}
	mockHelper.On("GetChartValuesFilePath", mock.Anything).Return("/tmp/test-values.yaml")
	mockHelper.On("CheckIfFileExists", mock.Anything).Return((*time.Time)(nil), false)

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", mock.Anything, mock.Anything).Return("ab", nil) // too short

	_, err := ensureValuesFile("test-chart", mockHelper, mockUi)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "do not satisfy the schema")
}

func Test_ensureValuesFile_Succeeds_WhenValuesPassSchemaValidation(t *testing.T) {
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["host"],
		"properties": {
			"host": {
				"type": "string",
				"minLength": 3
			}
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(schemaJSON))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	tmpDir := t.TempDir()
	valuesPath := tmpDir + "/test-chart-values.yaml"

	mockHelper := &mocks.MockHelperService{}
	mockHelper.On("GetChartValuesFilePath", mock.Anything).Return(valuesPath)
	mockHelper.On("CheckIfFileExists", mock.Anything).Return((*time.Time)(nil), false)
	mockHelper.On("GetChartValuesDir", mock.Anything).Return(tmpDir)

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", mock.Anything, mock.Anything).Return("hello", nil)

	path, err := ensureValuesFile("test-chart", mockHelper, mockUi)

	assert.NoError(t, err)
	assert.Equal(t, valuesPath, path)
}

func Test_ensureValuesFile_Fails_WhenMkdirAllFails(t *testing.T) {
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["host"],
		"properties": {
			"host": { "type": "string" }
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(schemaJSON))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	osMkdirAll = func(path string, perm os.FileMode) error {
		return errors.New("mock mkdir failure")
	}
	defer func() { osMkdirAll = os.MkdirAll }()

	mockHelper := &mocks.MockHelperService{}
	mockHelper.On("GetChartValuesFilePath", mock.Anything).Return("/tmp/test-chart-values.yaml")
	mockHelper.On("CheckIfFileExists", mock.Anything).Return((*time.Time)(nil), false)
	mockHelper.On("GetChartValuesDir", mock.Anything).Return("/tmp/bbe")

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", mock.Anything, mock.Anything).Return("hello", nil)

	_, err := ensureValuesFile("test-chart", mockHelper, mockUi)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create chart values directory")
}

func Test_ensureValuesFile_Fails_WhenWriteFileFails(t *testing.T) {
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["host"],
		"properties": {
			"host": { "type": "string" }
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(schemaJSON))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	osMkdirAll = func(path string, perm os.FileMode) error { return nil }
	defer func() { osMkdirAll = os.MkdirAll }()

	osWriteFile = func(name string, data []byte, perm os.FileMode) error {
		return errors.New("mock write failure")
	}
	defer func() { osWriteFile = os.WriteFile }()

	tmpDir := t.TempDir()
	mockHelper := &mocks.MockHelperService{}
	mockHelper.On("GetChartValuesFilePath", mock.Anything).Return(tmpDir + "/test-chart-values.yaml")
	mockHelper.On("CheckIfFileExists", mock.Anything).Return((*time.Time)(nil), false)
	mockHelper.On("GetChartValuesDir", mock.Anything).Return(tmpDir)

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", mock.Anything, mock.Anything).Return("hello", nil)

	_, err := ensureValuesFile("test-chart", mockHelper, mockUi)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write values file")
}

func Test_ensureValuesFile_Fails_WhenYamlMarshalFails(t *testing.T) {
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["host"],
		"properties": {
			"host": { "type": "string" }
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(schemaJSON))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	yamlMarshal = func(v interface{}) ([]byte, error) {
		return nil, errors.New("mock marshal failure")
	}
	defer func() { yamlMarshal = yaml.Marshal }()

	mockHelper := &mocks.MockHelperService{}
	mockHelper.On("GetChartValuesFilePath", mock.Anything).Return("/tmp/test-chart-values.yaml")
	mockHelper.On("CheckIfFileExists", mock.Anything).Return((*time.Time)(nil), false)

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", mock.Anything, mock.Anything).Return("hello", nil)

	_, err := ensureValuesFile("test-chart", mockHelper, mockUi)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal values")
}

func Test_fetchRemoteValuesSchema_Fails_WhenNetworkError(t *testing.T) {
	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = "thisIsNotReal/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	_, _, err := fetchRemoteValuesSchema("test-chart")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch schema")
}

func Test_fetchRemoteValuesSchema_Fails_WhenReadBodyFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	ioReadAll = func(r io.Reader) ([]byte, error) {
		return nil, errors.New("mock read failure")
	}
	defer func() { ioReadAll = io.ReadAll }()

	_, _, err := fetchRemoteValuesSchema("test-chart")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock read failure")
}

func Test_fetchRemoteValuesSchema_Fails_WhenSchemaInvalid(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{not valid json`))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	_, _, err := fetchRemoteValuesSchema("test-chart")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON Schema")
}

func Test_fetchRemoteValuesSchema_Fails_WhenUnmarshalFails(t *testing.T) {
	// JSON Schema boolean `true` is a valid schema (accepts all values) but cannot
	// be unmarshalled into models.RawSchemaNode (a struct).
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`true`))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	_, _, err := fetchRemoteValuesSchema("test-chart")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse schema")
}

func Test_ensureValuesFile_ReturnsEmpty_WhenSchemaFetchFails(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	mockHelper := &mocks.MockHelperService{}
	mockHelper.On("GetChartValuesFilePath", mock.Anything).Return("/tmp/test-chart-values.yaml")
	mockHelper.On("CheckIfFileExists", mock.Anything).Return((*time.Time)(nil), false)

	path, err := ensureValuesFile("test-chart", mockHelper, &mocks.MockUiService{})

	assert.NoError(t, err)
	assert.Empty(t, path)
}

func Test_ensureValuesFile_ReturnsEmpty_WhenNoRequiredFields(t *testing.T) {
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"host": { "type": "string" }
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(schemaJSON))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	mockHelper := &mocks.MockHelperService{}
	mockHelper.On("GetChartValuesFilePath", mock.Anything).Return("/tmp/test-chart-values.yaml")
	mockHelper.On("CheckIfFileExists", mock.Anything).Return((*time.Time)(nil), false)

	path, err := ensureValuesFile("test-chart", mockHelper, &mocks.MockUiService{})

	assert.NoError(t, err)
	assert.Empty(t, path)
}

func Test_promptForValues_String(t *testing.T) {
	fields := []models.RequiredField{{Path: "host", Type: "string"}}

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", "host", "").Return("myhost", nil)

	values, err := promptForValues(fields, mockUi)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"host": "myhost"}, values)
	mockUi.AssertExpectations(t)
}

func Test_promptForValues_String_WithDescription(t *testing.T) {
	fields := []models.RequiredField{{Path: "host", Type: "string", Description: "The hostname"}}

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", "host (The hostname)", "").Return("myhost", nil)

	values, err := promptForValues(fields, mockUi)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"host": "myhost"}, values)
	mockUi.AssertExpectations(t)
}

func Test_promptForValues_Boolean_True(t *testing.T) {
	fields := []models.RequiredField{{Path: "enabled", Type: "boolean"}}

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateSelect", "enabled", []string{"true", "false"}).Return("true", nil)

	values, err := promptForValues(fields, mockUi)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"enabled": true}, values)
	mockUi.AssertExpectations(t)
}

func Test_promptForValues_Boolean_False(t *testing.T) {
	fields := []models.RequiredField{{Path: "enabled", Type: "boolean"}}

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateSelect", "enabled", []string{"true", "false"}).Return("false", nil)

	values, err := promptForValues(fields, mockUi)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"enabled": false}, values)
	mockUi.AssertExpectations(t)
}

func Test_promptForValues_Enum(t *testing.T) {
	fields := []models.RequiredField{{Path: "mode", Type: "string", Enum: []string{"alpha", "beta"}}}

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateSelect", "mode", []string{"alpha", "beta"}).Return("beta", nil)

	values, err := promptForValues(fields, mockUi)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"mode": "beta"}, values)
	mockUi.AssertExpectations(t)
}

func Test_promptForValues_Integer_RepromptsOnInvalid(t *testing.T) {
	fields := []models.RequiredField{{Path: "port", Type: "integer"}}

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", "port", "").Return("notanumber", nil).Once()
	mockUi.On("CreateInput", "port", "").Return("8080", nil).Once()

	values, err := promptForValues(fields, mockUi)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"port": 8080}, values)
	mockUi.AssertExpectations(t)
}

func Test_promptForValues_Fails_WhenCreateInputFails(t *testing.T) {
	fields := []models.RequiredField{{Path: "host", Type: "string"}}

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", "host", "").Return("", errors.New("input cancelled"))

	_, err := promptForValues(fields, mockUi)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get value for host")
	mockUi.AssertExpectations(t)
}

func Test_promptForValues_Fails_WhenCreateSelectFails_Enum(t *testing.T) {
	fields := []models.RequiredField{{Path: "mode", Type: "string", Enum: []string{"a", "b"}}}

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateSelect", "mode", []string{"a", "b"}).Return("", errors.New("select cancelled"))

	_, err := promptForValues(fields, mockUi)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get value for mode")
	mockUi.AssertExpectations(t)
}

func Test_promptForValues_Fails_WhenCreateSelectFails_Boolean(t *testing.T) {
	fields := []models.RequiredField{{Path: "enabled", Type: "boolean"}}

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateSelect", "enabled", []string{"true", "false"}).Return("", errors.New("select cancelled"))

	_, err := promptForValues(fields, mockUi)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get value for enabled")
	mockUi.AssertExpectations(t)
}

func Test_promptForValues_Nested(t *testing.T) {
	fields := []models.RequiredField{{Path: "bbe.ip", Type: "string"}}

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", "bbe.ip", "").Return("192.168.1.1", nil)

	values, err := promptForValues(fields, mockUi)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{
		"bbe": map[string]interface{}{"ip": "192.168.1.1"},
	}, values)
	mockUi.AssertExpectations(t)
}

func Test_ensureValuesFile_Fails_WhenPromptForValuesFails(t *testing.T) {
	schemaJSON := `{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"required": ["host"],
		"properties": {
			"host": { "type": "string" }
		}
	}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(schemaJSON))
	}))
	defer ts.Close()

	originalUrl := constants.BbeChartSchemaUrl
	constants.BbeChartSchemaUrl = ts.URL + "/%s"
	defer func() { constants.BbeChartSchemaUrl = originalUrl }()

	mockHelper := &mocks.MockHelperService{}
	mockHelper.On("GetChartValuesFilePath", mock.Anything).Return("/tmp/test-chart-values.yaml")
	mockHelper.On("CheckIfFileExists", mock.Anything).Return((*time.Time)(nil), false)

	mockUi := &mocks.MockUiService{}
	mockUi.On("CreateInput", mock.Anything, mock.Anything).Return("", errors.New("input cancelled"))

	_, err := ensureValuesFile("test-chart", mockHelper, mockUi)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get value for host")
	mockUi.AssertExpectations(t)
}
