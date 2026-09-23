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
	"slices"
	"strings"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"gopkg.in/yaml.v3"
)

// The only files bbe needs from a chart, and from each chart it depends on
var chartFiles = []string{"Chart.yaml", "values.yaml", "values.schema.json"}

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
