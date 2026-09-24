package helm_service

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/models"
	"github.com/stretchr/testify/assert"
)

func Test_readChartArchive_Succeeds_WithNestedDependencies(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "parent-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"parent/Chart.yaml": `name: parent
dependencies:
  - name: child
    alias: renamed
    condition: renamed.enabled
    version: 1.0.0
  - name: not-bundled
`,
		"parent/values.yaml":                               "renamed:\n  enabled: true\n",
		"parent/values.schema.json":                        `{"required": ["bbe"], "properties": {"bbe": {"type": "string", "minLength": 1, "description": "A value"}}}`,
		"parent/templates/values.yaml":                     "ignored: true\n",
		"parent/charts/child/Chart.yaml":                   "name: child\n",
		"parent/charts/child/values.yaml":                  "port: 80\n",
		"parent/charts/child/charts/grandchild/Chart.yaml": "name: grandchild\n",
		"parent/charts/bundled/Chart.yaml":                 "name: bundled\n",
	})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Equal(t, "parent", chart.Name)
	assert.Equal(t, map[string]interface{}{"renamed": map[string]interface{}{"enabled": true}}, chart.Values)
	minLength := 1
	assert.Equal(t, &models.HelmChartSchema{
		Required:   []string{"bbe"},
		Properties: map[string]models.HelmChartSchema{"bbe": {Type: "string", MinLength: &minLength, Description: "A value"}},
	}, chart.Schema)

	assert.Len(t, chart.Dependencies, 3)
	child := chart.Dependencies[0]
	assert.Equal(t, "child", child.Name)
	assert.Equal(t, "renamed", child.Alias)
	assert.Equal(t, "renamed.enabled", child.Condition)
	assert.Equal(t, "child", child.Chart.Name)
	assert.Equal(t, map[string]interface{}{"port": 80}, child.Chart.Values)
	assert.Nil(t, child.Chart.Schema)
	assert.Len(t, child.Chart.Dependencies, 1)
	assert.Equal(t, "grandchild", child.Chart.Dependencies[0].Chart.Name)

	assert.Equal(t, "not-bundled", chart.Dependencies[1].Name)
	assert.Nil(t, chart.Dependencies[1].Chart)

	// Helm loads bundled charts even when they aren't listed as a dependency
	assert.Equal(t, "bundled", chart.Dependencies[2].Name)
	assert.Equal(t, "bundled", chart.Dependencies[2].Chart.Name)
}

func Test_readChartArchive_Succeeds_WithAliasedDependenciesSharingAChart(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "parent-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"parent/Chart.yaml":              "name: parent\ndependencies:\n  - name: child\n    alias: first\n  - name: child\n    alias: second\n",
		"parent/charts/child/Chart.yaml": "name: child\n",
	})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Len(t, chart.Dependencies, 2)
	assert.Equal(t, "child", chart.Dependencies[0].Chart.Name)
	assert.Equal(t, "child", chart.Dependencies[1].Chart.Name)
}

func Test_readChartArchive_Succeeds_IgnoringAnInvalidSchema(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "parent-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"parent/Chart.yaml":         "name: parent\n",
		"parent/values.schema.json": "not json",
	})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Nil(t, chart.Schema)
}

func Test_readChartArchive_Fails_WithoutChartYaml(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "parent-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"parent/values.yaml":             "replicas: 1\n",
		"parent/charts/child/Chart.yaml": "name: child\n",
	})

	chart, err := readChartArchive(archive)

	assert.Nil(t, chart)
	assert.EqualError(t, err, "Chart.yaml not found in the archive")
}

func Test_readChartArchive_Fails_WithInvalidChartYaml(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "parent-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"parent/Chart.yaml": "name: [parent\n",
	})

	chart, err := readChartArchive(archive)

	assert.Nil(t, chart)
	assert.ErrorContains(t, err, "Failed to parse parent/Chart.yaml")
}

func Test_readChartArchive_Fails_WithInvalidValues(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "parent-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"parent/Chart.yaml":  "name: parent\n",
		"parent/values.yaml": "replicas: [1\n",
	})

	chart, err := readChartArchive(archive)

	assert.Nil(t, chart)
	assert.ErrorContains(t, err, "Failed to parse parent/values.yaml")
}

func Test_readChartArchive_Fails_WithInvalidDependency(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "parent-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"parent/Chart.yaml":              "name: parent\n",
		"parent/charts/child/Chart.yaml": "name: [child\n",
	})

	chart, err := readChartArchive(archive)

	assert.Nil(t, chart)
	assert.ErrorContains(t, err, "Failed to parse parent/charts/child/Chart.yaml")
}

func Test_readChartArchive_Fails_WhenArchiveIsMissing(t *testing.T) {
	chart, err := readChartArchive(filepath.Join(t.TempDir(), "missing.tgz"))

	assert.Nil(t, chart)
	assert.Error(t, err)
}

func Test_readChartArchive_Fails_WhenArchiveIsCorrupt(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "parent-1.0.0.tgz")
	file, _ := os.Create(archive)
	gzipWriter := gzip.NewWriter(file)
	gzipWriter.Write([]byte("not a tar archive"))
	gzipWriter.Close()
	file.Close()

	chart, err := readChartArchive(archive)

	assert.Nil(t, chart)
	assert.Error(t, err)
}

// Writes a gzipped tar archive laid out like `helm pull` downloads one
func writeChartArchive(t *testing.T, archivePath string, files map[string]string) {
	t.Helper()

	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	tarWriter.WriteHeader(&tar.Header{Name: "chart/", Typeflag: tar.TypeDir, Mode: 0o755})
	for name, content := range files {
		tarWriter.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: int64(len(content))})
		tarWriter.Write([]byte(content))
	}
}
