package ipam

import (
	"context"
	"net"
	"time"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/microstorage"
)

const (
	ipamSubnetStorageKey = "/ipam/subnet"
)

// Config represents the configuration used to create a new ipam service.
type Config struct {
	Logger  micrologger.Logger
	Storage microstorage.Storage

	// Network is the network in which all returned subnets should exist.
	Network *net.IPNet
	// AllocatedSubnets is a list of subnets, contained by `Network`,
	// that have already been allocated outside of IPAM control.
	// Any subnets created by the IPAM service will not overlap with these subnets.
	AllocatedSubnets []net.IPNet
}

// New creates a new configured ipam service.
func New(config Config) (*Service, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "logger must not be empty")
	}
	if config.Storage == nil {
		return nil, microerror.Maskf(invalidConfigError, "storage must not be empty")
	}

	if config.Network == nil {
		return nil, microerror.Maskf(invalidConfigError, "network must not be empty")
	}
	for _, allocatedSubnet := range config.AllocatedSubnets {
		ipRange := newIPRange(allocatedSubnet)
		if !(config.Network.Contains(ipRange.start) && config.Network.Contains(ipRange.end)) {
			return nil, microerror.Maskf(
				invalidConfigError,
				"allocated subnet (%v) must be contained by network (%v)",
				allocatedSubnet.String(),
				config.Network.String(),
			)
		}
	}

	newService := &Service{
		logger:  config.Logger,
		storage: config.Storage,

		network:          *config.Network,
		allocatedSubnets: config.AllocatedSubnets,
	}

	return newService, nil
}

type Service struct {
	logger  micrologger.Logger
	storage microstorage.Storage

	network          net.IPNet
	allocatedSubnets []net.IPNet
}

// listSubnets retrieves the stored subnets from storage and returns them.
func (s *Service) listSubnets(ctx context.Context) ([]net.IPNet, error) {
	s.logger.Log("info", "listing subnets")

	k, err := microstorage.NewK(ipamSubnetStorageKey)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	kvs, err := s.storage.List(ctx, k)
	if err != nil && !microstorage.IsNotFound(err) {
		return nil, microerror.Mask(err)
	}

	existingSubnets := []net.IPNet{}
	for _, kv := range kvs {
		// Storage returns the relative key with List, not the values.
		// Instead of then requesting each value, we revert the key to a valid
		// CIDR string.
		existingSubnetString := decodeRelativeKey(kv.Val())

		_, existingSubnet, err := net.ParseCIDR(existingSubnetString)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		existingSubnets = append(existingSubnets, *existingSubnet)
	}

	subnetCounter.Set(float64(len(existingSubnets)))

	return existingSubnets, nil
}

// NewSubnet returns the next available subnet, of the configured size,
// from the configured network.
func (s *Service) NewSubnet(mask net.IPMask) (net.IPNet, error) {
	s.logger.Log("info", "creating new subnet")
	defer updateMetrics("create", time.Now())

	ctx := context.Background()

	existingSubnets, err := s.listSubnets(ctx)
	if err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}

	existingSubnets = append(existingSubnets, s.allocatedSubnets...)

	s.logger.Log("info", "computing next subnet")
	subnet, err := Free(s.network, mask, existingSubnets)
	if err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}

	s.logger.Log("info", "putting subnet", "subnet", subnet.String())
	kv, err := microstorage.NewKV(encodeKey(subnet), subnet.String())
	if err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}
	if err := s.storage.Put(ctx, kv); err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}

	return subnet, nil
}

// DeleteSubnet deletes the given subnet from IPAM storage,
// meaning it can be given out again.
func (s *Service) DeleteSubnet(subnet net.IPNet) error {
	s.logger.Log("info", "deleting subnet", "subnet", subnet.String())
	defer updateMetrics("delete", time.Now())

	ctx := context.Background()

	k, err := microstorage.NewK(encodeKey(subnet))
	if err != nil {
		return microerror.Mask(err)
	}
	if err := s.storage.Delete(ctx, k); err != nil {
		return microerror.Mask(err)
	}

	return nil
}
