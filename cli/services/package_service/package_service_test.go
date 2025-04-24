package package_service

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_GetAll_Succeeds(t *testing.T) {
	// Create a test server that returns our mock library.yaml
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return our mock library.yaml content
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

func Test_InstallPackage_Fails_WhenHelmRepositoryNotFound(t *testing.T) {
	mockErrorMessage := "Mock failed to add"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_InstallPackage_Fails_WhenHelmInstallFails(t *testing.T) {
	mockErrorMessage := "Mock failed to install"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_InstallPackage_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("InstallChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_InstallPackage_Skips_Already_Installed_And_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"

	err := packagesService.InstallPackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_UpgradePackage_Fails_WhenHelmRepositoryNotFound(t *testing.T) {
	mockErrorMessage := "Mockfailed adding repo"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_UpgradePackage_Fails_WhenHelmUpgradeFails(t *testing.T) {
	mockErrorMessage := "Mockfailed upgrading repo"
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New(mockErrorMessage))

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), mockErrorMessage)
}

func Test_UpgradePackage_Succeeds(t *testing.T) {
	mockHelmService := &mocks.MockHelmService{}
	mockHelmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	mockHelmService.On("AddRepo", mock.Anything, mock.Anything).Return(nil)
	mockHelmService.On("UpgradeChart", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradePackage(models.ChartEntry{Name: "ingress-nginx", Version: "4.12.0"}, bbeConfig, mockHelmService)

	// Assert an error occurred
	assert.NoError(t, err)
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
