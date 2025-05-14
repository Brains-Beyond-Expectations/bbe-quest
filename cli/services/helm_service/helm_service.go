package helm_service

import (
	"fmt"
	"os/exec"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
)

var execCommand = exec.Command

type HelmService struct{}

func (HelmService HelmService) AddRepo(repoName string, repoUrl string) error {
	cmd := execCommand("helm", "repo", "add", repoName, repoUrl)
	logger.Debug(fmt.Sprintf("Adding helm repository `%s` with url `%s`", repoName, repoUrl))
	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))

	if err != nil {
		return fmt.Errorf("Failed to add helm repository `%s`: %w", repoName, err)
	}

	updateRepoErr := HelmService.updateRepo(repoName)
	if updateRepoErr != nil {
		return fmt.Errorf("Failed to update helm repository `%s`: %w", repoName, updateRepoErr)
	}

	return nil
}

func (HelmService HelmService) InstallChart(pkgName string, chartName string, repoName string, version string, namespace string, context string) error {
	cmd := execCommand("helm", "install", pkgName, fmt.Sprintf("%s/%s", repoName, chartName),
		"--version", version,
		"--namespace", namespace,
		"--create-namespace",
		"--kube-context", context)
	logger.Debug(fmt.Sprintf("Installing helm chart `%s` from repo `%s` with version `%s` in namespace `%s`", pkgName, repoName, version, namespace))
	logger.Debug(fmt.Sprintf("Command: %s", cmd.String()))

	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))

	if err != nil {
		return fmt.Errorf("Failed to install helm package `%s`: %w", pkgName, err)
	}
	return nil
}

func (HelmService HelmService) UpgradeChart(pkgName string, chartName string, repoName string, version string, namespace string, context string) error {
	cmd := execCommand("helm", "upgrade", pkgName, fmt.Sprintf("%s/%s", repoName, chartName),
		"--version", version,
		"--namespace", namespace,
		"--create-namespace",
		"--kube-context", context)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to upgrade helm package `%s`: %w", pkgName, err)
	}
	return nil
}

func (HelmService HelmService) UninstallChart(pkgName string, namespace string, context string) error {
	cmd := execCommand("helm", "uninstall", pkgName,
		"--namespace", namespace,
		"--kube-context", context)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to uninstall helm package `%s`: %w", pkgName, err)
	}
	return nil
}

func (HelmService HelmService) IsPackageInstalled(pkgName string, namespace string, context string) bool {
	cmd := execCommand("helm", "status", pkgName,
		"--namespace", namespace,
		"--kube-context", context,
	)
	if err := cmd.Run(); err != nil {
		return false
	}

	return true
}

func (HelmService HelmService) updateRepo(repoName string) error {
	cmd := execCommand("helm", "repo", "update", repoName)
	logger.Debug(fmt.Sprintf("Updating helm repository `%s`", repoName))
	response, err := cmd.CombinedOutput()
	logger.Debug(fmt.Sprintf("Response: %s", string(response)))

	if err != nil {
		return fmt.Errorf("Failed to update helm repository `%s`: %w", repoName, err)
	}

	return nil
}
