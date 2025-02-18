package dependency_service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_VerifyDependencies_Succeeds(t *testing.T) {
	originalCommand := buildCommand

	stringList := []string{}
	buildCommand = buildBuildCommand(stringList)

	dependencyService := DependencyService{}
	result := dependencyService.VerifyDependencies()

	assert.True(t, result)

	buildCommand = originalCommand
}

func Test_VerifyDependencies_Fails_WithNoDependencies(t *testing.T) {
	originalCommand := buildCommand

	stringList := []string{"talosctl", "nmap", "grep", "bash"}
	buildCommand = buildBuildCommand(stringList)

	dependencyService := DependencyService{}
	result := dependencyService.VerifyDependencies()

	assert.False(t, result)

	buildCommand = originalCommand
}

func Test_BuildCommand(t *testing.T) {
	dependency := "test"
	args := []string{"--version"}

	result := buildCommand(dependency, args)

	assert.Equal(t, "test --version", result)
}

func buildBuildCommand(dependenciesToFail []string) func(dependency string, args []string) string {
	return func(dependency string, args []string) string {
		for _, dep := range dependenciesToFail {
			if dependency == dep {
				return "false"
			}
		}
		return "true"
	}
}
