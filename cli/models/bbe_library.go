package models

type Library struct {
	Library []LibraryEntry `mapstructure:"library"`
}

type LibraryEntry struct {
	MinBbeCli    string       `mapstructure:"min-bbe-cli" yaml:"min-bbe-cli"`
	ListRevision int          `mapstructure:"list-revision" yaml:"list-revision"`
	Charts       []ChartEntry `mapstructure:"charts" yaml:"charts"`
}

type ChartEntry struct {
	Name           string `mapstructure:"name" yaml:"name"`
	Version        string `mapstructure:"version" yaml:"version"`
	RepositoryUrl  string `mapstructure:"repositoryUrl" yaml:"repositoryUrl"`
	RepositoryName string `mapstructure:"repositoryName" yaml:"repositoryName"`
}
