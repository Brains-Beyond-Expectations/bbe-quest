package models

// Something a chart needs in the cluster before it can be installed, listed in its Chart.yaml under the
// `bbe/prerequisites` annotation. It's met when every check it sets passes.
type ChartPrerequisite struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// Checks that a storage class with this name exists
	StorageClass string `yaml:"storageClass"`
	// Checks that every Talos node is set up for the chart
	Talos *TalosPrerequisite `yaml:"talos"`
	// The bbe package that provides the prerequisite, installed when it's missing
	Package string `yaml:"package"`
}

// What every node needs, which bbe sets up by patching and upgrading each node when it's missing
type TalosPrerequisite struct {
	// Official system extensions, such as `siderolabs/iscsi-tools`
	Extensions []string `yaml:"extensions"`
	// Directories on each node's system disk that pods can store data in, such as `/var/mnt/longhorn`. They
	// have to be directly under /var/mnt, where Talos keeps user volumes.
	Directories []string `yaml:"directories"`
}

// How bbe reaches the cluster to check prerequisites and prepare its nodes
type ClusterAccess struct {
	KubeContext    string
	ControlPlaneIp string
	TalosConfig    string
}
