package ipam

import (
	"context"
	"net"
	"reflect"

	"github.com/giantswarm/microerror"
)

const (
	// minAllocatedSubnetMaskBits is the maximum size of guest subnet i.e.
	// smaller number here -> larger subnet per guest cluster. For now anything
	// under 16 doesn't make sense in here.
	minAllocatedSubnetMaskBits = 16
)

type AzureConfigNetworkRangeGetterConfig struct {
	NetworkRange            net.IPNet
	RequiredNetworkMaskBits int
}

// AzureConfigNetworkRangeGetter is NetworkRangeGetter implementation for
// AzureConfig.
type AzureConfigNetworkRangeGetter struct {
	networkRange        net.IPNet
	requiredNetworkMask net.IPMask
}

func NewAzureConfigNetworkRangeGetter(config AzureConfigNetworkRangeGetterConfig) (*AzureConfigNetworkRangeGetter, error) {
	if reflect.DeepEqual(config.NetworkRange, net.IPNet{}) {
		return nil, microerror.Maskf(invalidConfigError, "%T.NetworkRange must not be empty", config)
	}
	if config.RequiredNetworkMaskBits < minAllocatedSubnetMaskBits {
		return nil, microerror.Maskf(invalidConfigError, "%T.RequiredNetworkMaskBits (%d) must not be smaller than %d", config, config.RequiredNetworkMaskBits, minAllocatedSubnetMaskBits)
	}

	g := AzureConfigNetworkRangeGetter{
		networkRange: config.NetworkRange,
		requiredNetworkMask: net.CIDRMask(config.RequiredNetworkMaskBits, 32),
	}

	return &g, nil
}

// GetNetworkRange gets the predefined installation network range, since the
// tenant cluster virtual network is getting its IP range from all available
// address ranges in the installation.
func (g *AzureConfigNetworkRangeGetter) GetNetworkRange(_ context.Context, _ interface{}) (net.IPNet, error) {
	return g.networkRange, nil
}

// GetRequiredIPMask returns an IP mask for tenant cluster virtual network.
func (g *AzureConfigNetworkRangeGetter) GetRequiredIPMask() net.IPMask {
	return g.requiredNetworkMask
}
