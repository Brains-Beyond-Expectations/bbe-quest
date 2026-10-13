package cmd

import (
	"errors"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const clusterContext = "admin@present-sprite"

var storageUninstall = &models.ChartUninstall{
	Namespaces: []string{"longhorn-system"},
	Patches:    []models.UninstallPatch{{Resource: "settings.longhorn.io", Namespace: "longhorn-system", Name: "deleting-confirmation-flag", Patch: `{"value": "true"}`}},
	Keeps:      []string{"Longhorn's directory on each node"},
}

type uninstallMocks struct {
	helper     *mocks.MockHelperService
	ui         *mocks.MockUiService
	config     *mocks.MockConfigService
	pkg        *mocks.MockPackageService
	helm       *mocks.MockHelmService
	kubernetes *mocks.MockKubernetesService
	bbeConfig  *models.BbeConfig
}

// bbe-networking, bbe-storage and bbe-media are installed, and bbe-media needs bbe-storage
func newUninstallMocks() uninstallMocks {
	m := uninstallMocks{
		helper: &mocks.MockHelperService{}, ui: &mocks.MockUiService{}, config: &mocks.MockConfigService{},
		pkg: &mocks.MockPackageService{}, helm: &mocks.MockHelmService{}, kubernetes: &mocks.MockKubernetesService{},
		bbeConfig: &models.BbeConfig{},
	}
	m.bbeConfig.Bbe.Cluster.Name = "present-sprite"
	m.bbeConfig.Bbe.Cluster.Context = clusterContext
	m.bbeConfig.Bbe.Packages = []models.LocalPackage{
		{Name: "bbe-networking", Version: "0.3.1"},
		{Name: "bbe-storage", Version: "0.1.0"},
		{Name: "bbe-media", Version: "0.3.0"},
	}

	m.config.On("GetBbeConfig", m.helper).Return(m.bbeConfig, nil)
	m.config.On("UpdateBbePackages", m.helper, mock.Anything).Return(nil)
	m.pkg.On("GetAll").Return([]models.ChartEntry{
		{Name: "bbe-networking", RepositoryUrl: "https://charts.example"},
		{Name: "bbe-storage", RepositoryUrl: "https://charts.example"},
		{Name: "bbe-media", RepositoryUrl: "https://charts.example"},
	}, nil)
	m.helm.On("PullChart", "https://charts.example", "bbe-networking", "0.3.1").Return(&models.HelmChart{Name: "bbe-networking"}, nil)
	m.helm.On("PullChart", "https://charts.example", "bbe-storage", "0.1.0").Return(&models.HelmChart{Name: "bbe-storage", Uninstall: storageUninstall}, nil)
	m.helm.On("PullChart", "https://charts.example", "bbe-media", "0.3.0").Return(&models.HelmChart{
		Name:          "bbe-media",
		Prerequisites: []models.ChartPrerequisite{{Name: "storage", StorageClass: "longhorn", Package: "bbe-storage"}},
		Uninstall:     &models.ChartUninstall{Keeps: []string{"The media on the NAS"}},
	}, nil)
	m.helm.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(true)
	m.helm.On("UninstallChart", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m.kubernetes.On("NamespaceVolumes", clusterContext, mock.Anything).Return([]models.NamespaceVolume{{Name: "config", Size: "1Gi"}}, true, nil)
	m.kubernetes.On("Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	m.kubernetes.On("DeleteNamespace", mock.Anything, mock.Anything).Return(nil)

	return m
}

func (m uninstallMocks) confirm(name string, answer string) {
	m.ui.On("CreateSelect", "Uninstall `"+name+"` and delete all of this?", []string{"No", "Yes"}).Return(answer, nil)
}

func (m uninstallMocks) run(args ...string) error {
	return uninstallCommand(args, m.helper, m.ui, m.config, m.pkg, m.helm, m.kubernetes)
}

// The names of the namespaces deleted, in order
func deletedNamespaces(kubernetes *mocks.MockKubernetesService) []string {
	namespaces := []string{}
	for _, call := range kubernetes.Calls {
		if call.Method == "DeleteNamespace" {
			namespaces = append(namespaces, call.Arguments.String(1))
		}
	}
	return namespaces
}

func Test_uninstallCommand_Succeeds_DeletingThePackageAndItsNamespace(t *testing.T) {
	m := newUninstallMocks()
	m.confirm("bbe-networking", "Yes")

	err := m.run("bbe-networking")

	assert.NoError(t, err)
	m.helm.AssertCalled(t, "UninstallChart", "bbe-networking", "bbe-networking", clusterContext)
	assert.Equal(t, []string{"bbe-networking"}, deletedNamespaces(m.kubernetes))
	m.config.AssertCalled(t, "UpdateBbePackages", m.helper, []models.LocalPackage{{Name: "bbe-storage", Version: "0.1.0"}, {Name: "bbe-media", Version: "0.3.0"}})
	m.kubernetes.AssertNotCalled(t, "Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func Test_uninstallCommand_Succeeds_KeepingThePackageWhenNotConfirmed(t *testing.T) {
	m := newUninstallMocks()
	m.confirm("bbe-networking", "No")

	err := m.run("bbe-networking")

	assert.NoError(t, err)
	m.helm.AssertNotCalled(t, "UninstallChart", mock.Anything, mock.Anything, mock.Anything)
	m.kubernetes.AssertNotCalled(t, "DeleteNamespace", mock.Anything, mock.Anything)
	m.config.AssertNotCalled(t, "UpdateBbePackages", mock.Anything, mock.Anything)
}

func Test_uninstallCommand_Fails_WhenAnotherPackageNeedsIt(t *testing.T) {
	m := newUninstallMocks()

	err := m.run("bbe-storage")

	assert.EqualError(t, err, "Package `bbe-storage` can't be uninstalled, as `bbe-media` needs it. Uninstall `bbe-media` first, or together with it")
	m.ui.AssertNotCalled(t, "CreateSelect", mock.Anything, mock.Anything)
	m.helm.AssertNotCalled(t, "UninstallChart", mock.Anything, mock.Anything, mock.Anything)
}

func Test_uninstallCommand_Succeeds_RemovingWhatNeedsAPackageFirst(t *testing.T) {
	m := newUninstallMocks()
	m.ui.On("CreateMultiChoose", "Select packages to uninstall", []string{"bbe-networking", "bbe-storage", "bbe-media"}, []int(nil)).Return([]string{"bbe-storage", "bbe-media"}, nil)
	m.confirm("bbe-media", "Yes")
	m.confirm("bbe-storage", "Yes")

	err := m.run()

	assert.NoError(t, err)
	assert.Equal(t, []string{"bbe-media", "bbe-storage", "longhorn-system"}, deletedNamespaces(m.kubernetes))
	// Longhorn only lets itself be removed once its flag is set
	m.kubernetes.AssertCalled(t, "Patch", clusterContext, "longhorn-system", "settings.longhorn.io", "deleting-confirmation-flag", `{"value": "true"}`)
	m.kubernetes.AssertCalled(t, "NamespaceVolumes", clusterContext, "longhorn-system")
	m.config.AssertCalled(t, "UpdateBbePackages", m.helper, []models.LocalPackage{{Name: "bbe-networking", Version: "0.3.1"}})
}

func Test_uninstallCommand_Succeeds_DeletingTheNamespaceOfAReleaseThatsGone(t *testing.T) {
	m := newUninstallMocks()
	m.helm.ExpectedCalls = removeCalls(m.helm.ExpectedCalls, "IsPackageInstalled")
	m.helm.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	m.kubernetes.ExpectedCalls = removeCalls(m.kubernetes.ExpectedCalls, "NamespaceVolumes")
	m.kubernetes.On("NamespaceVolumes", mock.Anything, mock.Anything).Return(nil, false, nil)
	m.confirm("bbe-networking", "Yes")

	err := m.run("bbe-networking")

	assert.NoError(t, err)
	m.helm.AssertNotCalled(t, "UninstallChart", mock.Anything, mock.Anything, mock.Anything)
	assert.Equal(t, []string{"bbe-networking"}, deletedNamespaces(m.kubernetes))
}

func Test_uninstallCommand_Succeeds_WithoutReadingTheChart(t *testing.T) {
	cases := map[string]func(m uninstallMocks){
		"not in the library": func(m uninstallMocks) {
			m.pkg.ExpectedCalls = nil
			m.pkg.On("GetAll").Return([]models.ChartEntry{}, nil)
		},
		"unreadable": func(m uninstallMocks) {
			m.helm.ExpectedCalls = removeCalls(m.helm.ExpectedCalls, "PullChart")
			m.helm.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("Mock failed to pull"))
		},
	}

	for name, setup := range cases {
		m := newUninstallMocks()
		setup(m)
		m.confirm("bbe-storage", "Yes")

		err := m.run("bbe-storage")

		// Without the charts, bbe can't tell bbe-media needs it, and only deletes the package's own namespace
		assert.NoError(t, err, name)
		assert.Equal(t, []string{"bbe-storage"}, deletedNamespaces(m.kubernetes), name)
	}
}

func Test_uninstallCommand_Fails(t *testing.T) {
	failed := errors.New("Mock failure")
	cases := map[string]func(m uninstallMocks){
		"No BBE cluster found, please run 'bbe setup' to create your cluster": func(m uninstallMocks) {
			m.config.ExpectedCalls = nil
			m.config.On("GetBbeConfig", m.helper).Return(&models.BbeConfig{}, nil)
		},
		"No packages are installed": func(m uninstallMocks) {
			m.bbeConfig.Bbe.Packages = nil
		},
		"Package `bbe-library` isn't installed": func(m uninstallMocks) {},
		"Mock failed to read the library": func(m uninstallMocks) {
			m.pkg.ExpectedCalls = nil
			m.pkg.On("GetAll").Return([]models.ChartEntry(nil), errors.New("Mock failed to read the library"))
		},
		"Mock failed to read volumes": func(m uninstallMocks) {
			m.kubernetes.ExpectedCalls = removeCalls(m.kubernetes.ExpectedCalls, "NamespaceVolumes")
			m.kubernetes.On("NamespaceVolumes", mock.Anything, mock.Anything).Return(nil, false, errors.New("Mock failed to read volumes"))
		},
		"Mock prompt cancelled": func(m uninstallMocks) {
			m.ui.On("CreateSelect", mock.Anything, mock.Anything).Return("", errors.New("Mock prompt cancelled"))
		},
		"Mock failed to patch": func(m uninstallMocks) {
			m.confirm("bbe-storage", "Yes")
			m.kubernetes.ExpectedCalls = removeCalls(m.kubernetes.ExpectedCalls, "Patch")
			m.kubernetes.On("Patch", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("Mock failed to patch"))
		},
		"Mock failed to uninstall": func(m uninstallMocks) {
			m.confirm("bbe-storage", "Yes")
			m.helm.ExpectedCalls = removeCalls(m.helm.ExpectedCalls, "UninstallChart")
			m.helm.On("UninstallChart", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("Mock failed to uninstall"))
		},
		"Mock failed to delete": func(m uninstallMocks) {
			m.confirm("bbe-storage", "Yes")
			m.kubernetes.ExpectedCalls = removeCalls(m.kubernetes.ExpectedCalls, "DeleteNamespace")
			m.kubernetes.On("DeleteNamespace", mock.Anything, mock.Anything).Return(errors.New("Mock failed to delete"))
		},
		"Failed to update BBE configuration: Mock failure": func(m uninstallMocks) {
			m.confirm("bbe-storage", "Yes")
			m.config.ExpectedCalls = removeCalls(m.config.ExpectedCalls, "UpdateBbePackages")
			m.config.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(failed)
		},
	}

	for expected, setup := range cases {
		m := newUninstallMocks()
		setup(m)
		args := []string{"bbe-media", "bbe-storage"}
		if expected == "Package `bbe-library` isn't installed" {
			args = []string{"bbe-library"}
		}
		m.confirm("bbe-media", "Yes")

		err := m.run(args...)

		assert.EqualError(t, err, expected, expected)
	}
}

func Test_uninstallCommand_Fails_WhenChoosingFails(t *testing.T) {
	m := newUninstallMocks()
	m.ui.On("CreateMultiChoose", mock.Anything, mock.Anything, mock.Anything).Return([]string(nil), errors.New("Mock prompt cancelled"))

	err := m.run()

	assert.EqualError(t, err, "Mock prompt cancelled")
}

func removeCalls(calls []*mock.Call, method string) []*mock.Call {
	kept := []*mock.Call{}
	for _, call := range calls {
		if call.Method != method {
			kept = append(kept, call)
		}
	}
	return kept
}
