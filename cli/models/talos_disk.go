package models

// TalosDisk is a block device as reported by "talosctl get disks -o yaml".
type TalosDisk struct {
	Spec TalosDiskSpec `yaml:"spec"`
}

type TalosDiskSpec struct {
	DevPath    string `yaml:"dev_path"`
	PrettySize string `yaml:"pretty_size"`
	Model      string `yaml:"model"`
	Transport  string `yaml:"transport"`
	ReadOnly   bool   `yaml:"readonly"`
	CdRom      bool   `yaml:"cdrom"`
}
