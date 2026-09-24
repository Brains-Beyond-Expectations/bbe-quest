package helm_service

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
)

var originalYamlMarshal = yamlMarshal

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
	err := helmService.InstallChart("packageName", "chartName", "repoName", "version", "namespace", "context", nil)

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
	err := helmService.InstallChart("packageName", "chartName", "repoName", "version", "namespace", "context", nil)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_Helm_Service_Fails_Upgrade_Chart(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	helmService := HelmService{}
	err := helmService.UpgradeChart("packageName", "chartName", "repoName", "version", "namespace", "context", nil)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Failed to upgrade helm package `packageName`: exit status 1")
}

func Test_Helm_Service_Succeeds_Upgrade_Chart(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	helmService := HelmService{}
	err := helmService.UpgradeChart("packageName", "chartName", "repoName", "version", "namespace", "context", nil)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_Helm_Service_Fails_UnInstall(t *testing.T) {
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

func Test_Helm_Service_Succeeds_UnInstall(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	helmService := HelmService{}
	err := helmService.UninstallChart("packageName", "namespace", "context")

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_Helm_Service_Fails_IsPackageInstalled(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	helmService := HelmService{}
	res := helmService.IsPackageInstalled("packageName", "namespace", "context")

	// Assert an error occurred
	assert.False(t, res)
}

func Test_Helm_Service_Succeeds_IsPackageInstalled(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	helmService := HelmService{}
	res := helmService.IsPackageInstalled("packageName", "namespace", "context")

	// Assert an error occurred
	assert.True(t, res)
}

func Test_Helm_Service_Succeeds_Install_Chart_With_Values_On_Stdin(t *testing.T) {
	stdinFile := filepath.Join(t.TempDir(), "stdin.yaml")
	var receivedArgs []string
	execCommand = func(_ string, args ...string) *exec.Cmd {
		receivedArgs = args
		return exec.Command("sh", "-c", `cat > "$0"`, stdinFile)
	}

	helmService := HelmService{}
	err := helmService.InstallChart("packageName", "chartName", "repoName", "version", "namespace", "context", map[string]interface{}{
		"service": map[string]interface{}{"loadBalancerIP": "192.168.1.240"},
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"--values", "-"}, receivedArgs[len(receivedArgs)-2:])
	content, _ := os.ReadFile(stdinFile)
	assert.Equal(t, "service:\n    loadBalancerIP: 192.168.1.240\n", string(content))
}

func Test_Helm_Service_Succeeds_Upgrade_Chart_With_Values_On_Stdin(t *testing.T) {
	stdinFile := filepath.Join(t.TempDir(), "stdin.yaml")
	var receivedArgs []string
	execCommand = func(_ string, args ...string) *exec.Cmd {
		receivedArgs = args
		return exec.Command("sh", "-c", `cat > "$0"`, stdinFile)
	}

	helmService := HelmService{}
	err := helmService.UpgradeChart("packageName", "chartName", "repoName", "version", "namespace", "context", map[string]interface{}{"replicas": 2})

	assert.NoError(t, err)
	assert.Equal(t, []string{"--values", "-"}, receivedArgs[len(receivedArgs)-2:])
	content, _ := os.ReadFile(stdinFile)
	assert.Equal(t, "replicas: 2\n", string(content))
}

func Test_Helm_Service_Install_Chart_Skips_Values_When_Empty(t *testing.T) {
	var receivedArgs []string
	execCommand = func(_ string, args ...string) *exec.Cmd {
		receivedArgs = args
		return exec.Command("true")
	}

	helmService := HelmService{}
	err := helmService.InstallChart("packageName", "chartName", "repoName", "version", "namespace", "context", map[string]interface{}{})

	assert.NoError(t, err)
	assert.NotContains(t, receivedArgs, "--values")
}

func Test_Helm_Service_Fails_Install_Chart_With_Helm_Output(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'Error: values do not meet the specifications of the schema'; exit 1")
	}

	helmService := HelmService{}
	err := helmService.InstallChart("packageName", "chartName", "repoName", "version", "namespace", "context", nil)

	assert.Error(t, err)
	assert.Equal(t, "Failed to install helm package `packageName`: exit status 1\nError: values do not meet the specifications of the schema", err.Error())
}

func Test_Helm_Service_Fails_Upgrade_Chart_With_Helm_Output(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'Error: UPGRADE FAILED'; exit 1")
	}

	helmService := HelmService{}
	err := helmService.UpgradeChart("packageName", "chartName", "repoName", "version", "namespace", "context", nil)

	assert.Error(t, err)
	assert.Equal(t, "Failed to upgrade helm package `packageName`: exit status 1\nError: UPGRADE FAILED", err.Error())
}

func Test_Helm_Service_Fails_Install_And_Upgrade_Chart_When_Values_Cannot_Be_Marshalled(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}
	yamlMarshal = func(_ interface{}) ([]byte, error) {
		return nil, errors.New("Mock marshal failure")
	}
	defer func() { yamlMarshal = originalYamlMarshal }()

	helmService := HelmService{}
	values := map[string]interface{}{"replicas": 2}
	installErr := helmService.InstallChart("packageName", "chartName", "repoName", "version", "namespace", "context", values)
	upgradeErr := helmService.UpgradeChart("packageName", "chartName", "repoName", "version", "namespace", "context", values)

	assert.EqualError(t, installErr, "Failed to install helm package `packageName`: Mock marshal failure")
	assert.EqualError(t, upgradeErr, "Failed to upgrade helm package `packageName`: Mock marshal failure")
}

func Test_Helm_Service_Succeeds_Pull_Chart(t *testing.T) {
	var receivedArgs []string
	execCommand = func(_ string, args ...string) *exec.Cmd {
		receivedArgs = args
		destination := args[slices.Index(args, "--destination")+1]
		writeChartArchive(t, filepath.Join(destination, "chartName-1.0.0.tgz"), map[string]string{
			"chartName/Chart.yaml":  "name: chartName\n",
			"chartName/values.yaml": "replicas: 1\n",
		})
		return exec.Command("true")
	}

	helmService := HelmService{}
	chart, err := helmService.PullChart("https://example.com/charts", "chartName", "1.0.0")

	assert.NoError(t, err)
	assert.Equal(t, "chartName", chart.Name)
	assert.Equal(t, map[string]interface{}{"replicas": 1}, chart.Values)
	destination := receivedArgs[len(receivedArgs)-1]
	assert.Equal(t, []string{"pull", "chartName", "--repo", "https://example.com/charts", "--version", "1.0.0", "--destination", destination}, receivedArgs)
	assert.NoDirExists(t, destination)
}

func Test_Helm_Service_Fails_Pull_Chart_With_Helm_Output(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("sh", "-c", "echo 'Error: chart \"chartName\" version \"9.9.9\" not found'; exit 1")
	}

	helmService := HelmService{}
	chart, err := helmService.PullChart("https://example.com/charts", "chartName", "9.9.9")

	assert.Nil(t, chart)
	assert.EqualError(t, err, "Failed to pull helm chart `chartName`: exit status 1\nError: chart \"chartName\" version \"9.9.9\" not found")
}

func Test_Helm_Service_Fails_Pull_Chart_Without_Archive(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	helmService := HelmService{}
	chart, err := helmService.PullChart("https://example.com/charts", "chartName", "1.0.0")

	assert.Nil(t, chart)
	assert.EqualError(t, err, "Failed to find the archive for helm chart `chartName`")
}

func Test_Helm_Service_Fails_Pull_Chart_With_Invalid_Archive(t *testing.T) {
	execCommand = func(_ string, args ...string) *exec.Cmd {
		destination := args[slices.Index(args, "--destination")+1]
		os.WriteFile(filepath.Join(destination, "chartName-1.0.0.tgz"), []byte("not an archive"), 0o600)
		return exec.Command("true")
	}

	helmService := HelmService{}
	chart, err := helmService.PullChart("https://example.com/charts", "chartName", "1.0.0")

	assert.Nil(t, chart)
	assert.ErrorContains(t, err, "Failed to read helm chart `chartName`")
}

func Test_Helm_Service_Fails_Pull_Chart_Without_Temporary_Directory(t *testing.T) {
	osMkdirTemp = func(_ string, _ string) (string, error) {
		return "", errors.New("Mock mkdir failure")
	}
	defer func() { osMkdirTemp = os.MkdirTemp }()

	helmService := HelmService{}
	chart, err := helmService.PullChart("https://example.com/charts", "chartName", "1.0.0")

	assert.Nil(t, chart)
	assert.EqualError(t, err, "Failed to create a directory for helm chart `chartName`: Mock mkdir failure")
}

func Test_chartNotes_Succeeds_ReadingTheNotesFromHelmsOutput(t *testing.T) {
	response := []byte("NAME: bbe-media\nSTATUS: deployed\nREVISION: 1\nNOTES:\nEvery app uses the same login\n  username: admin\n\n")

	assert.Equal(t, "Every app uses the same login\n  username: admin", chartNotes(response))
	assert.Equal(t, "", chartNotes([]byte("NAME: bbe-media\nSTATUS: deployed\n")))
	assert.Equal(t, "", chartNotes(nil))
}

func Test_Helm_Service_Succeeds_Install_And_Upgrade_Chart_With_Notes(t *testing.T) {
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("printf", "NAME: test\nNOTES:\nRead the password with kubectl\n")
	}

	helmService := HelmService{}
	installErr := helmService.InstallChart("packageName", "chartName", "repoName", "version", "namespace", "context", nil)
	upgradeErr := helmService.UpgradeChart("packageName", "chartName", "repoName", "version", "namespace", "context", nil)

	assert.NoError(t, installErr)
	assert.NoError(t, upgradeErr)
}
