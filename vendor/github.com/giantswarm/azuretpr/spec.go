package azuretpr

import (
	"github.com/giantswarm/azuretpr/spec"
	"github.com/giantswarm/clustertpr"
)

type Spec struct {
	Cluster clustertpr.Spec `json:"cluster" yaml:"cluster"`
	Azure   spec.Azure      `json:"azure" yaml:"azure"`
}
