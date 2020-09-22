package index

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

const jsonTestInput = `
{
	"branches": {
		"include": [
			"master",
			"ef/.+",
		],
		"exclude": [
			"ef/test-.+",
		],
	},
	"indexJobs": [
		{
			"index": {
				"indexer": "sourcegraph/lsif-go:latest",
				"arguments": [
					"lsif-go",
					"--no-animation",
				],
			},
		},
		{
			"root": "web/",
			"install": {
				"image": "node:12",
				"commands": [
					"yarn",
					"install",
					"--frozen-lockfile",
					"--non-interactive",
				],
			},
			"index": {
				"indexer": "sourcegraph/lsif-node:latest",
				"arguments": [
					"lsif-tsc",
					"-p",
					".",
				],
			},
		},
	],
}
`

func TestUnmarshalJSON(t *testing.T) {
	actual, err := UnmarshalJSON([]byte(jsonTestInput))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	expected := AutoIndexConfiguration{
		Branches: BranchConfiguration{
			Include: []string{"master", "ef/.+"},
			Exclude: []string{"ef/test-.+"},
		},
		IndexJobs: []IndexJob{
			{
				Root: "",
				Index: IndexConfiguration{
					Indexer:   "sourcegraph/lsif-go:latest",
					Arguments: []string{"lsif-go", "--no-animation"},
				},
			},
			{
				Root: "web/",
				Install: InstallationConfiguration{
					Image:    "node:12",
					Commands: []string{"yarn", "install", "--frozen-lockfile", "--non-interactive"},
				},
				Index: IndexConfiguration{
					Indexer:   "sourcegraph/lsif-node:latest",
					Arguments: []string{"lsif-tsc", "-p", "."},
				},
			},
		},
	}
	if diff := cmp.Diff(expected, actual); diff != "" {
		t.Errorf("unexpected configuration (-want +got):\n%s", diff)
	}
}
