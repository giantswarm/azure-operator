package deployment

import (
	"net"

	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"
)

func secondHalf(cidr string) (string, error) {
	var network net.IPNet
	{
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			return "", microerror.Mask(err)
		}
		network = *n
	}

	// One bit bigger mask will give a second half of a network.
	ones, bits := network.Mask.Size()
	// TODO assert
	ones++
	mask := net.CIDRMask(ones, bits)

	// Compute first half.
	first, err := ipam.Free(network, mask, nil)
	if err != nil {
		return "", microerror.Mask(err)
	}

	// Second half is computed by getting next free.
	second, err := ipam.Free(network, mask, []net.IPNet{first})
	if err != nil {
		return "", microerror.Mask(err)
	}

	return second.String(), nil
}
