package ipam

import (
	"context"
	"net"
)

// Checker determines whether a subnet has been allocated. This decision is
// being made based on the status of the Kubernetes runtime object defined by
// namespace and name. If subnet has been allocated, it's returned. Otherwise
// return value is nil.
type Checker interface {
	Check(ctx context.Context, namespace, name string) (*net.IPNet, error)
}

// Collector implementation must return all networks that are allocated on any
// given moment. Failing to do that will result in overlapping allocations.
type Collector interface {
	Collect(ctx context.Context, obj interface{}) ([]net.IPNet, error)
}

// NetworkRangeGetter implementation returns a network range from which a free
// IP range can be allocated.
type NetworkRangeGetter interface {
	// GetParentNetworkRange return the network range from which the VNet/subnet range
	// will be allocated. It receives the CR that is being reconciled.
	GetParentNetworkRange(ctx context.Context, obj interface{}) (net.IPNet, error)

	// GetRequiredIPMask returns an IP mask that is required by the network range
	// that will be allocated.
	GetRequiredIPMask() net.IPMask
}

// Persister must mutate shared persistent state so that on successful execution
// persisted networks are visible by Collector implementations.
type Persister interface {
	Persist(ctx context.Context, subnet net.IPNet, namespace, name string) error
}

// Releaser must mutate shared persistent state so that on successful execution
// allocated subnet is released.
type Releaser interface {
	Release(ctx context.Context, subnet net.IPNet, namespace, name string) error
}
