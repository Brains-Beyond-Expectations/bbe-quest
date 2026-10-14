package package_service

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_GetAll_Succeeds(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(`library:
  - min-bbe-cli: "0.0.1"
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
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(&models.HelmChart{}, nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	_, err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, nil, bbeConfig, mockHelmService, &mocks.MockUiService{})

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_InstallPackage_Fails_WhenHelmInstallFails(t *testing.T) {
	mockErrorMessage := "Mock failed to install"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(&models.HelmChart{}, nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	_, err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, nil, bbeConfig, mockHelmService, &mocks.MockUiService{})

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_InstallPackage_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(&models.HelmChart{}, nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	_, err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, nil, bbeConfig, mockHelmService, &mocks.MockUiService{})

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_InstallPackage_Skips_Already_Installed_And_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"

	_, err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, nil, bbeConfig, mockHelmService, &mocks.MockUiService{})

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_UpgradePackage_Fails_WhenHelmRepositoryNotFound(t *testing.T) {
	mockErrorMessage := "Mockfailed adding repo"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(&models.HelmChart{}, nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	_, err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, nil, bbeConfig, mockHelmService, &mocks.MockUiService{}, true)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_UpgradePackage_Fails_WhenHelmUpgradeFails(t *testing.T) {
	mockErrorMessage := "Mockfailed upgrading repo"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(&models.HelmChart{}, nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	_, err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, nil, bbeConfig, mockHelmService, &mocks.MockUiService{}, true)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_UpgradePackage_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(&models.HelmChart{}, nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	_, err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, nil, bbeConfig, mockHelmService, &mocks.MockUiService{}, true)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_UpgradePackage_Fails_WhenPackageNotInstalled(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	_, err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, nil, bbeConfig, mockHelmService, &mocks.MockUiService{}, true)

	// Assert an error occurred
	assert.Error(t, err)
}

func Test_getRemoteLibrary_Fails_WhenTheLibraryFileIsMissing(t *testing.T) {
	originalUrl := constants.BbeLibraryUrl
	constants.BbeLibraryUrl = "thisIsNotReal"
	defer func() { constants.BbeLibraryUrl = originalUrl }()

	result, err := getRemoteLibrary()

	assert.Nil(t, result)
	assert.ErrorContains(t, err, "open thisIsNotReal: no such file or directory")
}

func Test_getRemoteLibrary_Fails_WhenTheLibraryIsNotFound(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	defer ts.Close()

	originalUrl := constants.BbeLibraryUrl
	constants.BbeLibraryUrl = ts.URL + "/library.yaml"
	defer func() { constants.BbeLibraryUrl = originalUrl }()

	result, err := getRemoteLibrary()

	assert.Nil(t, result)
	assert.EqualError(t, err, "Failed to fetch the package library from "+ts.URL+"/library.yaml: 404 Not Found")
}

func Test_getRemoteLibrary_Fails_WhenTheServerCannotBeReached(t *testing.T) {
	originalUrl := constants.BbeLibraryUrl
	constants.BbeLibraryUrl = "http://127.0.0.1:1/library.yaml"
	defer func() { constants.BbeLibraryUrl = originalUrl }()

	result, err := getRemoteLibrary()

	assert.Nil(t, result)
	assert.ErrorContains(t, err, "connection refused")
}

func Test_getRemoteLibrary_Succeeds_ReadingAFileWithChartFolders(t *testing.T) {
	dir := t.TempDir()
	library := filepath.Join(dir, "library.yaml")
	os.WriteFile(library, []byte(`library:
  - min-bbe-cli: "0.0.0"
    charts:
      - name: bbe-networking
        version: 0.0.0-local
        repositoryUrl: file://charts/bbe-networking
        repositoryName: local
      - name: bbe-storage
        version: 0.0.0-local
        repositoryUrl: file:///srv/bbe-storage
        repositoryName: local
      - name: blocky
        version: 1.1.1
        repositoryUrl: https://bbe.example
        repositoryName: bbe
`), 0644)

	for _, source := range []string{library, "file://" + library} {
		originalUrl := constants.BbeLibraryUrl
		constants.BbeLibraryUrl = source

		result, err := getRemoteLibrary()

		constants.BbeLibraryUrl = originalUrl
		assert.NoError(t, err, source)
		assert.Equal(t, "file://"+filepath.Join(dir, "charts", "bbe-networking"), result.Charts[0].RepositoryUrl, source)
		assert.Equal(t, "file:///srv/bbe-storage", result.Charts[1].RepositoryUrl, source)
		assert.Equal(t, "https://bbe.example", result.Charts[2].RepositoryUrl, source)
	}
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

func Test_InstallPackage_Succeeds_AskingForRequiredValues(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("PullChart", "https://example.com/charts", "bbe-networking", "0.3.0").Return(chartRequiring("service", "ip"), nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockUiService := &mocks.MockUiService{}
	mockUiService.On("CreateInput", mock.Anything, mock.Anything).Return("192.168.1.240", nil)

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	chart := models.ChartEntry{Name: "bbe-networking", Version: "0.3.0", RepositoryName: "bbe", RepositoryUrl: "https://example.com/charts"}
	values, err := packagesService.InstallPackage(chart, map[string]interface{}{"other": "kept"}, bbeConfig, mockHelmService, mockUiService)

	expected := map[string]interface{}{
		"other":   "kept",
		"service": map[string]interface{}{"ip": "192.168.1.240"},
	}
	assert.NoError(t, err)
	assert.Equal(t, expected, values)
	mockUiService.AssertNumberOfCalls(t, "CreateInput", 1)
	mockHelmService.AssertCalled(t, "InstallChart", "bbe-networking", "bbe-networking", "bbe", "0.3.0", "bbe-networking", "test-context", expected)
}

func Test_InstallPackage_Skips_Already_Installed_And_Keeps_Values(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)

	packagesService := PackageService{}

	stored := map[string]interface{}{"service": map[string]interface{}{"ip": "192.168.1.240"}}
	values, err := packagesService.InstallPackage(models.ChartEntry{Name: "bbe-networking"}, stored, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{})

	assert.NoError(t, err)
	assert.Equal(t, stored, values)
	mockHelmService.AssertNotCalled(t, "PullChart", mock.Anything, mock.Anything, mock.Anything)
}

func Test_InstallPackage_Fails_WhenChartCannotBePulled(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("Mock failed to pull"))

	packagesService := PackageService{}

	values, err := packagesService.InstallPackage(models.ChartEntry{Name: "bbe-networking"}, nil, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{})

	assert.Nil(t, values)
	assert.EqualError(t, err, "Mock failed to pull")
	mockHelmService.AssertNotCalled(t, "AddRepo", mock.Anything, mock.Anything)
}

func Test_UpgradePackage_Succeeds_WithStoredValues(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartRequiring("service", "ip"), nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockUiService := &mocks.MockUiService{}

	packagesService := PackageService{}

	// As read back from bbe.yaml by yaml.v2
	stored := map[string]interface{}{"service": map[interface{}]interface{}{"ip": "192.168.1.240"}}
	values, err := packagesService.UpgradePackage(models.ChartEntry{Name: "bbe-networking"}, stored, models.BbeConfig{}, mockHelmService, mockUiService, false)

	expected := map[string]interface{}{"service": map[string]interface{}{"ip": "192.168.1.240"}}
	assert.NoError(t, err)
	assert.Equal(t, expected, values)
	mockUiService.AssertNotCalled(t, "CreateInput", mock.Anything, mock.Anything)
	mockHelmService.AssertCalled(t, "UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, expected)
}

func Test_UpgradePackage_Fails_WhenRequiredValuesAreMissingAndNotInteractive(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartRequiring("service", "ip"), nil)
	mockUiService := &mocks.MockUiService{}

	packagesService := PackageService{}

	values, err := packagesService.UpgradePackage(models.ChartEntry{Name: "bbe-networking"}, nil, models.BbeConfig{}, mockHelmService, mockUiService, false)

	assert.Nil(t, values)
	assert.EqualError(t, err, "Package `bbe-networking` requires values that aren't set: `service.ip`. Run without --yes to enter them, or add them under `values` for the package in bbe.yaml")
	mockUiService.AssertNotCalled(t, "CreateInput", mock.Anything, mock.Anything)
	mockHelmService.AssertNotCalled(t, "UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func Test_GetPrerequisites_Succeeds_FromTheChart(t *testing.T) {
	prerequisites := []models.ChartPrerequisite{{Name: "storage", StorageClass: "longhorn", Package: "bbe-storage"}}
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("PullChart", "https://example.com/charts", "bbe-media", "1.0.0").Return(&models.HelmChart{Prerequisites: prerequisites}, nil)

	result, err := PackageService{}.GetPrerequisites(models.ChartEntry{Name: "bbe-media", Version: "1.0.0", RepositoryUrl: "https://example.com/charts"}, mockHelmService)

	assert.NoError(t, err)
	assert.Equal(t, prerequisites, result)
}

func Test_GetPrerequisites_Fails_WhenTheChartCannotBePulled(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("Mock failed to pull"))

	result, err := PackageService{}.GetPrerequisites(models.ChartEntry{Name: "bbe-media"}, mockHelmService)

	assert.Nil(t, result)
	assert.EqualError(t, err, "Mock failed to pull")
}

func Test_getRemoteLibrary_Succeeds_SkipsUnsupportedRevisions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(`library:
  - min-bbe-cli: "2.0.0"
    list-revision: 3
  - min-bbe-cli: "0.10.0"
    list-revision: 2
  - min-bbe-cli: "0.6.0"
    list-revision: 1`))
	}))
	defer ts.Close()

	// Override the BbeLibraryUrl constant to point to our test server
	originalUrl := constants.BbeLibraryUrl
	constants.BbeLibraryUrl = ts.URL
	defer func() { constants.BbeLibraryUrl = originalUrl }()

	originalVersion := constants.Version
	constants.Version = "v1.0.0"
	defer func() { constants.Version = originalVersion }()

	result, err := getRemoteLibrary()

	assert.NoError(t, err)
	assert.Equal(t, 2, result.ListRevision)
}

func Test_getRemoteLibrary_Fails_WhenNoRevisionSupported(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		w.Write([]byte(`library:
  - min-bbe-cli: "2.0.0"
    list-revision: 1`))
	}))
	defer ts.Close()

	// Override the BbeLibraryUrl constant to point to our test server
	originalUrl := constants.BbeLibraryUrl
	constants.BbeLibraryUrl = ts.URL
	defer func() { constants.BbeLibraryUrl = originalUrl }()

	originalVersion := constants.Version
	constants.Version = "v1.0.0"
	defer func() { constants.Version = originalVersion }()

	result, err := getRemoteLibrary()

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "v1.0.0")
}

func Test_isRevisionSupported_Succeeds_WhenDevelopmentBuild(t *testing.T) {
	assert.True(t, isRevisionSupported("99.0.0", "development"))
}

func Test_isRevisionSupported_Succeeds_WhenNoMinimumVersion(t *testing.T) {
	assert.True(t, isRevisionSupported("", "v0.7.1"))
}

func Test_isRevisionSupported_Succeeds_WhenVersionMeetsMinimum(t *testing.T) {
	assert.True(t, isRevisionSupported("0.7.1", "v0.7.1"))
	assert.True(t, isRevisionSupported("0.6.0", "v0.7.1"))
	assert.True(t, isRevisionSupported("0.9.0", "v0.10.0"))
	assert.True(t, isRevisionSupported("v0.6.0", "v0.7.1"))
	assert.True(t, isRevisionSupported("0.6.0", "0.7.1"))
}

func Test_isRevisionSupported_Fails_WhenVersionBelowMinimum(t *testing.T) {
	assert.False(t, isRevisionSupported("2.0.0", "v1.0.0"))
	assert.False(t, isRevisionSupported("0.10.0", "v0.9.0"))
	assert.False(t, isRevisionSupported("1.0.0", "v1.0.0-rc.1"))
}

func Test_isRevisionSupported_Fails_WhenVersionInvalid(t *testing.T) {
	assert.False(t, isRevisionSupported("latest", "v1.0.0"))
	assert.False(t, isRevisionSupported("0.6.0", "not-a-version"))
}

func Test_InstallPackage_Succeeds_PreparingTheNamespaceForThePodSecurityLevel(t *testing.T) {
	chart := chartRequiring()
	chart.PodSecurity = "privileged"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chart, nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("PrepareNamespace", "bbe-networking", "test-context", "privileged").Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	_, err := PackageService{}.InstallPackage(models.ChartEntry{Name: "bbe-networking"}, nil, bbeConfig, mockHelmService, &mocks.MockUiService{})

	assert.NoError(t, err)
	mockHelmService.AssertCalled(t, "PrepareNamespace", "bbe-networking", "test-context", "privileged")
	mockHelmService.AssertCalled(t, "InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func Test_InstallPackage_Fails_WhenTheNamespaceCannotBePrepared(t *testing.T) {
	chart := chartRequiring()
	chart.PodSecurity = "privileged"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chart, nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("PrepareNamespace", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("Mock failed to label"))

	values, err := PackageService{}.InstallPackage(models.ChartEntry{Name: "bbe-networking"}, nil, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{})

	assert.Nil(t, values)
	assert.EqualError(t, err, "Mock failed to label")
	mockHelmService.AssertNotCalled(t, "InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func Test_UpgradePackage_Succeeds_PreparingTheNamespaceForThePodSecurityLevel(t *testing.T) {
	chart := chartRequiring()
	chart.PodSecurity = "privileged"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chart, nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("PrepareNamespace", "bbe-networking", "", "privileged").Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := PackageService{}.UpgradePackage(models.ChartEntry{Name: "bbe-networking"}, nil, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{}, false)

	assert.NoError(t, err)
	mockHelmService.AssertCalled(t, "PrepareNamespace", "bbe-networking", "", "privileged")
}

func Test_UpgradePackage_Fails_WhenTheChartCannotBePulled(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("Mock failed to pull"))

	values, err := PackageService{}.UpgradePackage(models.ChartEntry{Name: "bbe-networking"}, nil, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{}, false)

	assert.Nil(t, values)
	assert.EqualError(t, err, "Mock failed to pull")
}

func Test_InstallPackage_Succeeds_FromAChartFolder(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("PullChart", "file:///charts/bbe-networking", "bbe-networking", "0.0.0-local").Return(chartRequiring(), nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	chart := models.ChartEntry{Name: "bbe-networking", Version: "0.0.0-local", RepositoryUrl: "file:///charts/bbe-networking", RepositoryName: "local"}
	_, err := PackageService{}.InstallPackage(chart, nil, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{})

	assert.NoError(t, err)
	mockHelmService.AssertNotCalled(t, "AddRepo", mock.Anything, mock.Anything)
	mockHelmService.AssertCalled(t, "InstallChart", "bbe-networking", "/charts/bbe-networking", "", "", "bbe-networking", "", map[string]interface{}(nil))
}

func Test_UpgradePackage_Succeeds_FromAChartFolder(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartRequiring(), nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	chart := models.ChartEntry{Name: "bbe-networking", RepositoryUrl: "file:///charts/bbe-networking"}
	_, err := PackageService{}.UpgradePackage(chart, nil, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{}, false)

	assert.NoError(t, err)
	mockHelmService.AssertNotCalled(t, "AddRepo", mock.Anything, mock.Anything)
	mockHelmService.AssertCalled(t, "UpgradeChart", "bbe-networking", "/charts/bbe-networking", "", "", "bbe-networking", "", map[string]interface{}(nil))
}

// A chart with CRDs of its own, in a dependency that's enabled, and in one its values turn off
func chartWithCrds() *models.HelmChart {
	return &models.HelmChart{
		Name:   "bbe-networking",
		Crds:   []string{"kind: CustomResourceDefinition # parent"},
		Values: map[string]interface{}{"disabled": map[string]interface{}{"enabled": false}},
		Dependencies: []models.HelmChartDependency{
			{Name: "traefik", Chart: &models.HelmChart{Name: "traefik", Crds: []string{"kind: CustomResourceDefinition # traefik"}}},
			{Name: "disabled", Condition: "disabled.enabled", Chart: &models.HelmChart{Name: "disabled", Crds: []string{"kind: CustomResourceDefinition # disabled"}}},
		},
	}
}

func Test_UpgradePackage_Succeeds_ApplyingTheCrdsOfTheChartAndItsEnabledDependencies(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartWithCrds(), nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("ApplyCrds", "test-context", mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	_, err := PackageService{}.UpgradePackage(models.ChartEntry{Name: "bbe-networking"}, nil, bbeConfig, mockHelmService, &mocks.MockUiService{}, false)

	assert.NoError(t, err)
	mockHelmService.AssertCalled(t, "ApplyCrds", "test-context", []string{"kind: CustomResourceDefinition # parent", "kind: CustomResourceDefinition # traefik"})
	mockHelmService.AssertCalled(t, "UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func Test_UpgradePackage_Succeeds_ApplyingTheCrdsOfADependencyItsValuesTurnOn(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartWithCrds(), nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("ApplyCrds", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	values := map[string]interface{}{"disabled": map[string]interface{}{"enabled": true}}
	_, err := PackageService{}.UpgradePackage(models.ChartEntry{Name: "bbe-networking"}, values, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{}, false)

	assert.NoError(t, err)
	mockHelmService.AssertCalled(t, "ApplyCrds", "", []string{"kind: CustomResourceDefinition # parent", "kind: CustomResourceDefinition # traefik", "kind: CustomResourceDefinition # disabled"})
}

func Test_UpgradePackage_Fails_WhenTheCrdsCannotBeApplied(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartWithCrds(), nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("ApplyCrds", mock.Anything, mock.Anything).Return(errors.New("Mock failed to apply"))

	values, err := PackageService{}.UpgradePackage(models.ChartEntry{Name: "bbe-networking"}, nil, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{}, false)

	assert.Nil(t, values)
	assert.EqualError(t, err, "Mock failed to apply")
	mockHelmService.AssertNotCalled(t, "UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func Test_InstallPackage_Succeeds_ApplyingTheCrdsFirst(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartWithCrds(), nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("ApplyCrds", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	_, err := PackageService{}.InstallPackage(models.ChartEntry{Name: "bbe-networking"}, nil, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{})

	assert.NoError(t, err)
	mockHelmService.AssertCalled(t, "ApplyCrds", "", []string{"kind: CustomResourceDefinition # parent", "kind: CustomResourceDefinition # traefik"})
}

func Test_InstallPackage_Fails_WhenTheCrdsCannotBeApplied(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartWithCrds(), nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("ApplyCrds", mock.Anything, mock.Anything).Return(errors.New("Mock failed to apply"))

	values, err := PackageService{}.InstallPackage(models.ChartEntry{Name: "bbe-networking"}, nil, models.BbeConfig{}, mockHelmService, &mocks.MockUiService{})

	assert.Nil(t, values)
	assert.EqualError(t, err, "Mock failed to apply")
	mockHelmService.AssertNotCalled(t, "InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// A chart suggesting two indexers to start with
func chartSuggesting() *models.HelmChart {
	return &models.HelmChart{
		Name: "bbe-media",
		Settings: []models.ChartSetting{{
			Name:   "indexers",
			Values: "bbe.media.prowlarr.indexers",
			Suggested: []map[string]interface{}{
				{"name": "1337x", "definition": "1337x"},
				{"name": "EZTV", "definition": "eztv"},
			},
		}},
	}
}

const suggestionTitle = "Package `bbe-media` suggests 2 indexers to start with: 1337x, EZTV. Add them? You can change them later with `bbe configure bbe-media`"

func Test_UpgradePackage_Succeeds_AddingTheSuggestedEntriesTheUserAccepts(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartSuggesting(), nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockUiService := &mocks.MockUiService{}
	mockUiService.On("CreateSelect", suggestionTitle, []string{"Yes", "No"}).Return("Yes", nil)

	values, err := PackageService{}.UpgradePackage(models.ChartEntry{Name: "bbe-media"}, map[string]interface{}{"bbe": map[string]interface{}{"admin": "admin"}}, models.BbeConfig{}, mockHelmService, mockUiService, true)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"bbe": map[string]interface{}{
		"admin": "admin",
		"media": map[string]interface{}{"prowlarr": map[string]interface{}{"indexers": []interface{}{
			map[string]interface{}{"name": "1337x", "definition": "1337x"},
			map[string]interface{}{"name": "EZTV", "definition": "eztv"},
		}}},
	}}, values)
}

func Test_InstallPackage_Succeeds_RecordingThatTheUserDeclinedTheSuggestedEntries(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartSuggesting(), nil)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockUiService := &mocks.MockUiService{}
	mockUiService.On("CreateSelect", suggestionTitle, []string{"Yes", "No"}).Return("No", nil)

	values, err := PackageService{}.InstallPackage(models.ChartEntry{Name: "bbe-media"}, nil, models.BbeConfig{}, mockHelmService, mockUiService)

	assert.NoError(t, err)
	assert.Equal(t, map[string]interface{}{"bbe": map[string]interface{}{"media": map[string]interface{}{"prowlarr": map[string]interface{}{"indexers": []interface{}{}}}}}, values)
}

func Test_UpgradePackage_Succeeds_WithoutAskingAboutSuggestedEntries(t *testing.T) {
	cases := map[string]struct {
		values      map[string]interface{}
		interactive bool
	}{
		"already answered": {map[string]interface{}{"bbe": map[string]interface{}{"media": map[string]interface{}{"prowlarr": map[string]interface{}{"indexers": []interface{}{}}}}}, true},
		"not interactive":  {nil, false},
	}

	for name, c := range cases {
		mockHelmService := &mocks.MockHelmService{}
		mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
		mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartSuggesting(), nil)
		mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
		mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockUiService := &mocks.MockUiService{}

		values, err := PackageService{}.UpgradePackage(models.ChartEntry{Name: "bbe-media"}, c.values, models.BbeConfig{}, mockHelmService, mockUiService, c.interactive)

		assert.NoError(t, err, name)
		assert.Equal(t, normalizeValues(c.values), values, name)
		mockUiService.AssertNotCalled(t, "CreateSelect", mock.Anything, mock.Anything)
	}
}

func Test_UpgradePackage_Fails_WhenTheSuggestionPromptFails(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(chartSuggesting(), nil)
	mockUiService := &mocks.MockUiService{}
	mockUiService.On("CreateSelect", mock.Anything, mock.Anything).Return("", errors.New("Mock prompt failed"))

	values, err := PackageService{}.UpgradePackage(models.ChartEntry{Name: "bbe-media"}, nil, models.BbeConfig{}, mockHelmService, mockUiService, true)

	assert.Nil(t, values)
	assert.EqualError(t, err, "Mock prompt failed")
	mockHelmService.AssertNotCalled(t, "UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}
