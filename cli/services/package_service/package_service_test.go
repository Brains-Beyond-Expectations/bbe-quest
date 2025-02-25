package package_service

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
)

func Test_GetAll_Succeeds(t *testing.T) {
	packagesService := PackageService{}

	// Get all packages directly from the service
	result := packagesService.GetAll()

	// Assert that the result contains the correct package data
	assert.Len(t, result, 1) // We have one package in the predefined array
	assert.Equal(t, "blocky", result[0].Name)
	assert.Equal(t, "0.1.3", result[0].Version)
}

func Test_InstallPackage_Fails_WhenPackageNotFound(t *testing.T) {
	packageName := "not-a-real-package"

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallPackage(models.Package{Name: "not-a-real-package", Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("package %s not found", packageName))
}

func Test_InstallPackage_Fails_WhenHelmRepositoryNotFound(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallPackage(models.Package{Name: "blocky", Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add helm repository bbe: exit status 1")
}

func Test_InstallPackage_Fails_WhenHelmInstallFails(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, args ...string) *exec.Cmd {
		if args[0] == "repo" {
			return exec.Command("true")
		} else {
			return exec.Command("false")
		}
	}

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallPackage(models.Package{Name: "blocky", Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to install helm package blocky: exit status 1")
}

func Test_InstallPackage_Succeeds(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.InstallPackage(models.Package{Name: "blocky", Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_UpgradePackage_Fails_WhenPackageNotFound(t *testing.T) {
	packageName := "not-a-real-package"

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradePackage(models.Package{Name: "not-a-real-package", Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("package %s not found", packageName))
}

func Test_UpgradePackage_Fails_WhenHelmRepositoryNotFound(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("false")
	}

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradePackage(models.Package{Name: "blocky", Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to add helm repository bbe: exit status 1")
}

func Test_UpgradePackage_Fails_WhenHelmInstallFails(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, args ...string) *exec.Cmd {
		if args[0] == "repo" {
			return exec.Command("true")
		} else {
			return exec.Command("false")
		}
	}

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradePackage(models.Package{Name: "blocky", Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to upgrade helm package blocky: exit status 1")
}

func Test_UpgradePackage_Succeeds(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UpgradePackage(models.Package{Name: "blocky", Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.NoError(t, err)
}

func Test_UninstallPackage_Fails_WhenPackageNotFound(t *testing.T) {
	packageName := "not-a-real-package"
	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UninstallPackage(models.Package{Name: packageName, Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf("package %s not found", packageName))
}

func Test_UninstallPackage_Fails_WhenHelmFails(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UninstallPackage(models.Package{Name: "blocky", Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to uninstall helm package blocky: exit status 1")
}

func Test_UninstallPackage_Succeeds(t *testing.T) {
	// Set the mock execCommand to return a mocked Command
	execCommand = func(_ string, _ ...string) *exec.Cmd {
		return exec.Command("true")
	}

	packagesService := PackageService{}

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "test-context"
	err := packagesService.UninstallPackage(models.Package{Name: "blocky", Version: "0.1.3"}, bbeConfig)

	// Assert an error occurred
	assert.NoError(t, err)
}
