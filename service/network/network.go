package network

import (
	"net"

	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"
)

const (
	masterSubnetMask = 24
	workerSubnetMask = 24
	vpnSubnetMask    = 24

	ipv4MaskSize = 32
)

// Compute computes subnets within network.
//
// subnets computation rely on ipam.Free and use ipv4MaskSize as IPMask length.
func Compute(network net.IPNet) (subnets *Subnets, err error) {
	subnets = new(Subnets)

	subnets.Parent = network

	_, subnets.Calico, err = ipam.Half(network)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	masterCIDRMask := net.CIDRMask(masterSubnetMask, ipv4MaskSize)
	subnets.Master, err = ipam.Free(network, masterCIDRMask, []net.IPNet{subnets.Calico})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	workerCIDRMask := net.CIDRMask(workerSubnetMask, ipv4MaskSize)
	subnets.Worker, err = ipam.Free(network, workerCIDRMask, []net.IPNet{subnets.Calico, subnets.Master})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vpnCIDRMask := net.CIDRMask(vpnSubnetMask, ipv4MaskSize)
	subnets.VPN, err = ipam.Free(network, vpnCIDRMask, []net.IPNet{subnets.Calico, subnets.Master, subnets.Worker})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return subnets, nil
}
