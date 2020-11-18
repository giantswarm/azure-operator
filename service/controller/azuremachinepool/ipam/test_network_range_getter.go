package ipam

import (
	"context"
	"net"
)

type TestNetworkRangeGetter struct {
	parentNetworkRange  net.IPNet
	requiredNetworkMask net.IPMask
}

func NewTestNetworkRangeGetter(parentNetworkRange net.IPNet, requiredNetworkMaskBits int) *TestNetworkRangeGetter {
	g := &TestNetworkRangeGetter{
		parentNetworkRange:  parentNetworkRange,
		requiredNetworkMask: net.CIDRMask(requiredNetworkMaskBits, 32),
	}

	return g
}

func (g *TestNetworkRangeGetter) GetParentNetworkRange(_ context.Context, _ interface{}) (net.IPNet, error) {
	return g.parentNetworkRange, nil
}

func (g *TestNetworkRangeGetter) GetRequiredIPMask() net.IPMask {
	return g.requiredNetworkMask
}
