package cmd

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_installCommand_Succeeds(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := initInstallCommand()
	kubernetesService := mockUninstall(helmService, uiService)

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService, kubernetesService)

	assert.Nil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 2)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	helmService.AssertCalled(t, "UninstallChart", "package_to_be_removed", "package_to_be_removed", "")
	packageService.AssertNumberOfCalls(t, "InstallPackage", 2)
	packageService.AssertCalled(t, "InstallPackage", models.ChartEntry{
		Name:    "package_to_be_installed",
		Version: "3.0.0",
	}, mock.Anything)
}

func Test_installCommand_Fails_WithNoCluster(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := initInstallCommand()
	kubernetesService := mockUninstall(helmService, uiService)

	configService.On("GetBbeConfig", mock.Anything).Return(&models.BbeConfig{}, errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService, kubernetesService)

	assert.Nil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 0)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 0)
	packageService.AssertNumberOfCalls(t, "GetAll", 0)
	helmService.AssertNotCalled(t, "UninstallChart", mock.Anything, mock.Anything, mock.Anything)
	packageService.AssertNumberOfCalls(t, "InstallPackage", 0)
}

func Test_installCommand_Succeeds_ProceedsWhenFailingToUninstallPackages(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := initInstallCommand()
	kubernetesService := mockUninstall(helmService, uiService)
	helmService.ExpectedCalls = slices.DeleteFunc(helmService.ExpectedCalls, func(call *mock.Call) bool { return call.Method == "UninstallChart" })
	helmService.On("UninstallChart", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService, kubernetesService)

	assert.Nil(t, err)
	helmService.AssertCalled(t, "UninstallChart", "package_to_be_removed", "package_to_be_removed", "")
	kubernetesService.AssertNotCalled(t, "DeleteNamespace", mock.Anything, mock.Anything)
	packageService.AssertNumberOfCalls(t, "InstallPackage", 2)
	// The package that failed to uninstall is still recorded
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 1)
	configService.AssertCalled(t, "UpdateBbePackages", mock.Anything, mock.MatchedBy(func(packages []models.LocalPackage) bool {
		return slices.ContainsFunc(packages, func(pkg models.LocalPackage) bool { return pkg.Name == "package_to_be_removed" })
	}))
}

func Test_installCommand_Fails_WhenFailingToUpdateBbeConfiguration(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := initInstallCommand()
	kubernetesService := mockUninstall(helmService, uiService)

	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService, kubernetesService)

	assert.EqualError(t, err, "Failed to install packages: Failed to update BBE configuration: test error")
	helmService.AssertCalled(t, "UninstallChart", "package_to_be_removed", "package_to_be_removed", "")
	packageService.AssertNumberOfCalls(t, "InstallPackage", 2)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 2)
}

func Test_installCommand_Fails_WhenFailingToInstallPackage(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := initInstallCommand()
	kubernetesService := mockUninstall(helmService, uiService)

	packageService.On("InstallPackage", mock.Anything, mock.Anything).Return(nil, errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService, kubernetesService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 2)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	helmService.AssertCalled(t, "UninstallChart", "package_to_be_removed", "package_to_be_removed", "")
	packageService.AssertNumberOfCalls(t, "InstallPackage", 1)
}

func Test_installCommand_Fails_WhenFailingToUpdateBbeConfigurationOnInstall(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := initInstallCommand()
	kubernetesService := mockUninstall(helmService, uiService)

	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(nil).Once()
	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(errors.New("test error"))

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService, kubernetesService)

	assert.NotNil(t, err)
	uiService.AssertNumberOfCalls(t, "CreateMultiChoose", 1)
	configService.AssertNumberOfCalls(t, "GetBbeConfig", 1)
	configService.AssertNumberOfCalls(t, "UpdateBbePackages", 2)
	packageService.AssertNumberOfCalls(t, "GetAll", 1)
	helmService.AssertCalled(t, "UninstallChart", "package_to_be_removed", "package_to_be_removed", "")
	packageService.AssertNumberOfCalls(t, "InstallPackage", 2)
}

func Test_installCommand_Succeeds_StoringPackageValues(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := initInstallCommand()
	kubernetesService := mockUninstall(helmService, uiService)

	storedValues := map[string]interface{}{"service": map[string]interface{}{"ip": "192.168.1.240"}}
	enteredValues := map[string]interface{}{"password": "entered"}
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Packages = []models.LocalPackage{
		{
			Name:    "package_always_installed",
			Version: "1.0.0",
			Values:  storedValues,
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)
	packageService.On("InstallPackage", models.ChartEntry{Name: "package_always_installed", Version: "1.0.0"}, storedValues).Return(storedValues, nil)
	packageService.On("InstallPackage", models.ChartEntry{Name: "package_to_be_installed", Version: "3.0.0"}, map[string]interface{}(nil)).Return(enteredValues, nil)

	mockSuccessfulInstallFlow(helperService, uiService, configService, packageService)

	err := installCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService, kubernetesService)

	assert.Nil(t, err)
	packageService.AssertNumberOfCalls(t, "InstallPackage", 2)
	configService.AssertCalled(t, "UpdateBbePackages", mock.Anything, []models.LocalPackage{
		{
			Name:    "package_always_installed",
			Version: "1.0.0",
			Values:  storedValues,
		},
		{
			Name:    "package_to_be_installed",
			Version: "3.0.0",
			Values:  enteredValues,
		},
	})
}

// The packages count as installed already, so their prerequisites aren't checked
func initInstallCommand() (*mocks.MockHelperService, *mocks.MockUiService, *mocks.MockConfigService, *mocks.MockPackageService, *mocks.MockHelmService, *mocks.MockTalosService, *mocks.MockPrerequisiteService) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := newInstallMocks()
	helmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)

	return helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService
}

func newInstallMocks() (*mocks.MockHelperService, *mocks.MockUiService, *mocks.MockConfigService, *mocks.MockPackageService, *mocks.MockHelmService, *mocks.MockTalosService, *mocks.MockPrerequisiteService) {
	return &mocks.MockHelperService{}, &mocks.MockUiService{}, &mocks.MockConfigService{}, &mocks.MockPackageService{}, &mocks.MockHelmService{}, &mocks.MockTalosService{}, &mocks.MockPrerequisiteService{}
}

func mockSuccessfulInstallFlow(_ *mocks.MockHelperService, uiService *mocks.MockUiService, configService *mocks.MockConfigService, packageService *mocks.MockPackageService) {
	uiService.On("CreateMultiChoose", mock.Anything, mock.Anything, mock.Anything).Return([]string{
		"package_always_installed",
		"package_to_be_installed",
	}, nil)

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Packages = []models.LocalPackage{
		{
			Name:    "package_always_installed",
			Version: "1.0.0",
		},
		{
			Name:    "package_to_be_removed",
			Version: "2.0.0",
		},
	}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)
	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(nil)

	packageService.On("GetAll").Return([]models.ChartEntry{
		{
			Name:    "package_always_installed",
			Version: "1.0.0",
		},
		{
			Name:    "package_to_be_removed",
			Version: "2.0.0",
		},
		{
			Name:    "package_to_be_installed",
			Version: "3.0.0",
		},
		{
			Name:    "package_to_be_ignored",
			Version: "4.0.0",
		},
	}, nil)
	packageService.On("InstallPackage", mock.Anything, mock.Anything).Return(nil, nil)
}

// Lets an unticked package be uninstalled: its chart reads as needing nothing, and the deletion is confirmed
func mockUninstall(helmService *mocks.MockHelmService, uiService *mocks.MockUiService) *mocks.MockKubernetesService {
	kubernetesService := &mocks.MockKubernetesService{}
	helmService.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(&models.HelmChart{}, nil)
	helmService.On("UninstallChart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	uiService.On("CreateSelect", mock.MatchedBy(func(title string) bool { return strings.HasPrefix(title, "Uninstall ") }), []string{"No", "Yes"}).Return("Yes", nil)
	kubernetesService.On("NamespaceVolumes", mock.Anything, mock.Anything).Return([]models.NamespaceVolume{}, true, nil)
	kubernetesService.On("DeleteNamespace", mock.Anything, mock.Anything).Return(nil)

	return kubernetesService
}
