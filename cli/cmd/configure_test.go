package cmd

import (
	"errors"
	"fmt"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/mocks"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var indexersSetting = models.ChartSetting{
	Name:   "indexers",
	Values: "bbe.media.prowlarr.indexers",
	Secret: "bbe-media-indexers",
	Catalog: &models.SettingCatalog{
		Type:    "prowlarr-indexers",
		Service: "bbe-media-prowlarr",
		Port:    9696,
		ApiKey:  &models.SecretKeyRef{Secret: "bbe-media-api-keys", Key: "prowlarr"},
	},
}

var indexerDefinitions = []models.IndexerDefinition{
	{
		Name: "BeyondHD", DefinitionName: "BeyondHD", Privacy: "private", Protocol: "torrent",
		IndexerUrls: []string{"https://beyond-hd.me/", "https://beyondhd.co/"},
		Fields: []models.IndexerField{
			{Name: "baseUrl", Label: "Base Url", Type: "select"},
			{Name: "apiKey", Label: "API Key", Type: "textbox", Privacy: "apiKey"},
			{Name: "rssKey", Label: "RSS Key", Type: "textbox", Privacy: "apiKey"},
			{Name: "freeleechOnly", Label: "Freeleech Only", Type: "checkbox", Value: false},
			{Name: "searchTypes", Label: "Search Types", Type: "select", Value: []interface{}{}},
			{Name: "baseSettings.queryLimit", Label: "Query Limit", Type: "number", Advanced: true},
		},
	},
	{
		Name: "LinuxTracker", DefinitionName: "linuxtracker", Privacy: "public", Protocol: "torrent",
		IndexerUrls: []string{"https://linuxtracker.org/"},
		Fields: []models.IndexerField{
			{Name: "definitionFile", Type: "textbox", Value: "linuxtracker"},
			{Name: "baseUrl", Label: "Base Url", Type: "select"},
			{Name: "info", Label: "About", Type: "info"},
			{Name: "torrentBaseSettings.seedRatio", Label: "Seed Ratio", Type: "number"},
			{Name: "titleOnly", Label: "Search only in title", Type: "checkbox", Value: true},
			{Name: "sort", Label: "Sort requested from site", Type: "select", Value: float64(0), SelectOptions: []models.IndexerFieldOption{
				{Value: float64(0), Name: "created"},
				{Value: float64(2), Name: "seeders"},
			}},
		},
	},
	{
		Name: "CaptchaTracker", DefinitionName: "captchatracker", Privacy: "private", Protocol: "torrent",
		Fields: []models.IndexerField{{Name: "captcha", Label: "Captcha", Type: "cardigannCaptcha"}},
	},
}

type configureMocks struct {
	helper     *mocks.MockHelperService
	ui         *mocks.MockUiService
	config     *mocks.MockConfigService
	pkg        *mocks.MockPackageService
	helm       *mocks.MockHelmService
	kubernetes *mocks.MockKubernetesService
	catalog    *mocks.MockCatalogService
	bbeConfig  *models.BbeConfig
	stopped    *bool
}

func newConfigureMocks(indexers []interface{}) configureMocks {
	m := configureMocks{
		helper: &mocks.MockHelperService{}, ui: &mocks.MockUiService{}, config: &mocks.MockConfigService{},
		pkg: &mocks.MockPackageService{}, helm: &mocks.MockHelmService{}, kubernetes: &mocks.MockKubernetesService{},
		catalog: &mocks.MockCatalogService{}, stopped: new(bool),
	}

	bbeConfig := &models.BbeConfig{}
	bbeConfig.Bbe.Cluster.Name = "present-sprite"
	bbeConfig.Bbe.Cluster.Context = "admin@present-sprite"
	// As yaml.v2 reads bbe.yaml
	media := map[interface{}]interface{}{"nfs": map[interface{}]interface{}{"server": "192.168.1.219"}}
	if indexers != nil {
		media["prowlarr"] = map[interface{}]interface{}{"indexers": indexers}
	}
	bbeConfig.Bbe.Packages = []models.LocalPackage{
		{Name: "bbe-networking", Version: "0.3.1"},
		{Name: "bbe-media", Version: "0.2.0", Values: map[string]interface{}{"bbe": map[interface{}]interface{}{"media": media}}},
	}

	m.bbeConfig = bbeConfig
	m.config.On("GetBbeConfig", m.helper).Return(bbeConfig, nil)
	m.config.On("UpdateBbePackages", m.helper, mock.Anything).Return(nil)
	m.pkg.On("GetAll").Return([]models.ChartEntry{{Name: "bbe-media", Version: "0.3.0", RepositoryUrl: "https://charts.example", RepositoryName: "bbe"}}, nil)
	m.helm.On("PullChart", "https://charts.example", "bbe-media", "0.2.0").Return(&models.HelmChart{Name: "bbe-media", Settings: []models.ChartSetting{indexersSetting}}, nil)
	m.kubernetes.On("GetSecretValue", "admin@present-sprite", "bbe-media", "bbe-media-api-keys", "prowlarr").Return("prowlarr-key", nil)
	m.kubernetes.On("PortForward", "admin@present-sprite", "bbe-media", "bbe-media-prowlarr", 9696).Return(43210, func() { *m.stopped = true }, nil)
	m.catalog.On("IndexerDefinitions", "http://127.0.0.1:43210", "prowlarr-key").Return(indexerDefinitions, nil)

	return m
}

func (m configureMocks) run(args ...string) error {
	return configureCommand(args, m.helper, m.ui, m.config, m.pkg, m.helm, m.kubernetes, m.catalog)
}

const actionTitle = "What would you like to do with the indexers?"
const searchTitle = "Search Prowlarr's indexers by name (empty to cancel)"

var actionsWithoutEntries = []string{"Add", "Done"}
var actionsWithEntries = []string{"Add", "Remove", "Done"}

// The values the package is upgraded with, as the mocks' bbe.yaml holds them plus the indexers
func mediaValues(indexers ...interface{}) map[string]interface{} {
	return map[string]interface{}{"bbe": map[string]interface{}{"media": map[string]interface{}{
		"nfs":      map[string]interface{}{"server": "192.168.1.219"},
		"prowlarr": map[string]interface{}{"indexers": append([]interface{}{}, indexers...)},
	}}}
}

var mediaChart = models.ChartEntry{Name: "bbe-media", Version: "0.2.0", RepositoryUrl: "https://charts.example", RepositoryName: "bbe"}

func Test_configureCommand_Succeeds_AddingAnIndexerWithCredentials(t *testing.T) {
	m := newConfigureMocks(nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Add", nil).Once()
	m.ui.On("CreateInput", searchTitle, "").Return(" Beyond ", nil).Once()
	m.ui.On("CreateSelect", "1 indexers match", []string{"BeyondHD (private torrent)", "Search again"}).Return("BeyondHD (private torrent)", nil)
	m.ui.On("CreateInput", "Name it", "BeyondHD").Return("", nil)
	m.ui.On("CreateSelect", "BeyondHD: Base Url", []string{"https://beyond-hd.me/", "https://beyondhd.co/"}).Return("https://beyondhd.co/", nil)
	m.ui.On("CreatePassword", "BeyondHD: API Key").Return("secret-key", nil)
	m.ui.On("CreatePassword", "BeyondHD: RSS Key").Return("", nil)
	m.ui.On("CreateSelect", "BeyondHD: Freeleech Only", []string{"No", "Yes"}).Return("Yes", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Done", nil)
	m.kubernetes.On("SetSecretValues", "admin@present-sprite", "bbe-media", "bbe-media-indexers", map[string]string{"beyondhd.apiKey": "secret-key"}).Return(nil)
	expected := mediaValues(map[string]interface{}{
		"name":         "BeyondHD",
		"definition":   "BeyondHD",
		"fields":       map[string]interface{}{"baseUrl": "https://beyondhd.co/", "freeleechOnly": true},
		"secretFields": map[string]interface{}{"apiKey": "beyondhd.apiKey"},
	})
	m.pkg.On("UpgradePackage", mediaChart, expected, true).Return(expected, nil)

	err := m.run("bbe-media")

	assert.NoError(t, err)
	assert.True(t, *m.stopped)
	m.pkg.AssertCalled(t, "UpgradePackage", mediaChart, expected, true)
	m.config.AssertCalled(t, "UpdateBbePackages", m.helper, mock.MatchedBy(func(packages []models.LocalPackage) bool {
		return assert.ObjectsAreEqual(expected, packages[1].Values)
	}))
	m.kubernetes.AssertNotCalled(t, "RemoveSecretValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func Test_configureCommand_Succeeds_AddingAPublicIndexerWithOnlyChangedFields(t *testing.T) {
	m := newConfigureMocks(nil)
	m.ui.On("CreateSelect", "Which package would you like to configure?", []string{"bbe-networking", "bbe-media"}).Return("bbe-media", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Add", nil).Once()
	m.ui.On("CreateInput", searchTitle, "").Return("nothing like it", nil).Once()
	m.ui.On("CreateInput", searchTitle, "").Return("tracker", nil).Once()
	m.ui.On("CreateSelect", "2 indexers match", []string{"CaptchaTracker (private torrent)", "LinuxTracker (public torrent)", "Search again"}).Return("Search again", nil).Once()
	m.ui.On("CreateInput", searchTitle, "").Return("linux", nil).Once()
	m.ui.On("CreateSelect", "1 indexers match", []string{"LinuxTracker (public torrent)", "Search again"}).Return("LinuxTracker (public torrent)", nil)
	m.ui.On("CreateInput", "Name it", "LinuxTracker").Return("Linux ISOs", nil)
	m.ui.On("CreateInput", "Linux ISOs: Seed Ratio", "").Return("lots", nil).Once()
	m.ui.On("CreateInput", "Linux ISOs: Seed Ratio", "").Return("2", nil).Once()
	m.ui.On("CreateSelect", "Linux ISOs: Search only in title", []string{"Yes", "No"}).Return("Yes", nil)
	m.ui.On("CreateSelect", "Linux ISOs: Sort requested from site", []string{"created", "seeders"}).Return("seeders", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Done", nil)
	expected := mediaValues(map[string]interface{}{
		"name":       "Linux ISOs",
		"definition": "linuxtracker",
		"fields":     map[string]interface{}{"baseUrl": "https://linuxtracker.org/", "torrentBaseSettings.seedRatio": int64(2), "sort": int64(2)},
	})
	m.pkg.On("UpgradePackage", mediaChart, expected, true).Return(expected, nil)

	err := m.run()

	assert.NoError(t, err)
	m.pkg.AssertCalled(t, "UpgradePackage", mediaChart, expected, true)
	m.kubernetes.AssertNotCalled(t, "SetSecretValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func Test_configureCommand_Succeeds_RemovingAnIndexerAndItsCredentials(t *testing.T) {
	m := newConfigureMocks([]interface{}{
		map[interface{}]interface{}{"name": "BeyondHD", "definition": "BeyondHD", "secretFields": map[interface{}]interface{}{"apiKey": "beyondhd.apiKey"}},
		map[interface{}]interface{}{"name": "Linux ISOs", "definition": "linuxtracker"},
	})
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Remove", nil).Once()
	m.ui.On("CreateSelect", "Which of the indexers would you like to remove?", []string{"BeyondHD", "Linux ISOs", "Cancel"}).Return("BeyondHD", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Done", nil).Once()
	expected := mediaValues(map[string]interface{}{"name": "Linux ISOs", "definition": "linuxtracker"})
	m.pkg.On("UpgradePackage", mediaChart, expected, true).Return(expected, nil)
	m.kubernetes.On("RemoveSecretValues", "admin@present-sprite", "bbe-media", "bbe-media-indexers", []string{"beyondhd.apiKey"}).Return(nil)

	err := m.run("bbe-media")

	assert.NoError(t, err)
	m.pkg.AssertCalled(t, "UpgradePackage", mediaChart, expected, true)
	m.kubernetes.AssertCalled(t, "RemoveSecretValues", "admin@present-sprite", "bbe-media", "bbe-media-indexers", []string{"beyondhd.apiKey"})
	m.kubernetes.AssertNotCalled(t, "PortForward", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func Test_configureCommand_Succeeds_ChangingNothing(t *testing.T) {
	m := newConfigureMocks([]interface{}{map[interface{}]interface{}{"name": "Linux ISOs", "definition": "linuxtracker"}})
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Remove", nil).Once()
	m.ui.On("CreateSelect", "Which of the indexers would you like to remove?", []string{"Linux ISOs", "Cancel"}).Return("Cancel", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Add", nil).Once()
	m.ui.On("CreateInput", searchTitle, "").Return("", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Done", nil).Once()

	err := m.run("bbe-media")

	assert.NoError(t, err)
	m.pkg.AssertNotCalled(t, "UpgradePackage", mock.Anything, mock.Anything, mock.Anything)
	m.config.AssertNotCalled(t, "UpdateBbePackages", mock.Anything, mock.Anything)
}

func Test_configureCommand_Succeeds_SkippingAnIndexerThatNeedsACaptcha(t *testing.T) {
	m := newConfigureMocks(nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Add", nil).Once()
	m.ui.On("CreateInput", searchTitle, "").Return("captcha", nil)
	m.ui.On("CreateSelect", "1 indexers match", []string{"CaptchaTracker (private torrent)", "Search again"}).Return("CaptchaTracker (private torrent)", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Done", nil).Once()

	err := m.run("bbe-media")

	assert.NoError(t, err)
	m.pkg.AssertNotCalled(t, "UpgradePackage", mock.Anything, mock.Anything, mock.Anything)
}

func Test_configureCommand_Succeeds_AskingForANameNotTaken(t *testing.T) {
	m := newConfigureMocks([]interface{}{map[interface{}]interface{}{"name": "LinuxTracker", "definition": "linuxtracker"}})
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Add", nil).Once()
	m.ui.On("CreateInput", searchTitle, "").Return("linux", nil)
	m.ui.On("CreateSelect", "1 indexers match", []string{"LinuxTracker (public torrent)", "Search again"}).Return("LinuxTracker (public torrent)", nil)
	m.ui.On("CreateInput", "Name it", "LinuxTracker").Return("LinuxTracker", nil).Once()
	m.ui.On("CreateInput", "Name it", "LinuxTracker").Return("Linux mirror", nil).Once()
	m.ui.On("CreateInput", "Linux mirror: Seed Ratio", "").Return("", nil)
	m.ui.On("CreateSelect", "Linux mirror: Search only in title", []string{"Yes", "No"}).Return("No", nil)
	m.ui.On("CreateSelect", "Linux mirror: Sort requested from site", []string{"created", "seeders"}).Return("created", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Done", nil).Once()
	expected := mediaValues(
		map[string]interface{}{"name": "LinuxTracker", "definition": "linuxtracker"},
		map[string]interface{}{"name": "Linux mirror", "definition": "linuxtracker", "fields": map[string]interface{}{"baseUrl": "https://linuxtracker.org/", "titleOnly": false}},
	)
	m.pkg.On("UpgradePackage", mediaChart, expected, true).Return(expected, nil)

	err := m.run("bbe-media")

	assert.NoError(t, err)
	m.pkg.AssertCalled(t, "UpgradePackage", mediaChart, expected, true)
}

func Test_configureCommand_Succeeds_WhenThePackageHasNothingToConfigure(t *testing.T) {
	m := newConfigureMocks(nil)
	m.pkg.ExpectedCalls = nil
	m.pkg.On("GetAll").Return([]models.ChartEntry{{Name: "bbe-networking", Version: "0.3.1", RepositoryUrl: "https://charts.example"}}, nil)
	m.helm.On("PullChart", "https://charts.example", "bbe-networking", "0.3.1").Return(&models.HelmChart{Name: "bbe-networking"}, nil)

	err := m.run("bbe-networking")

	assert.NoError(t, err)
	m.ui.AssertNotCalled(t, "CreateSelect", mock.Anything, mock.Anything)
}

func Test_configureCommand_Succeeds_ChoosingBetweenSettings(t *testing.T) {
	m := newConfigureMocks(nil)
	other := models.ChartSetting{Name: "lists", Description: "Other lists", Values: "bbe.lists", Catalog: indexersSetting.Catalog}
	indexers := indexersSetting
	indexers.Description = "The indexers"
	m.helm.ExpectedCalls = nil
	m.helm.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(&models.HelmChart{Settings: []models.ChartSetting{other, indexers}}, nil)
	m.ui.On("CreateSelect", "Which setting would you like to configure?", []string{"lists: Other lists", "indexers: The indexers"}).Return("indexers: The indexers", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Done", nil)

	err := m.run("bbe-media")

	assert.NoError(t, err)
	m.ui.AssertCalled(t, "CreateSelect", actionTitle, actionsWithoutEntries)
}

func Test_configureCommand_Fails(t *testing.T) {
	cases := map[string]func(m configureMocks){
		"No BBE cluster found, please run 'bbe setup' to create your cluster": func(m configureMocks) {
			m.config.ExpectedCalls = nil
			m.config.On("GetBbeConfig", m.helper).Return(&models.BbeConfig{}, nil)
		},
		"Package `bbe-library` isn't installed, install it with 'bbe install'": func(m configureMocks) {},
		"Package `bbe-media` isn't in the package library": func(m configureMocks) {
			m.pkg.ExpectedCalls = nil
			m.pkg.On("GetAll").Return([]models.ChartEntry{}, nil)
		},
		"Mock failed to read the library": func(m configureMocks) {
			m.pkg.ExpectedCalls = nil
			m.pkg.On("GetAll").Return([]models.ChartEntry(nil), errors.New("Mock failed to read the library"))
		},
		"Mock failed to pull": func(m configureMocks) {
			m.helm.ExpectedCalls = nil
			m.helm.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("Mock failed to pull"))
		},
		"Mock prompt cancelled": func(m configureMocks) {
			m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("", errors.New("Mock prompt cancelled"))
		},
		"Secret `bbe-media-api-keys` has no key `prowlarr`": func(m configureMocks) {
			m.kubernetes.ExpectedCalls = nil
			m.kubernetes.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("", errors.New("Secret `bbe-media-api-keys` has no key `prowlarr`"))
			m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Add", nil)
		},
		"Mock failed to forward": func(m configureMocks) {
			m.kubernetes.ExpectedCalls = nil
			m.kubernetes.On("GetSecretValue", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("key", nil)
			m.kubernetes.On("PortForward", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(0, nil, errors.New("Mock failed to forward"))
			m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Add", nil)
		},
		"Mock failed to read Prowlarr": func(m configureMocks) {
			m.catalog.ExpectedCalls = nil
			m.catalog.On("IndexerDefinitions", mock.Anything, mock.Anything).Return(nil, errors.New("Mock failed to read Prowlarr"))
			m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Add", nil)
		},
	}

	for expected, setup := range cases {
		m := newConfigureMocks(nil)
		setup(m)
		args := []string{"bbe-media"}
		if expected == "Package `bbe-library` isn't installed, install it with 'bbe install'" {
			args = []string{"bbe-library"}
		}

		err := m.run(args...)

		assert.EqualError(t, err, expected, expected)
	}
}

func Test_configureCommand_Fails_SavingTheChange(t *testing.T) {
	cases := map[string]func(m configureMocks){
		"Mock failed to save the secret": func(m configureMocks) {
			m.kubernetes.On("SetSecretValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("Mock failed to save the secret"))
		},
		"Mock failed to upgrade": func(m configureMocks) {
			m.kubernetes.On("SetSecretValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			m.pkg.On("UpgradePackage", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("Mock failed to upgrade"))
		},
		"Failed to update BBE configuration: Mock failed to write bbe.yaml": func(m configureMocks) {
			m.kubernetes.On("SetSecretValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
			m.pkg.On("UpgradePackage", mock.Anything, mock.Anything, mock.Anything).Return(map[string]interface{}{}, nil)
			m.config.ExpectedCalls = nil
			m.config.On("GetBbeConfig", m.helper).Return(m.bbeConfig, nil)
			m.config.On("UpdateBbePackages", m.helper, mock.Anything).Return(errors.New("Mock failed to write bbe.yaml"))
		},
	}

	for expected, setup := range cases {
		m := newConfigureMocks(nil)
		m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Add", nil).Once()
		m.ui.On("CreateInput", searchTitle, "").Return("beyond", nil)
		m.ui.On("CreateSelect", "1 indexers match", mock.Anything).Return("BeyondHD (private torrent)", nil)
		m.ui.On("CreateInput", "Name it", "BeyondHD").Return("", nil)
		m.ui.On("CreateSelect", "BeyondHD: Base Url", mock.Anything).Return("https://beyond-hd.me/", nil)
		m.ui.On("CreatePassword", mock.Anything).Return("secret", nil)
		m.ui.On("CreateSelect", "BeyondHD: Freeleech Only", mock.Anything).Return("No", nil)
		m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Done", nil)
		setup(m)

		err := m.run("bbe-media")

		assert.EqualError(t, err, expected, expected)
	}
}

func Test_configureCommand_Fails_RemovingTheCredentials(t *testing.T) {
	m := newConfigureMocks([]interface{}{map[interface{}]interface{}{"name": "BeyondHD", "definition": "BeyondHD", "secretFields": map[interface{}]interface{}{"apiKey": "beyondhd.apiKey"}}})
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Remove", nil).Once()
	m.ui.On("CreateSelect", "Which of the indexers would you like to remove?", mock.Anything).Return("BeyondHD", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Done", nil)
	m.pkg.On("UpgradePackage", mock.Anything, mock.Anything, mock.Anything).Return(map[string]interface{}{}, nil)
	m.kubernetes.On("RemoveSecretValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(errors.New("Mock failed to remove"))

	err := m.run("bbe-media")

	assert.EqualError(t, err, "Mock failed to remove")
}

func Test_secretKey(t *testing.T) {
	assert.Equal(t, "beyondhd.apiKey", secretKey("BeyondHD", "apiKey"))
	assert.Equal(t, "my-private-tracker.password", secretKey(" My Private Tracker! ", "password"))
}

var customDefinition = models.IndexerDefinition{
	Name: "Custom", DefinitionName: "custom", Privacy: "public", Protocol: "torrent",
	Fields: []models.IndexerField{
		{Name: "userAgent", Label: "User agent", Type: "textbox", Value: "bbe"},
		{Name: "sort", Label: "Sort", Type: "select", Value: float64(2), SelectOptions: []models.IndexerFieldOption{
			{Value: float64(0), Name: "created"},
			{Value: float64(2), Name: "seeders"},
		}},
		{Name: "order", Label: "Order", Type: "select", Value: "desc", SelectOptions: []models.IndexerFieldOption{
			{Value: "asc", Name: "ascending"},
			{Value: "desc", Name: "descending"},
		}},
		{Name: "limit", Label: "Limit", Type: "number", Value: float64(5)},
	},
}

func Test_configureCommand_Succeeds_RemovingAndAddingAgainInOneGo(t *testing.T) {
	m := newConfigureMocks([]interface{}{
		map[interface{}]interface{}{"name": "BeyondHD", "definition": "BeyondHD", "secretFields": map[interface{}]interface{}{"apiKey": "beyondhd.apiKey"}},
	})
	m.catalog.ExpectedCalls = nil
	m.catalog.On("IndexerDefinitions", mock.Anything, mock.Anything).Return(append(indexerDefinitions, customDefinition), nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Remove", nil).Once()
	m.ui.On("CreateSelect", "Which of the indexers would you like to remove?", []string{"BeyondHD", "Cancel"}).Return("BeyondHD", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Add", nil).Once()
	m.ui.On("CreateInput", searchTitle, "").Return("beyond", nil).Once()
	m.ui.On("CreateSelect", "1 indexers match", []string{"BeyondHD (private torrent)", "Search again"}).Return("BeyondHD (private torrent)", nil)
	m.ui.On("CreateInput", "Name it", "BeyondHD").Return("", nil)
	m.ui.On("CreateSelect", "BeyondHD: Base Url", mock.Anything).Return("https://beyond-hd.me/", nil)
	m.ui.On("CreatePassword", "BeyondHD: API Key").Return("new-key", nil)
	m.ui.On("CreatePassword", "BeyondHD: RSS Key").Return("", nil)
	m.ui.On("CreateSelect", "BeyondHD: Freeleech Only", mock.Anything).Return("No", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Add", nil).Once()
	m.ui.On("CreateInput", searchTitle, "").Return("custom", nil).Once()
	m.ui.On("CreateSelect", "1 indexers match", []string{"Custom (public torrent)", "Search again"}).Return("Custom (public torrent)", nil)
	m.ui.On("CreateInput", "Name it", "Custom").Return("", nil)
	m.ui.On("CreateInput", "Custom: User agent", "bbe").Return("firefox", nil)
	m.ui.On("CreateSelect", "Custom: Sort", []string{"seeders", "created"}).Return("created", nil)
	m.ui.On("CreateSelect", "Custom: Order", []string{"descending", "ascending"}).Return("ascending", nil)
	m.ui.On("CreateInput", "Custom: Limit", "5").Return("5", nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithEntries).Return("Done", nil).Once()
	m.kubernetes.On("SetSecretValues", mock.Anything, mock.Anything, mock.Anything, map[string]string{"beyondhd.apiKey": "new-key"}).Return(nil)
	expected := mediaValues(
		map[string]interface{}{"name": "BeyondHD", "definition": "BeyondHD", "fields": map[string]interface{}{"baseUrl": "https://beyond-hd.me/"}, "secretFields": map[string]interface{}{"apiKey": "beyondhd.apiKey"}},
		map[string]interface{}{"name": "Custom", "definition": "custom", "fields": map[string]interface{}{"userAgent": "firefox", "sort": int64(0), "order": "asc"}},
	)
	m.pkg.On("UpgradePackage", mediaChart, expected, true).Return(expected, nil)

	err := m.run("bbe-media")

	assert.NoError(t, err)
	m.pkg.AssertCalled(t, "UpgradePackage", mediaChart, expected, true)
	// The catalog is read once, and the credential added again isn't removed
	m.catalog.AssertNumberOfCalls(t, "IndexerDefinitions", 1)
	m.kubernetes.AssertNotCalled(t, "RemoveSecretValues", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func Test_configureCommand_Succeeds_ShowingTheFirstMatches(t *testing.T) {
	m := newConfigureMocks(nil)
	many := []models.IndexerDefinition{}
	for i := 0; i < 30; i++ {
		many = append(many, models.IndexerDefinition{Name: fmt.Sprintf("Tracker %02d", i), DefinitionName: fmt.Sprintf("tracker%02d", i), Privacy: "public", Protocol: "torrent"})
	}
	m.catalog.ExpectedCalls = nil
	m.catalog.On("IndexerDefinitions", mock.Anything, mock.Anything).Return(many, nil)
	m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Add", nil).Once()
	m.ui.On("CreateInput", searchTitle, "").Return("tracker", nil).Once()
	m.ui.On("CreateSelect", "30 indexers match, the first 25 are shown", mock.MatchedBy(func(options []string) bool {
		return len(options) == 26 && options[0] == "Tracker 00 (public torrent)" && options[25] == "Search again"
	})).Return("Search again", nil)
	m.ui.On("CreateInput", searchTitle, "").Return("", nil).Once()
	m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Done", nil).Once()

	err := m.run("bbe-media")

	assert.NoError(t, err)
}

func Test_configureCommand_Succeeds_ForAPackageWithoutValues(t *testing.T) {
	m := newConfigureMocks(nil)
	m.bbeConfig.Bbe.Packages[1].Values = nil
	m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Done", nil)

	err := m.run("bbe-media")

	assert.NoError(t, err)
}

func Test_configureCommand_Fails_WhenNothingIsInstalled(t *testing.T) {
	m := newConfigureMocks(nil)
	m.bbeConfig.Bbe.Packages = nil

	err := m.run()

	assert.EqualError(t, err, "No packages are installed, install one with 'bbe install'")
}

func Test_configureCommand_Fails_WhenAPromptFails(t *testing.T) {
	failed := errors.New("Mock prompt cancelled")
	addUntil := func(m configureMocks) {
		m.ui.On("CreateSelect", actionTitle, actionsWithoutEntries).Return("Add", nil)
	}
	choose := func(query string, option string) func(m configureMocks) {
		return func(m configureMocks) {
			addUntil(m)
			m.ui.On("CreateInput", searchTitle, "").Return(query, nil)
			m.ui.On("CreateSelect", "1 indexers match", mock.Anything).Return(option, nil)
		}
	}
	cases := map[string]func(m configureMocks){
		"choosing the package": func(m configureMocks) {
			m.ui.On("CreateSelect", "Which package would you like to configure?", mock.Anything).Return("", failed)
		},
		"choosing the setting": func(m configureMocks) {
			m.helm.ExpectedCalls = nil
			m.helm.On("PullChart", mock.Anything, mock.Anything, mock.Anything).Return(&models.HelmChart{Settings: []models.ChartSetting{indexersSetting, indexersSetting}}, nil)
			m.ui.On("CreateSelect", "Which setting would you like to configure?", mock.Anything).Return("", failed)
		},
		"searching": func(m configureMocks) {
			addUntil(m)
			m.ui.On("CreateInput", searchTitle, "").Return("", failed)
		},
		"picking a match": func(m configureMocks) {
			addUntil(m)
			m.ui.On("CreateInput", searchTitle, "").Return("linux", nil)
			m.ui.On("CreateSelect", "1 indexers match", mock.Anything).Return("", failed)
		},
		"naming it": func(m configureMocks) {
			choose("linux", "LinuxTracker (public torrent)")(m)
			m.ui.On("CreateInput", "Name it", mock.Anything).Return("", failed)
		},
		"a number": func(m configureMocks) {
			choose("linux", "LinuxTracker (public torrent)")(m)
			m.ui.On("CreateInput", "Name it", mock.Anything).Return("", nil)
			m.ui.On("CreateInput", "LinuxTracker: Seed Ratio", mock.Anything).Return("", failed)
		},
		"a select": func(m configureMocks) {
			choose("linux", "LinuxTracker (public torrent)")(m)
			m.ui.On("CreateInput", "Name it", mock.Anything).Return("", nil)
			m.ui.On("CreateInput", "LinuxTracker: Seed Ratio", mock.Anything).Return("", nil)
			m.ui.On("CreateSelect", "LinuxTracker: Search only in title", mock.Anything).Return("Yes", nil)
			m.ui.On("CreateSelect", "LinuxTracker: Sort requested from site", mock.Anything).Return("", failed)
		},
		"a credential": func(m configureMocks) {
			choose("beyond", "BeyondHD (private torrent)")(m)
			m.ui.On("CreateInput", "Name it", mock.Anything).Return("", nil)
			m.ui.On("CreateSelect", "BeyondHD: Base Url", mock.Anything).Return("https://beyond-hd.me/", nil)
			m.ui.On("CreatePassword", mock.Anything).Return("", failed)
		},
	}

	for name, setup := range cases {
		m := newConfigureMocks(nil)
		setup(m)
		args := []string{"bbe-media"}
		if name == "choosing the package" {
			args = nil
		}

		err := m.run(args...)

		assert.EqualError(t, err, "Mock prompt cancelled", name)
		m.pkg.AssertNotCalled(t, "UpgradePackage", mock.Anything, mock.Anything, mock.Anything)
	}
}
