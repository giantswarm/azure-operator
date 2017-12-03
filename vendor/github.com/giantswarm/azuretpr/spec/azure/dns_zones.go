package azure

// DNSZone points to a DNS Zone in Azure.
type DNSZone struct {
	// ResourceGroup is the resource group of the zone.
	ResourceGroup string `json:"resourceGroup" yaml:"resourceGroup"`
	// Name is the name of the zone.
	Name string `json:"name" yaml:"name"`
}

// DNSZones contains the DNS Zones of the cluster.
type DNSZones struct {
	// API is the DNS Zone for the Kubernetes API.
	API DNSZone `json:"api" yaml:"api"`
	// Etcd is the DNS Zone for the etcd cluster.
	Etcd DNSZone `json:"etcd" yaml:"etcd"`
	// Ingress is the DNS Zone for the Ingress resource, used for customer traffic.
	Ingress DNSZone `json:"ingress" yaml:"ingress"`
}
