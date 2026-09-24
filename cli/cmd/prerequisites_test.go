package cmd

import (
	"errors"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	mediaPackage   = models.ChartEntry{Name: "bbe-media", Version: "1.0.0"}
	storagePackage = models.ChartEntry{Name: "bbe-storage", Version: "1.0.0"}
	storageNeeded  = models.ChartPrerequisite{Name: "storage", Description: "Longhorn", StorageClass: "longhorn", Package: "bbe-storage"}
	nodesNeeded    = models.ChartPrerequisite{Name: "longhorn-nodes", Talos: &models.TalosPrerequisite{Extensions: []string{"siderolabs/util-linux-tools"}, Directories: []string{"/var/mnt/longhorn"}}}
	testCluster    = models.ClusterAccess{KubeContext: "admin@test", ControlPlaneIp: "192.168.1.161", TalosConfig: "/config/talosconfig"}
)

type runnerMocks struct {
	helperService       *mocks.MockHelperService
	uiService           *mocks.MockUiService
	packageService      *mocks.MockPackageService
	talosService        *mocks.MockTalosService
	prerequisiteService *mocks.MockPrerequisiteService
	installed           []string
}

func newTestRunner(chosen []string, interactive bool) (*prerequisiteRunner, *runnerMocks) {
	m := &runnerMocks{
		helperService:       &mocks.MockHelperService{},
		uiService:           &mocks.MockUiService{},
		packageService:      &mocks.MockPackageService{},
		talosService:        &mocks.MockTalosService{},
		prerequisiteService: &mocks.MockPrerequisiteService{},
	}
	m.talosService.On("GetControlPlaneIp", mock.Anything, "controlplane.yaml").Return("192.168.1.161", nil)
	m.helperService.On("GetConfigFilePath", "talosconfig").Return("/config/talosconfig")

	bbeConfig := models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Context = "admin@test"
	runner := &prerequisiteRunner{
		helperService:       m.helperService,
		uiService:           m.uiService,
		packageService:      m.packageService,
		helmService:         &mocks.MockHelmService{},
		talosService:        m.talosService,
		prerequisiteService: m.prerequisiteService,
		bbeConfig:           bbeConfig,
		allPackages:         []models.ChartEntry{mediaPackage, storagePackage},
		chosen:              chosen,
		interactive:         interactive,
	}
	runner.install = func(pkg models.ChartEntry) error {
		m.installed = append(m.installed, pkg.Name)
		return nil
	}

	return runner, m
}

func Test_prerequisiteRunner_Succeeds_WhenPrerequisitesAreMet(t *testing.T) {
	runner, m := newTestRunner(nil, true)
	m.packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	m.prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{}, nil)

	err := runner.ensure(mediaPackage)

	assert.NoError(t, err)
	assert.Empty(t, m.installed)
	m.uiService.AssertNotCalled(t, "CreateSelect", mock.Anything, mock.Anything)
}

func Test_prerequisiteRunner_Succeeds_WithoutPrerequisites(t *testing.T) {
	runner, m := newTestRunner(nil, true)
	m.packageService.On("GetPrerequisites", mediaPackage).Return(nil, nil)

	assert.NoError(t, runner.ensure(mediaPackage))
	m.talosService.AssertNotCalled(t, "GetControlPlaneIp", mock.Anything, mock.Anything)
}

func Test_prerequisiteRunner_Succeeds_InstallingAPickedPackageFirstWithoutAsking(t *testing.T) {
	runner, m := newTestRunner([]string{"bbe-media", "bbe-storage"}, true)
	m.packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	m.packageService.On("GetPrerequisites", storagePackage).Return(nil, nil)
	m.prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{"the `longhorn` storage class"}, nil)

	err := runner.ensure(mediaPackage)

	assert.NoError(t, err)
	assert.Equal(t, []string{"bbe-storage"}, m.installed)
	m.uiService.AssertNotCalled(t, "CreateSelect", mock.Anything, mock.Anything)
}

func Test_prerequisiteRunner_Succeeds_InstallingThePackageTheUserAccepts(t *testing.T) {
	runner, m := newTestRunner([]string{"bbe-media"}, true)
	m.packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	m.packageService.On("GetPrerequisites", storagePackage).Return(nil, nil)
	m.prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{"the `longhorn` storage class"}, nil)
	m.uiService.On("CreateSelect", "Set up storage for `bbe-media` now? bbe installs the `bbe-storage` package", []string{"Yes", "No"}).Return("Yes", nil)

	err := runner.ensure(mediaPackage)

	assert.NoError(t, err)
	assert.Equal(t, []string{"bbe-storage"}, m.installed)
}

func Test_prerequisiteRunner_Fails_WhenTheUserDeclines(t *testing.T) {
	runner, m := newTestRunner([]string{"bbe-media"}, true)
	m.packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	m.prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{"the `longhorn` storage class"}, nil)
	m.uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("No", nil)

	err := runner.ensure(mediaPackage)

	assert.EqualError(t, err, "Package `bbe-media` can't be installed without storage")
	assert.Empty(t, m.installed)
}

func Test_prerequisiteRunner_Fails_WhenNotInteractive(t *testing.T) {
	runner, m := newTestRunner(nil, false)
	m.packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	m.prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{"the `longhorn` storage class"}, nil)

	err := runner.ensure(mediaPackage)

	assert.EqualError(t, err, "Package `bbe-media` needs storage. Run without --yes to set it up")
	m.uiService.AssertNotCalled(t, "CreateSelect", mock.Anything, mock.Anything)
}

func Test_prerequisiteRunner_Succeeds_PreparingNodesThenInstallingTheProvider(t *testing.T) {
	runner, m := newTestRunner([]string{"bbe-media"}, true)
	m.packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	m.packageService.On("GetPrerequisites", storagePackage).Return([]models.ChartPrerequisite{nodesNeeded}, nil)
	m.prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{"the `longhorn` storage class"}, nil)
	m.prerequisiteService.On("Check", nodesNeeded, testCluster).Return([]string{"the siderolabs/util-linux-tools extension on node `node-1`"}, nil).Once()
	m.prerequisiteService.On("Check", nodesNeeded, testCluster).Return([]string{}, nil).Once()
	m.prerequisiteService.On("PrepareNodes", *nodesNeeded.Talos, testCluster).Return(nil).Run(func(_ mock.Arguments) {
		assert.Empty(t, m.installed, "the nodes are prepared before bbe-storage is installed")
	})
	m.uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("Yes", nil)

	err := runner.ensure(mediaPackage)

	assert.NoError(t, err)
	assert.Equal(t, []string{"bbe-storage"}, m.installed)
	m.prerequisiteService.AssertNumberOfCalls(t, "PrepareNodes", 1)
	m.uiService.AssertCalled(t, "CreateSelect", "Set up longhorn-nodes for `bbe-storage` now? bbe updates each node that needs it, and nodes missing an extension are upgraded and restart one at a time", []string{"Yes", "No"})
	// The cluster is only looked up once
	m.talosService.AssertNumberOfCalls(t, "GetControlPlaneIp", 1)
}

func Test_prerequisiteRunner_Fails_WhenNodesStillMissSomethingAfterPreparing(t *testing.T) {
	runner, m := newTestRunner(nil, true)
	m.packageService.On("GetPrerequisites", storagePackage).Return([]models.ChartPrerequisite{nodesNeeded}, nil)
	m.prerequisiteService.On("Check", nodesNeeded, testCluster).Return([]string{"the /var/mnt/longhorn directory on node `node-1`"}, nil)
	m.prerequisiteService.On("PrepareNodes", *nodesNeeded.Talos, testCluster).Return(nil)
	m.uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("Yes", nil)

	err := runner.ensure(storagePackage)

	assert.EqualError(t, err, "Package `bbe-storage` still needs longhorn-nodes after setting it up, as the cluster is missing the /var/mnt/longhorn directory on node `node-1`")
}

func Test_prerequisiteRunner_Fails_WhenPreparingNodesFails(t *testing.T) {
	runner, m := newTestRunner(nil, true)
	m.packageService.On("GetPrerequisites", storagePackage).Return([]models.ChartPrerequisite{nodesNeeded}, nil)
	m.prerequisiteService.On("Check", nodesNeeded, testCluster).Return([]string{"something"}, nil)
	m.prerequisiteService.On("PrepareNodes", mock.Anything, mock.Anything).Return(errors.New("upgrade failed"))
	m.uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("Yes", nil)

	assert.EqualError(t, runner.ensure(storagePackage), "upgrade failed")
}

func Test_prerequisiteRunner_Fails_WhenTheProviderIsNotInTheLibrary(t *testing.T) {
	runner, m := newTestRunner(nil, true)
	runner.allPackages = []models.ChartEntry{mediaPackage}
	m.packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	m.prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{"the `longhorn` storage class"}, nil)

	err := runner.ensure(mediaPackage)

	assert.EqualError(t, err, "Package `bbe-media` needs storage from package `bbe-storage`, which isn't in the library")
}

func Test_prerequisiteRunner_Fails_WhenPackagesNeedEachOther(t *testing.T) {
	runner, m := newTestRunner([]string{"bbe-media", "bbe-storage"}, true)
	mediaNeeded := models.ChartPrerequisite{Name: "media", StorageClass: "media", Package: "bbe-media"}
	m.packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	m.packageService.On("GetPrerequisites", storagePackage).Return([]models.ChartPrerequisite{mediaNeeded}, nil)
	m.prerequisiteService.On("Check", mock.Anything, testCluster).Return([]string{"missing"}, nil)

	err := runner.ensure(mediaPackage)

	assert.EqualError(t, err, "Packages need each other: bbe-media -> bbe-storage -> bbe-media")
	assert.Empty(t, m.installed)
}

func Test_prerequisiteRunner_Fails_WhenTheControlPlaneIsUnknown(t *testing.T) {
	runner, m := newTestRunner(nil, true)
	m.talosService.ExpectedCalls = nil
	m.talosService.On("GetControlPlaneIp", mock.Anything, mock.Anything).Return("", errors.New("no config"))
	m.packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)

	assert.EqualError(t, runner.ensure(mediaPackage), "Failed to find the cluster's control plane: no config")
}

func Test_prerequisiteRunner_Fails_WhenChecksFail(t *testing.T) {
	runner, m := newTestRunner(nil, true)
	m.packageService.On("GetPrerequisites", mediaPackage).Return(nil, errors.New("pull failed")).Once()

	assert.EqualError(t, runner.ensure(mediaPackage), "pull failed")

	m.packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	m.prerequisiteService.On("Check", storageNeeded, testCluster).Return(nil, errors.New("kubectl is needed"))

	assert.EqualError(t, runner.ensure(mediaPackage), "kubectl is needed")
}

func Test_installCommand_Succeeds_InstallingAPrerequisiteFirstAndRecordingIt(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := newInstallMocks()
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Cluster.Context = "admin@test"
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)
	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(nil)
	packageService.On("GetAll").Return([]models.ChartEntry{mediaPackage, storagePackage}, nil)
	uiService.On("CreateMultiChoose", mock.Anything, mock.Anything, mock.Anything).Return([]string{"bbe-media"}, nil)
	uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("Yes", nil)
	helmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	talosService.On("GetControlPlaneIp", mock.Anything, mock.Anything).Return("192.168.1.161", nil)
	helperService.On("GetConfigFilePath", "talosconfig").Return("/config/talosconfig")
	packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	packageService.On("GetPrerequisites", storagePackage).Return(nil, nil)
	prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{"the `longhorn` storage class"}, nil)
	packageService.On("InstallPackage", mock.Anything, mock.Anything).Return(nil, nil)

	err := installCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService)

	assert.NoError(t, err)
	assert.Equal(t, storagePackage, packageService.Calls[indexOfCall(packageService.Calls, "InstallPackage", 0)].Arguments[0])
	assert.Equal(t, mediaPackage, packageService.Calls[indexOfCall(packageService.Calls, "InstallPackage", 1)].Arguments[0])
	configService.AssertCalled(t, "UpdateBbePackages", mock.Anything, []models.LocalPackage{
		{Name: "bbe-storage", Version: "1.0.0"},
		{Name: "bbe-media", Version: "1.0.0"},
	})
}

func Test_installCommand_Fails_WithoutAPrerequisite_ButRecordsWhatWasInstalled(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := newInstallMocks()
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Cluster.Context = "admin@test"
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)
	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(nil)
	otherPackage := models.ChartEntry{Name: "bbe-networking", Version: "1.0.0"}
	packageService.On("GetAll").Return([]models.ChartEntry{otherPackage, mediaPackage, storagePackage}, nil)
	uiService.On("CreateMultiChoose", mock.Anything, mock.Anything, mock.Anything).Return([]string{"bbe-networking", "bbe-media"}, nil)
	uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("No", nil)
	helmService.On("IsPackageInstalled", mock.Anything, mock.Anything, mock.Anything).Return(false)
	talosService.On("GetControlPlaneIp", mock.Anything, mock.Anything).Return("192.168.1.161", nil)
	helperService.On("GetConfigFilePath", "talosconfig").Return("/config/talosconfig")
	packageService.On("GetPrerequisites", otherPackage).Return(nil, nil)
	packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{"the `longhorn` storage class"}, nil)
	packageService.On("InstallPackage", otherPackage, mock.Anything).Return(nil, nil)

	err := installCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService)

	assert.ErrorContains(t, err, "Package `bbe-media` can't be installed without storage")
	packageService.AssertNotCalled(t, "InstallPackage", mediaPackage, mock.Anything)
	configService.AssertCalled(t, "UpdateBbePackages", mock.Anything, []models.LocalPackage{{Name: "bbe-networking", Version: "1.0.0"}})
}

func Test_upgradeCommand_Fails_WhenANewerVersionNeedsAPrerequisite(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := initUpgradeCommand()
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Cluster.Context = "admin@test"
	bbeConfig.Bbe.Packages = []models.LocalPackage{{Name: "bbe-media", Version: "0.2.0"}}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)
	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(nil)
	packageService.On("GetAll").Return([]models.ChartEntry{mediaPackage, storagePackage}, nil)
	talosService.On("GetControlPlaneIp", mock.Anything, mock.Anything).Return("192.168.1.161", nil)
	helperService.On("GetConfigFilePath", "talosconfig").Return("/config/talosconfig")
	packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{"the `longhorn` storage class"}, nil)

	err := upgradeCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService, true)

	assert.EqualError(t, err, "Package `bbe-media` needs storage. Run without --yes to set it up")
	packageService.AssertNotCalled(t, "UpgradePackage", mock.Anything, mock.Anything, mock.Anything)
}

func Test_upgradeCommand_Succeeds_InstallingANewPrerequisite(t *testing.T) {
	helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService := initUpgradeCommand()
	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "test"
	bbeConfig.Bbe.Cluster.Context = "admin@test"
	bbeConfig.Bbe.Packages = []models.LocalPackage{{Name: "bbe-media", Version: "0.2.0"}}
	configService.On("GetBbeConfig", mock.Anything).Return(bbeConfig, nil)
	configService.On("UpdateBbePackages", mock.Anything, mock.Anything).Return(nil)
	packageService.On("GetAll").Return([]models.ChartEntry{mediaPackage, storagePackage}, nil)
	uiService.On("CreateSelect", mock.Anything, mock.Anything).Return("Yes", nil)
	talosService.On("GetControlPlaneIp", mock.Anything, mock.Anything).Return("192.168.1.161", nil)
	helperService.On("GetConfigFilePath", "talosconfig").Return("/config/talosconfig")
	packageService.On("GetPrerequisites", mediaPackage).Return([]models.ChartPrerequisite{storageNeeded}, nil)
	packageService.On("GetPrerequisites", storagePackage).Return(nil, nil)
	prerequisiteService.On("Check", storageNeeded, testCluster).Return([]string{"the `longhorn` storage class"}, nil)
	packageService.On("InstallPackage", storagePackage, mock.Anything).Return(nil, nil)
	packageService.On("UpgradePackage", mediaPackage, mock.Anything, true).Return(nil, nil)

	err := upgradeCommand(helperService, uiService, configService, packageService, helmService, talosService, prerequisiteService, false)

	assert.NoError(t, err)
	configService.AssertCalled(t, "UpdateBbePackages", mock.Anything, []models.LocalPackage{
		{Name: "bbe-media", Version: "1.0.0"},
		{Name: "bbe-storage", Version: "1.0.0"},
	})
}

// The index in `calls` of the n-th call to `method`
func indexOfCall(calls []mock.Call, method string, n int) int {
	for i, call := range calls {
		if call.Method == method {
			if n == 0 {
				return i
			}
			n--
		}
	}

	return -1
}
