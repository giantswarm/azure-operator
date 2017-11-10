package spec

import (
	"github.com/giantswarm/azuretpr/spec/azure"
)

type Azure struct {
	KeyVault azure.KeyVault `json:"keyVault" yaml:"keyVault"`
	// Location is the region for the resource group.
	Location       string               `json:"location" yaml:"location"`
	Storage        azure.Storage        `json:"storage" yaml:"storage"`
	VirtualNetwork azure.VirtualNetwork `json:"virtualNetwork" yaml:"virtualNetwork"`
	Masters        []azure.Node         `json:"masters" yaml:"masters"`
	Workers        []azure.Node         `json:"workers" yaml:"workers"`
	DNSZones       azure.DNSZones       `json:"dnsZones" yaml:"dnsZones"`
}
