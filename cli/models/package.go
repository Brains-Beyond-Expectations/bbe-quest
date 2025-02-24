package models

type Package struct {
	Name    string `yaml:"name,omitempty"`
	Version string `yaml:"version,omitempty"`
}
