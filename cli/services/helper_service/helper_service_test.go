package helper_service

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CheckIfFileExists_Succeeds_IfFileExists(t *testing.T) {
	helperService := HelperService{}

	tmpfile, err := os.CreateTemp("", "example")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	modTime, exists := helperService.CheckIfFileExists(tmpfile.Name())

	assert.True(t, exists)
	assert.NotNil(t, modTime)
}

func Test_CheckIfFileExists_Succeeds_IfFileDoesNotExist(t *testing.T) {
	helperService := HelperService{}

	modTime, exists := helperService.CheckIfFileExists("nonexistentfile")

	assert.False(t, exists)
	assert.Nil(t, modTime)
}

func Test_PipeCommands_Succeeds_WithMultipleCommands(t *testing.T) {
	helperService := HelperService{}

	cmd1 := exec.Command("echo", "hello\nworld")
	cmd2 := exec.Command("grep", "hello")

	output, err := helperService.PipeCommands(cmd1, cmd2)
	assert.Nil(t, err)
	assert.Equal(t, "hello\n", string(output))
}

func Test_PipeCommands_Succeeds_WithOneCommand(t *testing.T) {
	helperService := HelperService{}

	cmd1 := exec.Command("echo", "hello\nworld")

	output, err := helperService.PipeCommands(cmd1)
	assert.Nil(t, err)
	assert.Equal(t, "hello\nworld\n", string(output))
}

func Test_PipeCommands_Fails_WithNoCommandsProvided(t *testing.T) {
	helperService := HelperService{}

	output, err := helperService.PipeCommands()
	assert.NotNil(t, err)
	assert.Nil(t, output)
}

func Test_DeleteEmptyStrings_Succeeds(t *testing.T) {
	helperService := HelperService{}

	input := []string{"a", "", "b", "", "c"}
	expected := []string{"a", "b", "c"}
	result := helperService.DeleteEmptyStrings(input)

	assert.Equal(t, expected, result)

	for i, v := range result {
		assert.Equal(t, expected[i], v)
	}
}

func Test_IsWsl_Succeeds_OnLinux(t *testing.T) {
	helperService := HelperService{}

	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", "Linux")
	}

	isWsl := helperService.IsWsl()

	assert.False(t, isWsl)
}

func Test_IsWsl_Succeeds_OnWSL(t *testing.T) {
	helperService := HelperService{}

	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("echo", "microsoft WSL")
	}

	isWsl := helperService.IsWsl()

	assert.True(t, isWsl)
}

func Test_IsWsl_Succeeds_WithUnknownError(t *testing.T) {
	helperService := HelperService{}

	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	isWsl := helperService.IsWsl()

	assert.False(t, isWsl)
}

func TestIsValidIp(t *testing.T) {
	helperService := HelperService{}

	validIps := []string{"192.168.1.1", "10.0.0.1", "255.255.255.255"}
	invalidIps := []string{"256.256.256.256", "abc.def.ghi.jkl", "1234"}

	for _, ip := range validIps {
		if !helperService.IsValidIp(ip) {
			t.Errorf("Expected valid IP for %s", ip)
		}
	}

	for _, ip := range invalidIps {
		if helperService.IsValidIp(ip) {
			t.Errorf("Expected invalid IP for %s", ip)
		}
	}
}

func TestGetConfigDir(t *testing.T) {
	helperService := HelperService{}

	configDir := helperService.GetConfigDir()
	if configDir == "" {
		t.Errorf("Expected non-empty configDir")
	}
}

func TestGetConfigFilePath(t *testing.T) {
	helperService := HelperService{}

	configFilePath := helperService.GetConfigFilePath("config.json")
	if !strings.HasSuffix(configFilePath, ".bbe/config.json") {
		t.Errorf("Expected config file path to end with .bbe/config.json, got %s", configFilePath)
	}
}
