package spec

import (
	"github.com/giantswarm/azuretpr/spec/azure"
)

type Azure struct {
	Location       string               `json:"location" yaml:"location"`
	VirtualNetwork azure.VirtualNetwork `json:"virtualNetwork" yaml:"virtualNetwork"`
}
