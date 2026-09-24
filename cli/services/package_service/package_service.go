package package_service

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"golang.org/x/mod/semver"
	"gopkg.in/yaml.v3"
)

type PackageService struct{}

var ioReadAll = io.ReadAll

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
		if isRevisionSupported(revision.MinBbeCli, constants.Version) {
			return &revision, nil
		}
	}

	return nil, fmt.Errorf("No revision found for bbe-cli version %s", constants.Version)
}

// Development builds support every revision, as do revisions without a minimum version
func isRevisionSupported(minBbeCli string, cliVersion string) bool {
	if cliVersion == "development" || minBbeCli == "" {
		return true
	}

	minVersion := toSemver(minBbeCli)
	currentVersion := toSemver(cliVersion)
	if !semver.IsValid(minVersion) || !semver.IsValid(currentVersion) {
		logger.Debug(fmt.Sprintf("Skipping revision, cannot compare min-bbe-cli `%s` with bbe-cli version `%s`", minBbeCli, cliVersion))
		return false
	}

	return semver.Compare(minVersion, currentVersion) <= 0
}

// semver expects a leading `v`, which library.yaml versions don't have
func toSemver(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}

	return "v" + version
}

func (packageService PackageService) GetAll() ([]models.ChartEntry, error) {
	library, err := getRemoteLibrary()
	if err != nil {
		logger.Debug(fmt.Sprintf("Error fetching library: %v", err))
		return nil, err
	}

	return library.Charts, nil
}

// Returns the values the package was installed with, including any the user was asked for
func (packageService PackageService) InstallPackage(chart models.ChartEntry, values map[string]interface{}, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface, uiService interfaces.UiServiceInterface) (map[string]interface{}, error) {
	if helmService.IsPackageInstalled(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context) {
		logger.Debug(fmt.Sprintf("Package `%s` already installed", chart.Name))
		return values, nil
	}

	values, err := resolveValues(chart, values, helmService, uiService, true)
	if err != nil {
		return nil, err
	}

	logger.Debug(fmt.Sprintf("Package `%s` not installed, adding helm repo...", chart.Name))
	response := helmService.AddRepo(chart.RepositoryName, chart.RepositoryUrl)
	logger.Debug(fmt.Sprintf("Helm repo added: %v", response))

	if response != nil {
		return nil, response
	}

	err = helmService.InstallChart(chart.Name, chart.Name, chart.RepositoryName, chart.Version, chart.Name, bbeConfig.Bbe.Cluster.Context, values)
	if err != nil {
		return nil, err
	}

	return values, nil
}

// Returns the values the package was upgraded with, asking the user for any the new version requires when interactive
func (packageService PackageService) UpgradePackage(chart models.ChartEntry, values map[string]interface{}, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface, uiService interfaces.UiServiceInterface, interactive bool) (map[string]interface{}, error) {
	if !helmService.IsPackageInstalled(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context) {
		logger.Debug(fmt.Sprintf("Package `%s` not installed", chart.Name))
		return nil, fmt.Errorf("Package `%s` not installed", chart.Name)
	}

	values, err := resolveValues(chart, values, helmService, uiService, interactive)
	if err != nil {
		return nil, err
	}

	response := helmService.AddRepo(chart.RepositoryName, chart.RepositoryUrl)

	if response != nil {
		return nil, response
	}

	err = helmService.UpgradeChart(chart.Name, chart.Name, chart.RepositoryName, chart.Version, chart.Name, bbeConfig.Bbe.Cluster.Context, values)
	if err != nil {
		return nil, err
	}

	return values, nil
}

func (packageService PackageService) UninstallPackage(chart models.LocalPackage, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	if !helmService.IsPackageInstalled(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context) {
		logger.Debug(fmt.Sprintf("Package `%s` not installed", chart.Name))
		return fmt.Errorf("Package `%s` not installed", chart.Name)
	}

	return helmService.UninstallChart(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context)
}
