package virtualnetwork

type LoadBalancer struct {
	// APICIDR is the CIDR for the apiserver load balancer.
	APICIDR string `json:"apiCidr" yaml:"apiCidr"`
	// EtcdCidr is the CIDR for the etcd load balancer.
	EtcdCIDR string `json:"etcdCidr" yaml:"etcdCidr"`
	// IngressCidr is the CIDR for the ingress controller load balancer.
	IngressCIDR string `json:"ingressCidr" yaml:"ingressCidr"`
}
