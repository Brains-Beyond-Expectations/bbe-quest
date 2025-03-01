package helm_service

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Helm_Service_Fails_Add_Repo(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	helmService := HelmService{}
	err := helmService.AddRepo("repoName", "repoUrl")

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to add helm repository `repoName`: exit status 1")
}
func Test_Helm_Service_Succeeds_Add_Repo(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	helmService := HelmService{}
	err := helmService.AddRepo("repoName", "repoUrl")

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_Helm_Service_Fails_Install_Repo(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	helmService := HelmService{}
	err := helmService.InstallChart("packageName", "chartName", "repoName", "version", "namespace", "context")

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to install helm package `packageName`: exit status 1")
}

func Test_Helm_Service_Succeeds_Install_Repo(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	helmService := HelmService{}
	err := helmService.InstallChart("packageName", "chartName", "repoName", "version", "namespace", "context")

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_Helm_Service_Fails_UpgradeChart_Repo(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	helmService := HelmService{}
	err := helmService.UpgradeChart("packageName", "chartName", "repoName", "version", "namespace", "context")

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to upgrade helm package `packageName`: exit status 1")
}

func Test_Helm_Service_Succeeds_UpgradeChart_Repo(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	helmService := HelmService{}
	err := helmService.UpgradeChart("packageName", "chartName", "repoName", "version", "namespace", "context")

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_Helm_Service_Fails_UnInstall_Repo(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	helmService := HelmService{}
	err := helmService.UninstallChart("packageName", "namespace", "context")

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to uninstall helm package `packageName`: exit status 1")
}

func Test_Helm_Service_Succeeds_UnInstall_Repo(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	helmService := HelmService{}
	err := helmService.UninstallChart("packageName", "namespace", "context")

	// Assert an error occurred
	assert.NoError(t, err)
}
