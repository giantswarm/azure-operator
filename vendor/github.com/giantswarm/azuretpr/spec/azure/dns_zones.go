package azure

// DNSZones contains the DNS Zones of the cluster.
type DNSZones struct {
	// API is the DNS Zone for the Kubernetes API.
	API string `json:"api" yaml:"api"`
	// Etcd is the DNS Zone for the etcd cluster.
	Etcd string `json:"etcd" yaml:"etcd"`
	// Ingress is the DNS Zone for the Ingress resource, used for customer traffic.
	Ingress string `json:"ingress" yaml:"ingress"`
}
