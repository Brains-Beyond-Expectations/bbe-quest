package helm_service

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
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

func Test_readChartArchive_Succeeds_ReadingPrerequisites(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bbe-media-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"bbe-media/Chart.yaml": `name: bbe-media
annotations:
  bbe/prerequisites: |
    - name: storage
      description: Longhorn
      storageClass: longhorn
      package: bbe-storage
    - name: longhorn-nodes
      talos:
        extensions:
          - siderolabs/util-linux-tools
        directories:
          - /var/mnt/longhorn
`,
	})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Equal(t, []models.ChartPrerequisite{
		{Name: "storage", Description: "Longhorn", StorageClass: "longhorn", Package: "bbe-storage"},
		{Name: "longhorn-nodes", Talos: &models.TalosPrerequisite{Extensions: []string{"siderolabs/util-linux-tools"}, Directories: []string{"/var/mnt/longhorn"}}},
	}, chart.Prerequisites)
}

func Test_readChartArchive_Fails_WithInvalidPrerequisites(t *testing.T) {
	cases := map[string]string{
		"- name: [storage":                    "Failed to parse the bbe/prerequisites annotation in bbe-media/Chart.yaml",
		"- description: no name":              "every prerequisite needs a name",
		"- name: storage":                     "prerequisite `storage` has nothing to check",
		"- name: storage\n  storageClass: lh": "prerequisite `storage` needs a package that provides its storage class",
		"- name: nodes\n  talos:\n    directories: [/var/lib/longhorn]": "prerequisite `nodes` has directory `/var/lib/longhorn`, which has to be directly under /var/mnt",
	}

	for annotation, expected := range cases {
		archive := filepath.Join(t.TempDir(), "bbe-media-1.0.0.tgz")
		indented := "    " + strings.ReplaceAll(annotation, "\n", "\n    ")
		writeChartArchive(t, archive, map[string]string{"bbe-media/Chart.yaml": "name: bbe-media\nannotations:\n  bbe/prerequisites: |\n" + indented + "\n"})

		chart, err := readChartArchive(archive)

		assert.Nil(t, chart, annotation)
		assert.ErrorContains(t, err, expected, annotation)
	}
}

func Test_readChartArchive_Succeeds_ReadingThePodSecurityLevel(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bbe-networking-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{"bbe-networking/Chart.yaml": "name: bbe-networking\nannotations:\n  bbe/pod-security: privileged\n"})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Equal(t, "privileged", chart.PodSecurity)
}

func Test_readChartArchive_Fails_WithAnUnknownPodSecurityLevel(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bbe-networking-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{"bbe-networking/Chart.yaml": "name: bbe-networking\nannotations:\n  bbe/pod-security: host\n"})

	chart, err := readChartArchive(archive)

	assert.Nil(t, chart)
	assert.EqualError(t, err, "The bbe/pod-security annotation in bbe-networking/Chart.yaml is `host`, but has to be one of privileged, baseline, restricted")
}

func Test_readChartArchive_Succeeds_ReadingSettings(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bbe-media-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"bbe-media/Chart.yaml": `name: bbe-media
annotations:
  bbe/settings: |
    - name: indexers
      description: The indexers Prowlarr searches
      values: bbe.media.prowlarr.indexers
      secret: bbe-media-indexers
      catalog:
        type: prowlarr-indexers
        service: bbe-media-prowlarr
        port: 9696
        apiKey:
          secret: bbe-media-api-keys
          key: prowlarr
`,
	})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Equal(t, []models.ChartSetting{{
		Name:        "indexers",
		Description: "The indexers Prowlarr searches",
		Values:      "bbe.media.prowlarr.indexers",
		Secret:      "bbe-media-indexers",
		Catalog: &models.SettingCatalog{
			Type:    "prowlarr-indexers",
			Service: "bbe-media-prowlarr",
			Port:    9696,
			ApiKey:  &models.SecretKeyRef{Secret: "bbe-media-api-keys", Key: "prowlarr"},
		},
	}}, chart.Settings)
}

func Test_readChartArchive_Fails_WithInvalidSettings(t *testing.T) {
	catalog := "\n  catalog: {type: prowlarr-indexers, service: prowlarr, port: 9696, apiKey: {secret: keys, key: prowlarr}}"
	cases := map[string]string{
		"- name: [indexers":                                                     "Failed to parse the bbe/settings annotation in bbe-media/Chart.yaml",
		"- values: a.b" + catalog:                                               "every setting needs a name and the path of its values",
		"- name: indexers\n  values: a.b":                                       "setting `indexers` needs a catalog to add entries from",
		"- name: indexers\n  values: a.b\n  catalog: {type: shop}":              "setting `indexers` has catalog type `shop`, which bbe doesn't know",
		"- name: indexers\n  values: a.b\n  catalog: {type: prowlarr-indexers}": "setting `indexers` needs the catalog's service, port and API key",
	}

	for annotation, expected := range cases {
		archive := filepath.Join(t.TempDir(), "bbe-media-1.0.0.tgz")
		indented := "    " + strings.ReplaceAll(annotation, "\n", "\n    ")
		writeChartArchive(t, archive, map[string]string{"bbe-media/Chart.yaml": "name: bbe-media\nannotations:\n  bbe/settings: |\n" + indented + "\n"})

		chart, err := readChartArchive(archive)

		assert.Nil(t, chart, annotation)
		assert.ErrorContains(t, err, expected, annotation)
	}
}

func Test_readChartArchive_Succeeds_ReadingWhatUninstallingDeletes(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bbe-storage-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"bbe-storage/Chart.yaml": `name: bbe-storage
annotations:
  bbe/uninstall: |
    namespaces:
      - longhorn-system
    patches:
      - resource: settings.longhorn.io
        namespace: longhorn-system
        name: deleting-confirmation-flag
        patch: '{"value": "true"}'
    keeps:
      - Longhorn's directory on each node
`,
	})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Equal(t, &models.ChartUninstall{
		Namespaces: []string{"longhorn-system"},
		Patches:    []models.UninstallPatch{{Resource: "settings.longhorn.io", Namespace: "longhorn-system", Name: "deleting-confirmation-flag", Patch: `{"value": "true"}`}},
		Keeps:      []string{"Longhorn's directory on each node"},
	}, chart.Uninstall)
}

func Test_readChartArchive_Fails_WithAnInvalidUninstall(t *testing.T) {
	cases := map[string]string{
		"namespaces: [longhorn":                                  "Failed to parse the bbe/uninstall annotation in bbe-storage/Chart.yaml",
		"namespaces: [Longhorn_System]":                          "`Longhorn_System` isn't a namespace name",
		"patches: [{resource: settings.longhorn.io, name: flag}]": "every patch needs a resource, a name and a namespace",
		"patches: [{resource: settings, name: flag, namespace: longhorn-system, patch: 'value: true'}]": "the patch to settings `flag` isn't JSON",
	}

	for annotation, expected := range cases {
		archive := filepath.Join(t.TempDir(), "bbe-storage-1.0.0.tgz")
		writeChartArchive(t, archive, map[string]string{"bbe-storage/Chart.yaml": "name: bbe-storage\nannotations:\n  bbe/uninstall: |\n    " + annotation + "\n"})

		chart, err := readChartArchive(archive)

		assert.Nil(t, chart, annotation)
		assert.ErrorContains(t, err, expected, annotation)
	}
}

func Test_readChartArchive_Succeeds_ReadingTheCrdsOfEachChart(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "parent-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"parent/Chart.yaml":                      "name: parent\n",
		"parent/crds/b.yaml":                     "kind: CustomResourceDefinition # b\n",
		"parent/crds/a.json":                     `{"kind": "CustomResourceDefinition"}`,
		"parent/crds/nested/c.yml":               "kind: CustomResourceDefinition # c\n",
		"parent/crds/README.md":                  "Not a manifest\n",
		"parent/templates/crds.yaml":             "kind: CustomResourceDefinition # a template, which helm upgrades itself\n",
		"parent/charts/child/Chart.yaml":         "name: child\n",
		"parent/charts/child/crds/child.yaml":    "kind: CustomResourceDefinition # child\n",
		"parent/charts/crds/Chart.yaml":          "name: crds\n",
		"parent/charts/crds/templates/crds.yaml": "kind: CustomResourceDefinition # a subchart named crds, with templates\n",
	})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Equal(t, []string{`{"kind": "CustomResourceDefinition"}`, "kind: CustomResourceDefinition # b\n", "kind: CustomResourceDefinition # c\n"}, chart.Crds)
	assert.Equal(t, "child", chart.Dependencies[0].Chart.Name)
	assert.Equal(t, []string{"kind: CustomResourceDefinition # child\n"}, chart.Dependencies[0].Chart.Crds)
	assert.Equal(t, "crds", chart.Dependencies[1].Chart.Name)
	assert.Empty(t, chart.Dependencies[1].Chart.Crds)
}

func Test_readChartArchive_Succeeds_ReadingAPrerequisiteThatIsOnlyAPackage(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bbe-media-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{"bbe-media/Chart.yaml": "name: bbe-media\nannotations:\n  bbe/prerequisites: |\n    - name: sso\n      package: bbe-auth\n"})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Equal(t, []models.ChartPrerequisite{{Name: "sso", Package: "bbe-auth"}}, chart.Prerequisites)
}

func Test_readChartArchive_Succeeds_ReadingSettingsWithFields(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bbe-auth-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{
		"bbe-auth/Chart.yaml": `name: bbe-auth
annotations:
  bbe/settings: |
    - name: users
      values: bbe.auth.users
      secret: bbe-auth-users
      fields:
        - name: username
          label: Username
          pattern: "^[a-z]+$"
          unique: true
        - name: firstName
          optional: true
        - name: groups
          options:
            - name: users
              value: [users]
        - name: password
          generate: true
          secret: true
`,
	})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Equal(t, []models.SettingField{
		{Name: "username", Label: "Username", Pattern: "^[a-z]+$", Unique: true},
		{Name: "firstName", Optional: true},
		{Name: "groups", Options: []models.SettingFieldOption{{Name: "users", Value: []interface{}{"users"}}}},
		{Name: "password", Generate: true, Secret: true},
	}, chart.Settings[0].Fields)
	assert.Nil(t, chart.Settings[0].Catalog)
}

func Test_readChartArchive_Fails_WithInvalidSettingFields(t *testing.T) {
	setting := "- name: users\n  values: a.b\n  secret: users\n  fields:\n"
	unique := "  - {name: username, unique: true}\n"
	cases := map[string]string{
		"- name: users\n  values: a.b\n  catalog: {type: prowlarr-indexers}\n  fields: [{name: username, unique: true}]": "setting `users` has both a catalog and fields, but can only add entries from one",
		setting + "  - {label: Username, unique: true}\n":                                                                "every field of setting `users` needs a name",
		setting + unique + unique:                                                                     "setting `users` has two fields named `username`",
		setting + "  - {name: email}\n":                                                               "setting `users` needs one unique field, which names its entries",
		setting + unique + "  - {name: email, unique: true}\n":                                        "setting `users` needs one unique field, which names its entries",
		setting + unique + "  - {name: email, pattern: '[a-'}\n":                                      "the pattern of field `email` in setting `users` isn't a regular expression",
		setting + unique + "  - {name: password, generate: true}\n":                                   "field `password` in setting `users` is generated, so it has to be secret",
		"- name: users\n  values: a.b\n  fields:\n" + unique + "  - {name: password, secret: true}\n": "field `password` in setting `users` is secret, but the setting has no secret to keep it in",
		setting + "  - {name: username, unique: true, optional: true}\n":                              "field `username` in setting `users` names its entries, so it has to be typed in, required and not secret",
		setting + unique + "  - {name: groups, options: [{name: users}]}\n":                           "every option of field `groups` in setting `users` needs a name and a value",
	}

	for annotation, expected := range cases {
		archive := filepath.Join(t.TempDir(), "bbe-auth-1.0.0.tgz")
		indented := "    " + strings.ReplaceAll(strings.TrimSuffix(annotation, "\n"), "\n", "\n    ")
		writeChartArchive(t, archive, map[string]string{"bbe-auth/Chart.yaml": "name: bbe-auth\nannotations:\n  bbe/settings: |\n" + indented + "\n"})

		chart, err := readChartArchive(archive)

		assert.Nil(t, chart, annotation)
		assert.ErrorContains(t, err, expected, annotation)
	}
}

func Test_readChartArchive_Succeeds_ReadingSuggestedEntries(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "bbe-media-1.0.0.tgz")
	writeChartArchive(t, archive, map[string]string{"bbe-media/Chart.yaml": "name: bbe-media\nannotations:\n  bbe/settings: |\n    - name: indexers\n      values: a.b\n      catalog: {type: prowlarr-indexers, service: prowlarr, port: 9696, apiKey: {secret: keys, key: prowlarr}}\n      suggested:\n        - {name: 1337x, definition: 1337x}\n"})

	chart, err := readChartArchive(archive)

	assert.NoError(t, err)
	assert.Equal(t, []map[string]interface{}{{"name": "1337x", "definition": "1337x"}}, chart.Settings[0].Suggested)
}

func Test_readChartArchive_Fails_WithInvalidSuggestedEntries(t *testing.T) {
	catalog := "\n  catalog: {type: prowlarr-indexers, service: prowlarr, port: 9696, apiKey: {secret: keys, key: prowlarr}}"
	cases := map[string]string{
		"- name: indexers\n  values: a.b" + catalog + "\n  suggested: [{definition: 1337x}]":                         "every suggested entry of setting `indexers` needs a name",
		"- name: indexers\n  values: a.b" + catalog + "\n  suggested: [{name: x, secretFields: {apiKey: x.apiKey}}]": "suggested entry `x` of setting `indexers` can't have credentials",
	}

	for annotation, expected := range cases {
		archive := filepath.Join(t.TempDir(), "bbe-media-1.0.0.tgz")
		indented := "    " + strings.ReplaceAll(annotation, "\n", "\n    ")
		writeChartArchive(t, archive, map[string]string{"bbe-media/Chart.yaml": "name: bbe-media\nannotations:\n  bbe/settings: |\n" + indented + "\n"})

		chart, err := readChartArchive(archive)

		assert.Nil(t, chart, annotation)
		assert.ErrorContains(t, err, expected, annotation)
	}
}
