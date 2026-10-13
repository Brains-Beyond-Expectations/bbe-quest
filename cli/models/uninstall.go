package models

// What else bbe deletes when uninstalling a chart, besides its release and its namespace, declared in its Chart.yaml
// under the `bbe/uninstall` annotation
type ChartUninstall struct {
	// Other namespaces the chart keeps its resources in, such as Longhorn's `longhorn-system`
	Namespaces []string `yaml:"namespaces"`
	// Changes made before uninstalling, such as a setting the chart needs to let itself be removed
	Patches []UninstallPatch `yaml:"patches"`
	// What's deliberately left, shown when asking to confirm, such as media on a NAS
	Keeps []string `yaml:"keeps"`
}

// A JSON merge patch to a resource, as `kubectl patch --type merge` applies it
type UninstallPatch struct {
	Resource  string `yaml:"resource"`
	Namespace string `yaml:"namespace"`
	Name      string `yaml:"name"`
	Patch     string `yaml:"patch"`
}

// A persistent volume claim in a namespace, and the storage it requests
type NamespaceVolume struct {
	Name string
	Size string
}
