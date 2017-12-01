package azure

type HostCluster struct {
	// CIDR is the CIDR of the host cluster Virtual Network.
	// This is going to be used by the Guest Cluster to allow SSH traffic from that CIDR.
	CIDR string `json:"cidr" yaml:"cidr"`
}
