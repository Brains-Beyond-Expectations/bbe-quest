package cmd

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/interfaces"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/catalog_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/config_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helm_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/helper_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/kubernetes_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/package_service"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/services/ui_service"
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure [package]",
	Short: "Add to or remove from an installed package's settings, such as the indexers of bbe-media",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		helperService := helper_service.HelperService{}
		uiService := ui_service.UiService{}
		configService := config_service.ConfigService{}
		packageService := package_service.PackageService{}
		helmService := helm_service.HelmService{}
		kubernetesService := kubernetes_service.KubernetesService{}
		catalogService := catalog_service.CatalogService{}

		err := configureCommand(args, helperService, uiService, configService, packageService, helmService, kubernetesService, catalogService)
		if err != nil {
			logger.Error("", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)
}

const (
	addAction    = "Add"
	removeAction = "Remove"
	doneAction   = "Done"
	searchAgain  = "Search again"
	cancelChoice = "Cancel"
)

// The most matches offered at once, so the list fits on screen
const maxMatches = 25

func configureCommand(args []string, helperService interfaces.HelperServiceInterface, uiService interfaces.UiServiceInterface, configService interfaces.ConfigServiceInterface, packageService interfaces.PackageServiceInterface, helmService interfaces.HelmServiceInterface, kubernetesService interfaces.KubernetesServiceInterface, catalogService interfaces.CatalogServiceInterface) error {
	bbeConfig, err := configService.GetBbeConfig(helperService)
	if err != nil || bbeConfig.Bbe.Cluster.Name == "" {
		return errors.New("No BBE cluster found, please run 'bbe setup' to create your cluster")
	}

	index, err := choosePackage(args, bbeConfig.Bbe.Packages, uiService)
	if err != nil {
		return err
	}
	installed := bbeConfig.Bbe.Packages[index]

	library, err := packageService.GetAll()
	if err != nil {
		return err
	}
	chartIndex := slices.IndexFunc(library, func(chart models.ChartEntry) bool { return chart.Name == installed.Name })
	if chartIndex < 0 {
		return fmt.Errorf("Package `%s` isn't in the package library", installed.Name)
	}
	// The installed version is configured, not a newer one in the library
	chart := library[chartIndex]
	chart.Version = installed.Version

	helmChart, err := helmService.PullChart(chart.RepositoryUrl, chart.Name, chart.Version)
	if err != nil {
		return err
	}
	if len(helmChart.Settings) == 0 {
		logger.Info(fmt.Sprintf("Package `%s` has nothing to configure", installed.Name))
		return nil
	}

	setting, err := chooseSetting(helmChart.Settings, uiService)
	if err != nil {
		return err
	}

	values := package_service.NormalizeValues(installed.Values)
	if values == nil {
		values = map[string]interface{}{}
	}
	editor := &settingEditor{
		uiService:         uiService,
		kubernetesService: kubernetesService,
		catalogService:    catalogService,
		context:           bbeConfig.Bbe.Cluster.Context,
		namespace:         installed.Name,
		setting:           setting,
		entries:           settingEntries(values, setting.Values),
		secrets:           map[string]string{},
	}
	if err := editor.run(); err != nil {
		return err
	}
	if !editor.changed {
		logger.Info("Nothing changed")
		return nil
	}

	// The credentials go first, so the package finds them when it's upgraded
	if len(editor.secrets) > 0 {
		if err := kubernetesService.SetSecretValues(editor.context, editor.namespace, setting.Secret, editor.secrets); err != nil {
			return err
		}
	}

	setSettingEntries(values, setting.Values, editor.entries)
	upgradedValues, err := packageService.UpgradePackage(chart, values, *bbeConfig, helmService, uiService, true)
	if err != nil {
		return err
	}
	bbeConfig.Bbe.Packages[index].Values = upgradedValues
	if err := configService.UpdateBbePackages(helperService, bbeConfig.Bbe.Packages); err != nil {
		return fmt.Errorf("Failed to update BBE configuration: %w", err)
	}

	if len(editor.removedKeys) > 0 {
		if err := kubernetesService.RemoveSecretValues(editor.context, editor.namespace, setting.Secret, editor.removedKeys); err != nil {
			return err
		}
	}

	logger.Info(fmt.Sprintf("Package `%s` is updated, and applies the change to its %s in the background", installed.Name, setting.Name))
	for _, generated := range editor.generated {
		logger.Info(fmt.Sprintf("%s: %s is `%s`, which bbe only shows now. It stays in the `%s` secret as `%s`", generated.entry, generated.label, generated.value, setting.Secret, generated.key))
	}
	return nil
}

func choosePackage(args []string, packages []models.LocalPackage, uiService interfaces.UiServiceInterface) (int, error) {
	if len(packages) == 0 {
		return 0, errors.New("No packages are installed, install one with 'bbe install'")
	}

	name := ""
	if len(args) == 1 {
		name = args[0]
	} else {
		names := []string{}
		for _, pkg := range packages {
			names = append(names, pkg.Name)
		}
		chosen, err := uiService.CreateSelect("Which package would you like to configure?", names)
		if err != nil {
			return 0, err
		}
		name = chosen
	}

	index := slices.IndexFunc(packages, func(pkg models.LocalPackage) bool { return pkg.Name == name })
	if index < 0 {
		return 0, fmt.Errorf("Package `%s` isn't installed, install it with 'bbe install'", name)
	}

	return index, nil
}

func chooseSetting(settings []models.ChartSetting, uiService interfaces.UiServiceInterface) (models.ChartSetting, error) {
	if len(settings) == 1 {
		return settings[0], nil
	}

	options := []string{}
	for _, setting := range settings {
		options = append(options, fmt.Sprintf("%s: %s", setting.Name, setting.Description))
	}
	chosen, err := uiService.CreateSelect("Which setting would you like to configure?", options)
	if err != nil {
		return models.ChartSetting{}, err
	}

	return settings[slices.Index(options, chosen)], nil
}

// Adds to and removes from one setting's entries, which are only saved once the user is done
type settingEditor struct {
	uiService         interfaces.UiServiceInterface
	kubernetesService interfaces.KubernetesServiceInterface
	catalogService    interfaces.CatalogServiceInterface
	context           string
	namespace         string
	setting           models.ChartSetting
	entries           []map[string]interface{}
	definitions       []models.IndexerDefinition
	// Credentials of added entries to write, and keys of removed ones to delete, from the setting's secret
	secrets     map[string]string
	removedKeys []string
	changed     bool
	// Values bbe generated, shown once the change is saved
	generated []generatedValue
}

type generatedValue struct {
	entry string
	label string
	value string
	key   string
}

func (editor *settingEditor) run() error {
	for {
		editor.list()

		actions := []string{addAction}
		if len(editor.entries) > 0 {
			actions = append(actions, removeAction)
		}
		actions = append(actions, doneAction)

		action, err := editor.uiService.CreateSelect(fmt.Sprintf("What would you like to do with the %s?", editor.setting.Name), actions)
		if err != nil {
			return err
		}

		switch action {
		case addAction:
			err = editor.add()
		case removeAction:
			err = editor.remove()
		default:
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func (editor *settingEditor) list() {
	if len(editor.entries) == 0 {
		logger.Info(fmt.Sprintf("No %s yet", editor.setting.Name))
		return
	}

	lines := []string{}
	for _, entry := range editor.entries {
		lines = append(lines, "  "+editor.describe(entry))
	}
	logger.Info(fmt.Sprintf("The %s:\n%s", editor.setting.Name, strings.Join(lines, "\n")))
}

func (editor *settingEditor) add() error {
	if editor.setting.Catalog == nil {
		return editor.addFields()
	}

	definitions, err := editor.catalog()
	if err != nil {
		return err
	}

	definition, err := editor.search(definitions)
	if err != nil || definition == nil {
		return err
	}
	if slices.ContainsFunc(definition.Fields, func(field models.IndexerField) bool { return field.Type == "cardigannCaptcha" }) {
		logger.Warning(fmt.Sprintf("%s needs a captcha solved to sign in, so add it in Prowlarr's web UI instead", definition.Name))
		return nil
	}

	name, err := editor.entryName(definition.Name)
	if err != nil {
		return err
	}

	fields := map[string]interface{}{}
	secretFields := map[string]interface{}{}
	for _, field := range definition.Fields {
		if !promptable(field) {
			continue
		}

		value, set, err := editor.promptField(name, *definition, field)
		if err != nil {
			return err
		}
		if !set {
			continue
		}

		if isCredential(field) {
			key := secretKey(name, field.Name)
			editor.secrets[key] = fmt.Sprint(value)
			editor.removedKeys = slices.DeleteFunc(editor.removedKeys, func(removed string) bool { return removed == key })
			secretFields[field.Name] = key
		} else {
			fields[field.Name] = value
		}
	}

	entry := map[string]interface{}{"name": name, "definition": definition.DefinitionName}
	if len(fields) > 0 {
		entry["fields"] = fields
	}
	if len(secretFields) > 0 {
		entry["secretFields"] = secretFields
	}
	editor.entries = append(editor.entries, entry)
	editor.changed = true
	logger.Info(fmt.Sprintf("Added %s", name))

	return nil
}

func (editor *settingEditor) remove() error {
	names := []string{}
	for _, entry := range editor.entries {
		names = append(names, editor.entryLabel(entry))
	}

	chosen, err := editor.uiService.CreateSelect(fmt.Sprintf("Which of the %s would you like to remove?", editor.setting.Name), append(names, cancelChoice))
	if err != nil || chosen == cancelChoice {
		return err
	}

	index := slices.Index(names, chosen)
	if secretFields, ok := editor.entries[index]["secretFields"].(map[string]interface{}); ok {
		for _, key := range secretFields {
			editor.removedKeys = append(editor.removedKeys, fmt.Sprint(key))
			delete(editor.secrets, fmt.Sprint(key))
		}
	}
	editor.entries = slices.Delete(editor.entries, index, index+1)
	editor.changed = true
	logger.Info(fmt.Sprintf("Removed %s", chosen))

	return nil
}

// What names an entry: its unique field, or the name it was given when added from a catalog
func (editor *settingEditor) entryLabel(entry map[string]interface{}) string {
	for _, field := range editor.setting.Fields {
		if field.Unique {
			return fmt.Sprint(entry[field.Name])
		}
	}
	return fmt.Sprint(entry["name"])
}

// An entry as it's listed, with what it was added with besides its credentials
func (editor *settingEditor) describe(entry map[string]interface{}) string {
	if editor.setting.Catalog != nil {
		return fmt.Sprintf("%v (%v)", entry["name"], entry["definition"])
	}

	details := []string{}
	for _, field := range editor.setting.Fields {
		value, set := entry[field.Name]
		if field.Unique || field.Secret || !set {
			continue
		}
		if list, isList := value.([]interface{}); isList {
			parts := []string{}
			for _, item := range list {
				parts = append(parts, fmt.Sprint(item))
			}
			value = strings.Join(parts, ", ")
		}
		details = append(details, fmt.Sprint(value))
	}
	if len(details) == 0 {
		return editor.entryLabel(entry)
	}
	return fmt.Sprintf("%s (%s)", editor.entryLabel(entry), strings.Join(details, ", "))
}

// Asks for each of the setting's fields, and generates the ones it says to
func (editor *settingEditor) addFields() error {
	values := map[string]interface{}{}
	label := ""
	for _, field := range editor.setting.Fields {
		value, set, err := editor.askField(field)
		if err != nil {
			return err
		}
		if !set {
			continue
		}
		values[field.Name] = value
		if field.Unique {
			label = fmt.Sprint(value)
		}
	}

	entry := map[string]interface{}{}
	secretFields := map[string]interface{}{}
	for _, field := range editor.setting.Fields {
		value, set := values[field.Name]
		if !set {
			continue
		}
		if !field.Secret {
			entry[field.Name] = value
			continue
		}

		key := secretKey(label, field.Name)
		editor.secrets[key] = fmt.Sprint(value)
		editor.removedKeys = slices.DeleteFunc(editor.removedKeys, func(removed string) bool { return removed == key })
		secretFields[field.Name] = key
		if field.Generate {
			editor.generated = append(editor.generated, generatedValue{entry: label, label: fieldLabel(field), value: fmt.Sprint(value), key: key})
		}
	}
	if len(secretFields) > 0 {
		entry["secretFields"] = secretFields
	}

	editor.entries = append(editor.entries, entry)
	editor.changed = true
	logger.Info(fmt.Sprintf("Added %s", label))

	return nil
}

// Asks for one field until it's answered as the field requires, returning whether it has a value
func (editor *settingEditor) askField(field models.SettingField) (interface{}, bool, error) {
	if field.Generate {
		value, err := generatePassword()
		return value, err == nil, err
	}

	title := fieldLabel(field)
	if field.Description != "" {
		title = fmt.Sprintf("%s (%s)", title, field.Description)
	}

	if len(field.Options) > 0 {
		names := []string{}
		for _, option := range field.Options {
			names = append(names, option.Name)
		}
		chosen, err := editor.uiService.CreateSelect(title, names)
		if err != nil {
			return nil, false, err
		}
		return field.Options[slices.Index(names, chosen)].Value, true, nil
	}

	pattern := regexp.MustCompile(field.Pattern)
	for {
		var input string
		var err error
		if field.Secret {
			input, err = editor.uiService.CreatePassword(title)
		} else {
			input, err = editor.uiService.CreateInput(title, "")
		}
		if err != nil {
			return nil, false, err
		}
		input = strings.TrimSpace(input)

		switch {
		case input == "" && field.Optional:
			return nil, false, nil
		case input == "":
			logger.Warning(fmt.Sprintf("%s is required", fieldLabel(field)))
		case !pattern.MatchString(input):
			logger.Warning(fmt.Sprintf("`%s` isn't a valid %s, please try again", input, strings.ToLower(fieldLabel(field))))
		case field.Unique && slices.ContainsFunc(editor.entries, func(entry map[string]interface{}) bool { return fmt.Sprint(entry[field.Name]) == input }):
			logger.Warning(fmt.Sprintf("There's already one with %s `%s`", strings.ToLower(fieldLabel(field)), input))
		default:
			return input, true, nil
		}
	}
}

func fieldLabel(field models.SettingField) string {
	if field.Label != "" {
		return field.Label
	}
	return field.Name
}

// Letters and digits, without those easily mistaken for one another: I, O, l, o, 0 and 1
var passwordCharacters = func() string {
	characters := []byte{}
	for _, span := range [][2]byte{{'A', 'Z'}, {'a', 'z'}, {'2', '9'}} {
		for character := span[0]; character <= span[1]; character++ {
			if !strings.ContainsRune("IOlo", rune(character)) {
				characters = append(characters, character)
			}
		}
	}
	return string(characters)
}()

// A random password, without characters easily mistaken for one another, as it may be read out to someone
var generatePassword = func() (string, error) {
	password := make([]byte, 20)
	for i := range password {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(passwordCharacters))))
		if err != nil {
			return "", fmt.Errorf("Failed to generate a password: %w", err)
		}
		password[i] = passwordCharacters[index.Int64()]
	}
	return string(password), nil
}

// Reads the catalog once, through a port forwarded to its service for as long as that takes
func (editor *settingEditor) catalog() ([]models.IndexerDefinition, error) {
	if editor.definitions != nil {
		return editor.definitions, nil
	}

	catalog := editor.setting.Catalog
	apiKey, err := editor.kubernetesService.GetSecretValue(editor.context, editor.namespace, catalog.ApiKey.Secret, catalog.ApiKey.Key)
	if err != nil {
		return nil, err
	}

	port, stop, err := editor.kubernetesService.PortForward(editor.context, editor.namespace, catalog.Service, catalog.Port)
	if err != nil {
		return nil, err
	}
	defer stop()

	definitions, err := editor.catalogService.IndexerDefinitions(fmt.Sprintf("http://127.0.0.1:%d", port), apiKey)
	if err != nil {
		return nil, err
	}
	editor.definitions = definitions

	return definitions, nil
}

// Returns the definition the user picks from a search, or nil when they cancel
func (editor *settingEditor) search(definitions []models.IndexerDefinition) (*models.IndexerDefinition, error) {
	for {
		query, err := editor.uiService.CreateInput("Search Prowlarr's indexers by name (empty to cancel)", "")
		if err != nil {
			return nil, err
		}
		query = strings.ToLower(strings.TrimSpace(query))
		if query == "" {
			return nil, nil
		}

		matches := []models.IndexerDefinition{}
		for _, definition := range definitions {
			if strings.Contains(strings.ToLower(definition.Name), query) || strings.Contains(strings.ToLower(definition.DefinitionName), query) {
				matches = append(matches, definition)
			}
		}
		if len(matches) == 0 {
			logger.Warning(fmt.Sprintf("No indexer matches `%s`", query))
			continue
		}
		slices.SortFunc(matches, func(a, b models.IndexerDefinition) int {
			return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
		})

		title := fmt.Sprintf("%d indexers match", len(matches))
		if len(matches) > maxMatches {
			title = fmt.Sprintf("%d indexers match, the first %d are shown", len(matches), maxMatches)
			matches = matches[:maxMatches]
		}
		options := []string{}
		for _, definition := range matches {
			options = append(options, fmt.Sprintf("%s (%s %s)", definition.Name, definition.Privacy, definition.Protocol))
		}

		chosen, err := editor.uiService.CreateSelect(title, append(options, searchAgain))
		if err != nil {
			return nil, err
		}
		if index := slices.Index(options, chosen); index >= 0 {
			return &matches[index], nil
		}
	}
}

// Asks for a name no other entry has, suggesting the definition's
func (editor *settingEditor) entryName(suggestion string) (string, error) {
	for {
		name, err := editor.uiService.CreateInput("Name it", suggestion)
		if err != nil {
			return "", err
		}
		name = strings.TrimSpace(name)
		if name == "" {
			name = suggestion
		}

		taken := slices.ContainsFunc(editor.entries, func(entry map[string]interface{}) bool { return fmt.Sprint(entry["name"]) == name })
		if !taken {
			return name, nil
		}
		logger.Warning(fmt.Sprintf("There's already one named `%s`", name))
	}
}

// Asks for a field's value, returning whether it differs from the definition's default
func (editor *settingEditor) promptField(name string, definition models.IndexerDefinition, field models.IndexerField) (interface{}, bool, error) {
	title := fmt.Sprintf("%s: %s", name, field.Label)

	if field.Name == "baseUrl" && len(definition.IndexerUrls) > 0 {
		if len(definition.IndexerUrls) == 1 {
			return definition.IndexerUrls[0], true, nil
		}
		url, err := editor.uiService.CreateSelect(title, definition.IndexerUrls)
		return url, err == nil, err
	}

	if isCredential(field) {
		value, err := editor.uiService.CreatePassword(title)
		return value, err == nil && value != "", err
	}

	switch field.Type {
	case "checkbox":
		current, _ := field.Value.(bool)
		options := []string{"No", "Yes"}
		if current {
			options = []string{"Yes", "No"}
		}
		chosen, err := editor.uiService.CreateSelect(title, options)
		return chosen == "Yes", err == nil && (chosen == "Yes") != current, err

	case "select":
		options := []string{}
		currentIndex := -1
		for i, option := range field.SelectOptions {
			options = append(options, option.Name)
			if fmt.Sprint(option.Value) == fmt.Sprint(field.Value) {
				currentIndex = i
			}
		}
		// The current option first, so accepting the prompt keeps it
		if currentIndex > 0 {
			options = append([]string{options[currentIndex]}, slices.Delete(slices.Clone(options), currentIndex, currentIndex+1)...)
		}
		chosen, err := editor.uiService.CreateSelect(title, options)
		if err != nil {
			return nil, false, err
		}
		option := field.SelectOptions[slices.IndexFunc(field.SelectOptions, func(option models.IndexerFieldOption) bool { return option.Name == chosen })]
		return wholeNumber(option.Value), fmt.Sprint(option.Value) != fmt.Sprint(field.Value), nil

	case "number":
		for {
			input, err := editor.uiService.CreateInput(title, defaultText(field.Value))
			if err != nil {
				return nil, false, err
			}
			input = strings.TrimSpace(input)
			if input == "" || input == defaultText(field.Value) {
				return nil, false, nil
			}
			if number, err := strconv.ParseFloat(input, 64); err == nil {
				return wholeNumber(number), true, nil
			}
			logger.Warning(fmt.Sprintf("`%s` isn't a number, please try again", input))
		}

	default:
		input, err := editor.uiService.CreateInput(title, defaultText(field.Value))
		input = strings.TrimSpace(input)
		return input, err == nil && input != "" && input != defaultText(field.Value), err
	}
}

// Fields a user sets when adding an indexer in Prowlarr: advanced ones keep Prowlarr's defaults, info fields are
// only text, unlabelled ones are internal, and multi-select ones are left to Prowlarr's web UI
func promptable(field models.IndexerField) bool {
	if field.Advanced || field.Label == "" || field.Type == "info" {
		return false
	}
	_, isList := field.Value.([]interface{})
	return !isList
}

func isCredential(field models.IndexerField) bool {
	return (field.Privacy != "" && field.Privacy != "normal") || field.Type == "password"
}

var notKeyCharacters = regexp.MustCompile(`[^a-z0-9]+`)

// A secret key for an entry's field, such as `beyondhd.apiKey`
func secretKey(name string, field string) string {
	return strings.Trim(notKeyCharacters.ReplaceAllString(strings.ToLower(name), "-"), "-") + "." + field
}

func defaultText(value interface{}) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

// JSON numbers are floats, so whole ones are kept as integers in bbe.yaml
func wholeNumber(value interface{}) interface{} {
	if number, ok := value.(float64); ok && number == float64(int64(number)) {
		return int64(number)
	}
	return value
}

// The entries of a setting's list in the package's values
func settingEntries(values map[string]interface{}, path string) []map[string]interface{} {
	var current interface{} = values
	for _, key := range strings.Split(path, ".") {
		nested, isMap := current.(map[string]interface{})
		if !isMap {
			return nil
		}
		current = nested[key]
	}

	entries := []map[string]interface{}{}
	list, _ := current.([]interface{})
	for _, item := range list {
		if entry, isMap := item.(map[string]interface{}); isMap {
			entries = append(entries, entry)
		}
	}

	return entries
}

func setSettingEntries(values map[string]interface{}, path string, entries []map[string]interface{}) {
	keys := strings.Split(path, ".")
	for _, key := range keys[:len(keys)-1] {
		nested, isMap := values[key].(map[string]interface{})
		if !isMap {
			nested = map[string]interface{}{}
			values[key] = nested
		}
		values = nested
	}

	list := make([]interface{}, len(entries))
	for i, entry := range entries {
		list[i] = entry
	}
	values[keys[len(keys)-1]] = list
}
