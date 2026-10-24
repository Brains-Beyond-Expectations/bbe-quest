package models

// Talos machine configs are multi-document YAML files: one primary (kind-less)
// v1alpha1.Config document followed by many small typed documents, each identified by its
// kind. Only the documents bbe modifies are modelled here - every other document is kept
// as-is by talos_service, and unknown keys within the modelled documents are preserved
// through the inlined Unmapped fields.
const TalosConfigApiVersion = "v1alpha1"

const (
	TalosLinkConfigKind                  = "LinkConfig"
	TalosHostnameConfigKind              = "HostnameConfig"
	TalosUnattendedInstallConfigKind     = "UnattendedInstallConfig"
	TalosKubeClusterConfigKind           = "KubeClusterConfig"
	TalosResolverConfigKind              = "ResolverConfig"
	TalosKubeNodeConfigKind              = "KubeNodeConfig"
	TalosUserVolumeConfigKind            = "UserVolumeConfig"
	TalosKubeNetworkConfigKind           = "KubeNetworkConfig"
	TalosKubeFlannelCNIConfigKind        = "KubeFlannelCNIConfig"
	TalosKubeControllerManagerConfigKind = "KubeControllerManagerConfig"
	TalosKubeSchedulerConfigKind         = "KubeSchedulerConfig"
	TalosKubeProxyConfigKind             = "KubeProxyConfig"
	TalosNetworkRuleConfigKind           = "NetworkRuleConfig"
)

// TalosControlPlaneTaint keeps workloads off control plane nodes; removing it is what
// allows scheduling on them.
const TalosControlPlaneTaint = "node-role.kubernetes.io/control-plane"

// TalosMachineConfig is the primary (kind-less) v1alpha1.Config document. Only the settings
// bbe modifies are modelled - the rest of the document, including the machine's PKI and
// tokens, is preserved through the inlined Unmapped fields.
type TalosMachineConfig struct {
	Machine  TalosMachine           `yaml:"machine,omitempty"`
	Cluster  TalosCluster           `yaml:"cluster,omitempty"`
	Unmapped map[string]interface{} `yaml:",inline"`
}

type TalosMachine struct {
	Time     TalosMachineTime       `yaml:"time,omitempty"`
	Unmapped map[string]interface{} `yaml:",inline"`
}

type TalosCluster struct {
	Etcd     TalosEtcd              `yaml:"etcd,omitempty"`
	Unmapped map[string]interface{} `yaml:",inline"`
}

// TalosEtcd is etcd's configuration, which in Talos v1.14 is still part of the primary document
type TalosEtcd struct {
	ExtraArgs map[string]interface{} `yaml:"extraArgs,omitempty"`
	Unmapped  map[string]interface{} `yaml:",inline"`
}

// TalosMachineTime configures NTP. Talos falls back to a public time server when no servers
// are set, which leaves the node unable to sync its clock on networks that don't allow
// outbound NTP - and a node with the wrong time fails certificate validation on boot.
type TalosMachineTime struct {
	Servers  []string               `yaml:"servers,omitempty"`
	Unmapped map[string]interface{} `yaml:",inline"`
}

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

// TalosResolverConfig sets the DNS nameservers. This has to be set whenever bbe assigns a
// static address: applying any LinkConfig disables DHCP on that link, and DHCP is what
// would otherwise supply the node's nameservers.
type TalosResolverConfig struct {
	ApiVersion  string                 `yaml:"apiVersion,omitempty"`
	Kind        string                 `yaml:"kind,omitempty"`
	Nameservers []TalosNameserver      `yaml:"nameservers,omitempty"`
	Unmapped    map[string]interface{} `yaml:",inline"`
}

type TalosNameserver struct {
	Address  string                 `yaml:"address,omitempty"`
	Unmapped map[string]interface{} `yaml:",inline"`
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
	Installer    TalosInstaller         `yaml:"installer,omitempty"`
	Provisioning TalosProvisioning      `yaml:"provisioning,omitempty"`
	Unmapped     map[string]interface{} `yaml:",inline"`
}

// TalosInstaller is the image Talos installs itself to disk from, whose system extensions the node then has
type TalosInstaller struct {
	Image    string                 `yaml:"image,omitempty"`
	Unmapped map[string]interface{} `yaml:",inline"`
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

// A directory user volume is a folder on the system disk, which Talos creates at /var/mnt/<name>
type TalosUserVolumeConfig struct {
	ApiVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Name       string                 `yaml:"name"`
	VolumeType string                 `yaml:"volumeType,omitempty"`
	Unmapped   map[string]interface{} `yaml:",inline"`
}

// TalosKubeNetworkConfig holds the cluster's pod and service subnets
type TalosKubeNetworkConfig struct {
	ApiVersion string                 `yaml:"apiVersion,omitempty"`
	Kind       string                 `yaml:"kind,omitempty"`
	PodSubnets []string               `yaml:"podSubnets,omitempty"`
	Unmapped   map[string]interface{} `yaml:",inline"`
}

// TalosKubeFlannelCNIConfig configures Flannel, the network Talos sets up for pods. With kubeNetworkPoliciesEnabled,
// Talos runs kube-network-policies alongside it, without which NetworkPolicies are accepted but not enforced
type TalosKubeFlannelCNIConfig struct {
	ApiVersion                 string                 `yaml:"apiVersion,omitempty"`
	Kind                       string                 `yaml:"kind,omitempty"`
	KubeNetworkPoliciesEnabled bool                   `yaml:"kubeNetworkPoliciesEnabled,omitempty"`
	Unmapped                   map[string]interface{} `yaml:",inline"`
}

// TalosKubeComponentConfig is the part bbe changes of the documents configuring the controller manager and the
// scheduler: their command line arguments
type TalosKubeComponentConfig struct {
	ApiVersion string                 `yaml:"apiVersion,omitempty"`
	Kind       string                 `yaml:"kind,omitempty"`
	ExtraArgs  map[string]interface{} `yaml:"extraArgs,omitempty"`
	Unmapped   map[string]interface{} `yaml:",inline"`
}

// TalosKubeProxyConfig configures kube-proxy, through its own configuration file rather than arguments
type TalosKubeProxyConfig struct {
	ApiVersion string                 `yaml:"apiVersion,omitempty"`
	Kind       string                 `yaml:"kind,omitempty"`
	Config     map[string]interface{} `yaml:"config,omitempty"`
	Unmapped   map[string]interface{} `yaml:",inline"`
}

// TalosNetworkRuleConfig is a rule of the node's firewall. While its default action is to accept, Talos drops new
// connections to the rule's ports from anywhere outside its subnets, and leaves every other port open
type TalosNetworkRuleConfig struct {
	ApiVersion   string                    `yaml:"apiVersion"`
	Kind         string                    `yaml:"kind"`
	Name         string                    `yaml:"name"`
	PortSelector TalosNetworkPortSelector  `yaml:"portSelector"`
	Ingress      []TalosNetworkRuleIngress `yaml:"ingress"`
}

// TalosNetworkPortSelector lists ports as numbers, and ranges as text such as "2379-2380", which Talos requires
type TalosNetworkPortSelector struct {
	Ports    []interface{} `yaml:"ports"`
	Protocol string        `yaml:"protocol"`
}

type TalosNetworkRuleIngress struct {
	Subnet string `yaml:"subnet"`
}
