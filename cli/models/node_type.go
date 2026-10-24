package models

type NodeType struct {
	OutputFile   string
	ImagerType   string
	ImageLink    string
	Extensions   []string
	TalosVersion string
	// The image Talos installs itself to disk from, with the same extensions as the image it boots from
	InstallerImage string
}
