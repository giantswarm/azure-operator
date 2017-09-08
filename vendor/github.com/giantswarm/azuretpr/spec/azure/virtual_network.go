package azure

type VirtualNetwork struct {
	CIDR             string `json:"cidr" yaml:"cidr"`
	MasterSubnetCIDR string `json:"masterSubnetCidr" yaml:"masterSubnetCidr"`
	WorkerSubnetCIDR string `json:"workerSubnetCidr" yaml:"workerSubnetCidr"`
}
