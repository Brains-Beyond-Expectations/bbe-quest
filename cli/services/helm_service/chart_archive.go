package helm_service

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/constants"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"gopkg.in/yaml.v3"
)

// The only files bbe needs from a chart, and from each chart it depends on
var chartFiles = []string{"Chart.yaml", "values.yaml", "values.schema.json"}

// The levels of the pod security standards, see https://kubernetes.io/docs/concepts/security/pod-security-standards
var podSecurityLevels = []string{"privileged", "baseline", "restricted"}

func readChartArchive(archivePath string) (*models.HelmChart, error) {
	archive, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer archive.Close()

	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	files := map[string][]byte{}
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}

		name := path.Clean(header.Name)
		if header.Typeflag != tar.TypeReg || !(slices.Contains(chartFiles, path.Base(name)) || inCrdsFolder(name)) {
			continue
		}

		content, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, err
		}
		files[name] = content
	}

	for name := range files {
		if dir, file := path.Split(name); file == "Chart.yaml" && !strings.Contains(strings.TrimSuffix(dir, "/"), "/") {
			return parseChart(files, strings.TrimSuffix(dir, "/"))
		}
	}

	return nil, errors.New("Chart.yaml not found in the archive")
}

// Packaged charts keep the charts they depend on in `charts/<name>/`, which can nest further
func parseChart(files map[string][]byte, dir string) (*models.HelmChart, error) {
	var metadata models.HelmChartMetadata
	if err := yaml.Unmarshal(files[dir+"/Chart.yaml"], &metadata); err != nil {
		return nil, fmt.Errorf("Failed to parse %s/Chart.yaml: %w", dir, err)
	}

	chart := &models.HelmChart{Name: metadata.Name, Dependencies: metadata.Dependencies}

	prerequisites, err := parsePrerequisites(metadata.Annotations[constants.PrerequisitesAnnotation])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse the %s annotation in %s/Chart.yaml: %w", constants.PrerequisitesAnnotation, dir, err)
	}
	chart.Prerequisites = prerequisites

	podSecurity := metadata.Annotations[constants.PodSecurityAnnotation]
	if podSecurity != "" && !slices.Contains(podSecurityLevels, podSecurity) {
		return nil, fmt.Errorf("The %s annotation in %s/Chart.yaml is `%s`, but has to be one of %s", constants.PodSecurityAnnotation, dir, podSecurity, strings.Join(podSecurityLevels, ", "))
	}
	chart.PodSecurity = podSecurity

	settings, err := parseSettings(metadata.Annotations[constants.SettingsAnnotation])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse the %s annotation in %s/Chart.yaml: %w", constants.SettingsAnnotation, dir, err)
	}
	chart.Settings = settings

	uninstall, err := parseUninstall(metadata.Annotations[constants.UninstallAnnotation])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse the %s annotation in %s/Chart.yaml: %w", constants.UninstallAnnotation, dir, err)
	}
	chart.Uninstall = uninstall

	if content, found := files[dir+"/values.yaml"]; found {
		if err := yaml.Unmarshal(content, &chart.Values); err != nil {
			return nil, fmt.Errorf("Failed to parse %s/values.yaml: %w", dir, err)
		}
	}

	// Helm still validates the values if bbe can't read the schema, so carry on without it
	if content, found := files[dir+"/values.schema.json"]; found {
		var schema models.HelmChartSchema
		if err := json.Unmarshal(content, &schema); err != nil {
			logger.Debug(fmt.Sprintf("Ignoring %s/values.schema.json, it could not be parsed: %v", dir, err))
		} else {
			chart.Schema = &schema
		}
	}

	for _, name := range slices.Sorted(maps.Keys(files)) {
		if file, found := strings.CutPrefix(name, dir+"/crds/"); found && isManifest(file) {
			chart.Crds = append(chart.Crds, string(files[name]))
		}
	}

	subcharts := map[string]*models.HelmChart{}
	for _, subchartDir := range subchartDirs(files, dir) {
		subchart, err := parseChart(files, subchartDir)
		if err != nil {
			return nil, err
		}
		subcharts[subchart.Name] = subchart
	}

	declared := map[string]bool{}
	for i := range chart.Dependencies {
		chart.Dependencies[i].Chart = subcharts[chart.Dependencies[i].Name]
		declared[chart.Dependencies[i].Name] = true
	}

	// Helm also loads charts that are bundled without being listed as a dependency
	for _, name := range slices.Sorted(maps.Keys(subcharts)) {
		if !declared[name] {
			chart.Dependencies = append(chart.Dependencies, models.HelmChartDependency{Name: name, Chart: subcharts[name]})
		}
	}

	return chart, nil
}

// Helm reads every manifest under a chart's crds/ folder as a CRD
func inCrdsFolder(name string) bool {
	return slices.Contains(strings.Split(path.Dir(name), "/"), "crds") && isManifest(name)
}

func isManifest(name string) bool {
	return slices.Contains([]string{".yaml", ".yml", ".json"}, path.Ext(name))
}

func subchartDirs(files map[string][]byte, dir string) []string {
	var dirs []string
	prefix := dir + "/charts/"
	for name := range files {
		rest, found := strings.CutPrefix(name, prefix)
		if !found {
			continue
		}

		if subchart, file, _ := strings.Cut(rest, "/"); file == "Chart.yaml" {
			dirs = append(dirs, prefix+subchart)
		}
	}
	slices.Sort(dirs)

	return dirs
}

// Talos names user volumes with letters, digits and hyphens, and mounts them at /var/mnt/<name>
var userVolumePath = regexp.MustCompile(`^/var/mnt/[A-Za-z0-9-]+$`)

// A mistake in the annotation fails loudly, as installing without a prerequisite would fail later and less clearly
func parsePrerequisites(annotation string) ([]models.ChartPrerequisite, error) {
	if strings.TrimSpace(annotation) == "" {
		return nil, nil
	}

	var prerequisites []models.ChartPrerequisite
	if err := yaml.Unmarshal([]byte(annotation), &prerequisites); err != nil {
		return nil, err
	}

	for _, prerequisite := range prerequisites {
		if prerequisite.Name == "" {
			return nil, errors.New("every prerequisite needs a name")
		}
		if prerequisite.StorageClass == "" && prerequisite.Talos == nil && prerequisite.Package == "" {
			return nil, fmt.Errorf("prerequisite `%s` has nothing to check", prerequisite.Name)
		}
		if prerequisite.Talos != nil {
			for _, directory := range prerequisite.Talos.Directories {
				if !userVolumePath.MatchString(directory) {
					return nil, fmt.Errorf("prerequisite `%s` has directory `%s`, which has to be directly under /var/mnt", prerequisite.Name, directory)
				}
			}
		}
		if prerequisite.StorageClass != "" && prerequisite.Package == "" {
			return nil, fmt.Errorf("prerequisite `%s` needs a package that provides its storage class", prerequisite.Name)
		}
	}

	return prerequisites, nil
}

// The catalogs bbe knows how to read
var settingCatalogTypes = []string{"prowlarr-indexers"}

func parseSettings(annotation string) ([]models.ChartSetting, error) {
	if strings.TrimSpace(annotation) == "" {
		return nil, nil
	}

	var settings []models.ChartSetting
	if err := yaml.Unmarshal([]byte(annotation), &settings); err != nil {
		return nil, err
	}

	for _, setting := range settings {
		if setting.Name == "" || setting.Values == "" {
			return nil, errors.New("every setting needs a name and the path of its values")
		}
		if setting.Catalog == nil && len(setting.Fields) == 0 {
			return nil, fmt.Errorf("setting `%s` needs a catalog to add entries from, or the fields to ask for", setting.Name)
		}
		if setting.Catalog != nil && len(setting.Fields) > 0 {
			return nil, fmt.Errorf("setting `%s` has both a catalog and fields, but can only add entries from one", setting.Name)
		}
		for _, entry := range setting.Suggested {
			if label := setting.EntryLabel(entry); label == "" || label == "<nil>" {
				return nil, fmt.Errorf("every suggested entry of setting `%s` needs a name", setting.Name)
			}
			// They're written to bbe.yaml, with nothing in the setting's secret
			if _, secret := entry["secretFields"]; secret {
				return nil, fmt.Errorf("suggested entry `%s` of setting `%s` can't have credentials", setting.EntryLabel(entry), setting.Name)
			}
		}
		if len(setting.Fields) > 0 {
			if err := validateSettingFields(setting); err != nil {
				return nil, err
			}
			continue
		}
		if !slices.Contains(settingCatalogTypes, setting.Catalog.Type) {
			return nil, fmt.Errorf("setting `%s` has catalog type `%s`, which bbe doesn't know", setting.Name, setting.Catalog.Type)
		}
		if setting.Catalog.Service == "" || setting.Catalog.Port == 0 || setting.Catalog.ApiKey == nil || setting.Catalog.ApiKey.Secret == "" || setting.Catalog.ApiKey.Key == "" {
			return nil, fmt.Errorf("setting `%s` needs the catalog's service, port and API key", setting.Name)
		}
	}

	return settings, nil
}

func validateSettingFields(setting models.ChartSetting) error {
	unique := 0
	names := map[string]bool{}
	for _, field := range setting.Fields {
		if field.Name == "" {
			return fmt.Errorf("every field of setting `%s` needs a name", setting.Name)
		}
		if names[field.Name] {
			return fmt.Errorf("setting `%s` has two fields named `%s`", setting.Name, field.Name)
		}
		names[field.Name] = true
		if field.Unique {
			unique++
		}
		if _, err := regexp.Compile(field.Pattern); err != nil {
			return fmt.Errorf("the pattern of field `%s` in setting `%s` isn't a regular expression: %w", field.Name, setting.Name, err)
		}
		if field.Generate && !field.Secret {
			return fmt.Errorf("field `%s` in setting `%s` is generated, so it has to be secret", field.Name, setting.Name)
		}
		if field.Secret && setting.Secret == "" {
			return fmt.Errorf("field `%s` in setting `%s` is secret, but the setting has no secret to keep it in", field.Name, setting.Name)
		}
		if field.Unique && (field.Secret || field.Optional || len(field.Options) > 0) {
			return fmt.Errorf("field `%s` in setting `%s` names its entries, so it has to be typed in, required and not secret", field.Name, setting.Name)
		}
		for _, option := range field.Options {
			if option.Name == "" || option.Value == nil {
				return fmt.Errorf("every option of field `%s` in setting `%s` needs a name and a value", field.Name, setting.Name)
			}
		}
	}
	if unique != 1 {
		return fmt.Errorf("setting `%s` needs one unique field, which names its entries", setting.Name)
	}

	return nil
}

// A namespace name, which Kubernetes limits to a DNS label
var namespaceName = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

func parseUninstall(annotation string) (*models.ChartUninstall, error) {
	if strings.TrimSpace(annotation) == "" {
		return nil, nil
	}

	var uninstall models.ChartUninstall
	if err := yaml.Unmarshal([]byte(annotation), &uninstall); err != nil {
		return nil, err
	}

	for _, namespace := range uninstall.Namespaces {
		if !namespaceName.MatchString(namespace) {
			return nil, fmt.Errorf("`%s` isn't a namespace name", namespace)
		}
	}
	for _, patch := range uninstall.Patches {
		if patch.Resource == "" || patch.Name == "" || patch.Namespace == "" {
			return nil, errors.New("every patch needs a resource, a name and a namespace")
		}
		if !json.Valid([]byte(patch.Patch)) {
			return nil, fmt.Errorf("the patch to %s `%s` isn't JSON", patch.Resource, patch.Name)
		}
	}

	return &uninstall, nil
}
