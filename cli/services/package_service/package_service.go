package package_service

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
var osReadFile = os.ReadFile

func getRemoteLibrary() (*models.LibraryEntry, error) {
	source := constants.BbeLibraryUrl
	if source != constants.DefaultBbeLibraryUrl {
		logger.Warning(fmt.Sprintf("Using the package library from %s instead of the released one", source))
	}

	body, err := readLibrary(source)
	if err != nil {
		logger.Debug(fmt.Sprintf("Error reading library.yaml: %v", err))
		return nil, err
	}

	var library models.Library
	if err := yaml.Unmarshal(body, &library); err != nil {
		logger.Debug(fmt.Sprintf("Error parsing YAML: %v", err))
		return nil, err
	}

	for _, revision := range library.Library {
		if isRevisionSupported(revision.MinBbeCli, constants.Version) {
			resolveLocalCharts(&revision, source)
			return &revision, nil
		}
	}

	return nil, fmt.Errorf("No revision found for bbe-cli version %s", constants.Version)
}

// The library is a URL, or a file on disk when testing charts that aren't released yet
func readLibrary(source string) ([]byte, error) {
	if !strings.HasPrefix(source, "http://") && !strings.HasPrefix(source, "https://") {
		return osReadFile(strings.TrimPrefix(source, "file://"))
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(source)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Failed to fetch the package library from %s: %s", source, resp.Status)
	}

	return ioReadAll(resp.Body)
}

// Chart folders in a library on disk are relative to the library, so it can sit next to the charts
func resolveLocalCharts(revision *models.LibraryEntry, source string) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return
	}

	for i, chart := range revision.Charts {
		path, local := chart.LocalPath()
		if local && !filepath.IsAbs(path) {
			revision.Charts[i].RepositoryUrl = "file://" + filepath.Join(filepath.Dir(strings.TrimPrefix(source, "file://")), path)
		}
	}
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

	helmChart, err := helmService.PullChart(chart.RepositoryUrl, chart.Name, chart.Version)
	if err != nil {
		return nil, err
	}

	values, err = resolveValues(chart, helmChart, values, uiService, true)
	if err != nil {
		return nil, err
	}

	values, err = offerSuggestedEntries(chart, helmChart, values, uiService, true)
	if err != nil {
		return nil, err
	}

	chartName, repoName, version, err := chartSource(chart, helmService)
	if err != nil {
		return nil, err
	}

	if err := prepareNamespace(chart, helmChart, bbeConfig, helmService); err != nil {
		return nil, err
	}

	if err := applyCrds(helmChart, values, bbeConfig, helmService); err != nil {
		return nil, err
	}

	err = helmService.InstallChart(chart.Name, chartName, repoName, version, chart.Name, bbeConfig.Bbe.Cluster.Context, values)
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

	helmChart, err := helmService.PullChart(chart.RepositoryUrl, chart.Name, chart.Version)
	if err != nil {
		return nil, err
	}

	values, err = resolveValues(chart, helmChart, values, uiService, interactive)
	if err != nil {
		return nil, err
	}

	values, err = offerSuggestedEntries(chart, helmChart, values, uiService, interactive)
	if err != nil {
		return nil, err
	}

	chartName, repoName, version, err := chartSource(chart, helmService)
	if err != nil {
		return nil, err
	}

	// Also for packages installed before their chart asked for a pod security level
	if err := prepareNamespace(chart, helmChart, bbeConfig, helmService); err != nil {
		return nil, err
	}

	if err := applyCrds(helmChart, values, bbeConfig, helmService); err != nil {
		return nil, err
	}

	err = helmService.UpgradeChart(chart.Name, chartName, repoName, version, chart.Name, bbeConfig.Bbe.Cluster.Context, values)
	if err != nil {
		return nil, err
	}

	return values, nil
}

// Returns what helm installs the package from: a chart in a repository, which is added first, or a chart folder,
// which has no repository or version
func chartSource(chart models.ChartEntry, helmService interfaces.HelmServiceInterface) (chartName string, repoName string, version string, err error) {
	if path, local := chart.LocalPath(); local {
		logger.Debug(fmt.Sprintf("Package `%s` is installed from the folder %s", chart.Name, path))
		return path, "", "", nil
	}

	logger.Debug(fmt.Sprintf("Adding helm repo `%s` for package `%s`", chart.RepositoryName, chart.Name))
	if err := helmService.AddRepo(chart.RepositoryName, chart.RepositoryUrl); err != nil {
		return "", "", "", err
	}

	return chart.Name, chart.RepositoryName, chart.Version, nil
}

// Charts whose pods need more than the cluster allows by default say so in their Chart.yaml
func prepareNamespace(chart models.ChartEntry, helmChart *models.HelmChart, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	if helmChart.PodSecurity == "" {
		return nil
	}

	return helmService.PrepareNamespace(chart.Name, bbeConfig.Bbe.Cluster.Context, helmChart.PodSecurity)
}

// Offers the entries a chart suggests for a setting whose list the values don't set yet, such as bbe-media's
// indexers. The answer is kept either way, a "no" as an empty list, so it's only asked once. Without asking, nothing is
// added, as the user hasn't chosen
func offerSuggestedEntries(chart models.ChartEntry, helmChart *models.HelmChart, values map[string]interface{}, uiService interfaces.UiServiceInterface, interactive bool) (map[string]interface{}, error) {
	if !interactive {
		return values, nil
	}

	for _, setting := range helmChart.Settings {
		path := strings.Split(setting.Values, ".")
		if len(setting.Suggested) == 0 || lookupValue(values, path) != nil {
			continue
		}

		labels := []string{}
		for _, entry := range setting.Suggested {
			labels = append(labels, setting.EntryLabel(entry))
		}
		answer, err := uiService.CreateSelect(fmt.Sprintf("Package `%s` suggests %d %s to start with: %s. Add them? You can change them later with `bbe configure %s`", chart.Name, len(labels), setting.Name, strings.Join(labels, ", "), chart.Name), []string{"Yes", "No"})
		if err != nil {
			return nil, err
		}

		entries := []interface{}{}
		if answer == "Yes" {
			for _, entry := range setting.Suggested {
				entries = append(entries, entry)
			}
		}
		if values == nil {
			values = map[string]interface{}{}
		}
		setValue(values, path, entries)
	}

	return values, nil
}

// Helm only installs CRDs from a chart's crds/ folder on its first install, so bbe applies them before every install
// and upgrade
func applyCrds(helmChart *models.HelmChart, values map[string]interface{}, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	crds := chartCrds(helmChart, coalesceValues(helmChart, values))
	if len(crds) == 0 {
		return nil
	}

	return helmService.ApplyCrds(bbeConfig.Bbe.Cluster.Context, crds)
}

// Returns what the chart needs in the cluster before it can be installed, from its Chart.yaml
func (packageService PackageService) GetPrerequisites(chart models.ChartEntry, helmService interfaces.HelmServiceInterface) ([]models.ChartPrerequisite, error) {
	helmChart, err := helmService.PullChart(chart.RepositoryUrl, chart.Name, chart.Version)
	if err != nil {
		return nil, err
	}

	return helmChart.Prerequisites, nil
}
