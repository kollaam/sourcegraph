package index

type AutoIndexConfiguration struct {
	Branches  BranchConfiguration
	IndexJobs []IndexJob
}

type BranchConfiguration struct {
	Include []string
	Exclude []string
}

type IndexJob struct {
	Root    string
	Install InstallationConfiguration
	Index   IndexConfiguration
}

type InstallationConfiguration struct {
	Image    string
	Commands []string
}

type IndexConfiguration struct {
	Indexer   string
	Arguments []string
}
