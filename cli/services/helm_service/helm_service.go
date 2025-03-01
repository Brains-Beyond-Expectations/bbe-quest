package helm_service

import (
	"fmt"
	"os/exec"
)

var execCommand = exec.Command

type HelmService struct{}

func (HelmService HelmService) AddRepo(repoName string, repoUrl string) error {
	cmd := execCommand("helm", "repo", "add", repoName, repoUrl)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to add helm repository `%s`: %w", repoName, err)
	}
	return nil
}

func (HelmService HelmService) InstallChart(pkgName string, chartName string, repoName string, version string, namespace string, context string) error {
	cmd := execCommand("helm", "install", pkgName, fmt.Sprintf("%s/%s", repoName, chartName),
		"--version", version,
		"--namespace", namespace,
		"--create-namespace",
		"--kube-context", context)

	if err := cmd.Run(); err != nil {
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

func (HelmService HelmService) Status(pkgName string, namespace string, context string) (bool, error) {
	cmd := execCommand("helm", "status", pkgName,
		"--namespace", namespace,
		"--kube-context", context)

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("Failed to get helm status for `%s`: %w", pkgName, err)
	}
	return true, nil
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
