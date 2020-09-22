package index

import (
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/jsonx"
)

type jsonAutoIndexConfiguration struct {
	Branches  jsonBranchConfiguration `json:"branches"`
	IndexJobs []jsonIndexJob          `json:"indexJobs"`
}

type jsonBranchConfiguration struct {
	Include []string `json:"include"`
	Exclude []string `json:"exclude"`
}

type jsonIndexJob struct {
	Root    string                        `json:"root"`
	Install jsonInstallationConfiguration `json:"install"`
	Index   jsonIndexConfiguration        `json:"index"`
}

type jsonInstallationConfiguration struct {
	Image    string   `json:"image"`
	Commands []string `json:"commands"`
}

type jsonIndexConfiguration struct {
	Indexer   string   `json:"indexer"`
	Arguments []string `json:"arguments"`
}

func UnmarshalJSON(data []byte) (AutoIndexConfiguration, error) {
	jsonData, errs := jsonx.Parse(string(data), jsonx.ParseOptions{Comments: true, TrailingCommas: true})
	if len(errs) > 0 {
		return AutoIndexConfiguration{}, fmt.Errorf("invalid JSON: %v", errs)
	}

	configuration := jsonAutoIndexConfiguration{}
	if err := json.Unmarshal(jsonData, &configuration); err != nil {
		return AutoIndexConfiguration{}, fmt.Errorf("invalid JSON: %v", err)
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
