package virtualnetwork

type LoadBalancer struct {
	// EtcdCidr is the CIDR for the etcd load balancer.
	EtcdCIDR string `json:"etcdCidr" yaml:"etcdCidr"`
}
