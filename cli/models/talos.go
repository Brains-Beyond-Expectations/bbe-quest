package models

type TalosMachineConfig struct {
	Machine struct {
		Network struct {
			Interfaces []TalosInterface       `mapstructure:"interfaces,omitempty"`
			Hostname   string                 `mapstructure:"hostname,omitempty"`
			Unmapped   map[string]interface{} `mapstructure:",remain"`
		} `mapstructure:"network,omitempty"`
		Install struct {
			Disk     string                 `mapstructure:"disk,omitempty"`
			Unmapped map[string]interface{} `mapstructure:",remain"`
		} `mapstructure:"install,omitempty"`
		Unmapped map[string]interface{} `mapstructure:",remain"`
	} `mapstructure:"machine,omitempty"`
	Cluster struct {
		ControlPlane struct {
			Endpoint string                 `mapstructure:"endpoint,omitempty"`
			Unmapped map[string]interface{} `mapstructure:",remain"`
		} `mapstructure:"controlPlane,omitempty"`
		AllowSchedulingOnControlPlanes bool                   `mapstructure:"allowSchedulingOnControlPlanes"`
		Unmapped                       map[string]interface{} `mapstructure:",remain"`
	} `mapstructure:"cluster,omitempty"`
	Unmapped map[string]interface{} `mapstructure:",remain"`
}

type TalosInterface struct {
	Interface string       `mapstructure:"interface,omitempty"`
	Routes    []TalosRoute `mapstructure:"routes,omitempty"`
	Addresses []string     `mapstructure:"addresses,omitempty"`
}

type TalosRoute struct {
	Network string `mapstructure:"network,omitempty"`
	Gateway string `mapstructure:"gateway,omitempty"`
}
