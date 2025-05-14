package package_service

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
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

func (packageService PackageService) InstallPackage(chart models.ChartEntry, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	if !helmService.IsPackageInstalled(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context) {
		logger.Debug(fmt.Sprintf("Package `%s` not installed, adding helm repo...", chart.Name))
		response := helmService.AddRepo(chart.RepositoryName, chart.RepositoryUrl)
		logger.Debug(fmt.Sprintf("Helm repo added: %v", response))

		if response != nil {
			return response
		}

		return helmService.InstallChart(chart.Name, chart.Name, chart.RepositoryName, chart.Version, chart.Name, bbeConfig.Bbe.Cluster.Context)
	}
	logger.Debug(fmt.Sprintf("Package `%s` already installed", chart.Name))

	return nil
}

func (packageService PackageService) UpgradePackage(chart models.ChartEntry, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	if !helmService.IsPackageInstalled(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context) {
		logger.Debug(fmt.Sprintf("Package `%s` not installed", chart.Name))
		return fmt.Errorf("Package `%s` not installed", chart.Name)
	}

	response := helmService.AddRepo(chart.RepositoryName, chart.RepositoryUrl)

	if response != nil {
		return response
	}

	return helmService.UpgradeChart(chart.Name, chart.Name, chart.RepositoryName, chart.Version, chart.Name, bbeConfig.Bbe.Cluster.Context)
}

func (packageService PackageService) UninstallPackage(chart models.LocalPackage, bbeConfig models.BbeConfig, helmService interfaces.HelmServiceInterface) error {
	if !helmService.IsPackageInstalled(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context) {
		logger.Debug(fmt.Sprintf("Package `%s` not installed", chart.Name))
		return fmt.Errorf("Package `%s` not installed", chart.Name)
	}

	return helmService.UninstallChart(chart.Name, chart.Name, bbeConfig.Bbe.Cluster.Context)
}
