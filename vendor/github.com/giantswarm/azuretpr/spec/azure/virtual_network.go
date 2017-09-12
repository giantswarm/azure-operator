package azure

import "github.com/giantswarm/azuretpr/spec/azure/loadbalancer"

type VirtualNetwork struct {
	// CIDR is the CIDR for the Virtual Network.
	CIDR string `json:"cidr" yaml:"cidr"`
	// MasterSubnetCIDR is the CIDR for the master subnet,
	MasterSubnetCIDR string `json:"masterSubnetCidr" yaml:"masterSubnetCidr"`
	// WorkerSubnetCIDR is the CIDR for the worker subnet,
	WorkerSubnetCIDR string                    `json:"workerSubnetCidr" yaml:"workerSubnetCidr"`
	LoadBalancer     loadbalancer.LoadBalancer `json:"loadBalancer" yaml:"loadBalancer"`
}
