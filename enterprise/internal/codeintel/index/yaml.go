package index

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

type yamlAutoIndexConfiguration struct {
	Branches  yamlBranchConfiguration `yaml:"branches"`
	IndexJobs []yamlIndexJob          `yaml:"indexJobs"`
}

type yamlBranchConfiguration struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

type yamlIndexJob struct {
	Root    string                        `yaml:"root"`
	Install yamlInstallationConfiguration `yaml:"install"`
	Index   yamlIndexConfiguration        `yaml:"index"`
}

type yamlInstallationConfiguration struct {
	Image    string   `yaml:"image"`
	Commands []string `yaml:"commands"`
}

type yamlIndexConfiguration struct {
	Indexer   string   `yaml:"indexer"`
	Arguments []string `yaml:"arguments"`
}

func UnmarshalYAML(data []byte) (AutoIndexConfiguration, error) {
	configuration := yamlAutoIndexConfiguration{}
	if err := yaml.Unmarshal(data, &configuration); err != nil {
		return AutoIndexConfiguration{}, fmt.Errorf("invalid YAML: %v", err)
	}

	var indexJobs []IndexJob
	for _, indexJob := range configuration.IndexJobs {
		indexJobs = append(indexJobs, IndexJob{
			Root: indexJob.Root,
			Install: InstallationConfiguration{
				Image:    indexJob.Install.Image,
				Commands: sliceize(indexJob.Install.Commands),
			},
			Index: IndexConfiguration{
				Indexer:   indexJob.Index.Indexer,
				Arguments: sliceize(indexJob.Index.Arguments),
			},
		})
	}

	return AutoIndexConfiguration{
		Branches: BranchConfiguration{
			Include: sliceize(configuration.Branches.Include),
			Exclude: sliceize(configuration.Branches.Exclude),
		},
		IndexJobs: indexJobs,
	}, nil
}

func sliceize(v []string) []string {
	if v == nil {
		return []string{}
	}
	return v
}
