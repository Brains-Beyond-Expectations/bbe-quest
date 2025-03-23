package models

type BbeConfig struct {
	Bbe struct {
		Cluster struct {
			Name    string `yaml:"name,omitempty"`
			Context string `yaml:"context,omitempty"`
		} `yaml:"cluster,omitempty"`
		Storage struct {
			Type string `yaml:"type,omitempty"` // "local" or "aws"
			Aws  struct {
				BucketName string `yaml:"bucket_name,omitempty"`
			} `yaml:"aws,omitempty"`
		} `yaml:"storage,omitempty"`
		Bundles []BbeBundle `yaml:"bundles"`
	} `yaml:"bbe,omitempty"`
}
