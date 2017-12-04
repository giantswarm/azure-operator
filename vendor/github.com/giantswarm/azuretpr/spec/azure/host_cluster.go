package azure

type HostCluster struct {
	// CIDR is the CIDR of the host cluster Virtual Network.
	// This is going to be used by the Guest Cluster to allow SSH traffic from that CIDR.
	CIDR string `json:"cidr" yaml:"cidr"`
	// ResourceGroup is the resource group name of the host cluster. It is used to determine DNS hosted zone to put NS records in.
	ResourceGroup string `json:"resourceGroup" yaml:"resourceGroup"`
}
