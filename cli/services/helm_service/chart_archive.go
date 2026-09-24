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
		if header.Typeflag != tar.TypeReg || !slices.Contains(chartFiles, path.Base(name)) {
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
		if prerequisite.StorageClass == "" && prerequisite.Talos == nil {
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
