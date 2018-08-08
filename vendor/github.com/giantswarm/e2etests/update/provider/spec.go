package provider

type Interface interface {
	CurrentVersion() (string, error)
	IsCreated() (bool, error)
	IsUpdated() (bool, error)
	NextVersion() (string, error)
	UpdateVersion(nextVersion string) error
}

type Patch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}
