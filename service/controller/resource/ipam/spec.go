package ipam

import (
	"context"
	"net"
)

// Checker determines whether a subnet has to be allocated. This decision is
// being made based on the status of the Kubernetes runtime object defined by
// namespace and name.
type Checker interface {
	Check(ctx context.Context, namespace string, name string) (bool, error)
}

// Collector implementation must return all networks that are allocated on any
// given moment. Failing to do that will result in overlapping allocations.
type Collector interface {
	Collect(ctx context.Context, obj interface{}) ([]net.IPNet, error)
}

// NetworkRangeGetter implementation returns a network range from which a free
// IP range can be allocated.
type NetworkRangeGetter interface {
	// GetNetworkRange gets the network range from which the VNet/subnet range
	// will be allocated. It receives the CR that is being reconciled.
	GetNetworkRange(ctx context.Context, obj interface{}) (net.IPNet, error)

	// GetRequiredIPMask returns an IP mask that is required by the network range
	// that will be allocated.
	GetRequiredIPMask() net.IPMask
}

// Persister must mutate shared persistent state so that on successful execution
// persisted networks are visible by Collector implementations.
type Persister interface {
	Persist(ctx context.Context, subnet net.IPNet, namespace string, name string) error
}
