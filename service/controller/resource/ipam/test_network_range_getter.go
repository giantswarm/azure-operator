package ipam

import (
	"context"
	"net"
)

type TestNetworkRangeGetter struct {
	networkRange        net.IPNet
	requiredNetworkMask net.IPMask
}

func NewTestNetworkRangeGetter(networkRange net.IPNet, requiredNetworkMaskBits int) *TestNetworkRangeGetter {
	g := &TestNetworkRangeGetter{
		networkRange:        networkRange,
		requiredNetworkMask: net.CIDRMask(requiredNetworkMaskBits, 32),
	}

	return g
}

func (g *TestNetworkRangeGetter) GetNetworkRange(_ context.Context, _ interface{}) (net.IPNet, error) {
	return g.networkRange, nil
}

func (g *TestNetworkRangeGetter) GetRequiredIPMask() net.IPMask {
	return g.requiredNetworkMask
}
