package spec

import (
	"github.com/giantswarm/azuretpr/spec/azure"
)

type Azure struct {
	// Location is the region for the resource group.
	Location       string               `json:"location" yaml:"location"`
	Storage        azure.Storage        `json:"storage" yaml:"storage"`
	VirtualNetwork azure.VirtualNetwork `json:"virtualNetwork" yaml:"virtualNetwork"`
}
