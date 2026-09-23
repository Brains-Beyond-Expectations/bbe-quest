package models

// Talos machine configs are multi-document YAML files: one primary (kind-less)
// v1alpha1.Config document followed by many small typed documents, each identified by its
// kind. Only the documents bbe modifies are modelled here - every other document is kept
// as-is by talos_service, and unknown keys within the modelled documents are preserved
// through the inlined Unmapped fields.
const TalosConfigApiVersion = "v1alpha1"

const (
	TalosLinkConfigKind              = "LinkConfig"
	TalosHostnameConfigKind          = "HostnameConfig"
	TalosUnattendedInstallConfigKind = "UnattendedInstallConfig"
	TalosKubeClusterConfigKind       = "KubeClusterConfig"
	TalosKubeNodeConfigKind          = "KubeNodeConfig"
)

// TalosControlPlaneTaint keeps workloads off control plane nodes; removing it is what
// allows scheduling on them.
const TalosControlPlaneTaint = "node-role.kubernetes.io/control-plane"

// TalosLinkConfig configures a single network link. Talos does not generate this document
// by default - it assumes DHCP until a link is configured explicitly.
type TalosLinkConfig struct {
	ApiVersion string                 `yaml:"apiVersion,omitempty"`
	Kind       string                 `yaml:"kind,omitempty"`
	Name       string                 `yaml:"name,omitempty"`
	Addresses  []TalosLinkAddress     `yaml:"addresses,omitempty"`
	Routes     []TalosLinkRoute       `yaml:"routes,omitempty"`
	Unmapped   map[string]interface{} `yaml:",inline"`
}

// TalosLinkAddress holds a single address in CIDR notation, e.g. "192.168.1.161/24".
type TalosLinkAddress struct {
	Address  string                 `yaml:"address,omitempty"`
	Unmapped map[string]interface{} `yaml:",inline"`
}

// TalosLinkRoute holds a single route. An empty Destination means the default route.
type TalosLinkRoute struct {
	Destination string                 `yaml:"destination,omitempty"`
	Gateway     string                 `yaml:"gateway,omitempty"`
	Unmapped    map[string]interface{} `yaml:",inline"`
}

// TalosHostnameConfig sets the machine's hostname. Hostname and Auto are mutually
// exclusive - Talos rejects a config that sets both.
type TalosHostnameConfig struct {
	ApiVersion string                 `yaml:"apiVersion,omitempty"`
	Kind       string                 `yaml:"kind,omitempty"`
	Hostname   string                 `yaml:"hostname,omitempty"`
	Auto       string                 `yaml:"auto,omitempty"`
	Unmapped   map[string]interface{} `yaml:",inline"`
}

// TalosUnattendedInstallConfig controls where Talos installs itself.
type TalosUnattendedInstallConfig struct {
	ApiVersion   string                 `yaml:"apiVersion,omitempty"`
	Kind         string                 `yaml:"kind,omitempty"`
	Provisioning TalosProvisioning      `yaml:"provisioning,omitempty"`
	Unmapped     map[string]interface{} `yaml:",inline"`
}

type TalosProvisioning struct {
	DiskSelector TalosDiskSelector      `yaml:"diskSelector,omitempty"`
	Unmapped     map[string]interface{} `yaml:",inline"`
}

// TalosDiskSelector picks the install disk with a CEL expression rather than a bare path,
// e.g. `disk.dev_path == "/dev/sda"`.
type TalosDiskSelector struct {
	Match    string                 `yaml:"match,omitempty"`
	Unmapped map[string]interface{} `yaml:",inline"`
}

// TalosKubeClusterConfig holds the Kubernetes API endpoint of the cluster.
type TalosKubeClusterConfig struct {
	ApiVersion string                 `yaml:"apiVersion,omitempty"`
	Kind       string                 `yaml:"kind,omitempty"`
	Endpoint   string                 `yaml:"endpoint,omitempty"`
	Unmapped   map[string]interface{} `yaml:",inline"`
}

// TalosKubeNodeConfig holds the node's labels and taints.
type TalosKubeNodeConfig struct {
	ApiVersion string                 `yaml:"apiVersion,omitempty"`
	Kind       string                 `yaml:"kind,omitempty"`
	Taints     map[string]string      `yaml:"taints"`
	Unmapped   map[string]interface{} `yaml:",inline"`
}
