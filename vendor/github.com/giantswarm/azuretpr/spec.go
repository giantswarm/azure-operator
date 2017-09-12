package azuretpr

import (
	"github.com/giantswarm/clustertpr"

	"github.com/giantswarm/azuretpr/spec"
)

type Spec struct {
	Cluster clustertpr.Spec `json:"cluster" yaml:"cluster"`
	Azure   spec.Azure      `json:"azure" yaml:"azure"`
}
