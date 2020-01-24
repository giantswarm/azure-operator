package network

import (
	"fmt"
	"net"

	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/network"
)

const (
	e2eNetwork        = "11.%d.0.0"
	e2eSubnetQuantity = 256
)

func ComputeSubnets(buildNumber uint) (*network.Subnets, error) {
	azureNetwork := determineSubnet(e2eNetwork, e2eSubnetQuantity, buildNumber)

	return network.Compute(azureNetwork)
}

func ComputeE2EBastionSubnet(buildNumber uint, subnets *network.Subnets) (net.IPNet, error) {
	azureNetwork := determineSubnet(e2eNetwork, e2eSubnetQuantity, buildNumber)
	subnet, err := ipam.Free(azureNetwork, net.CIDRMask(24, 32), []net.IPNet{subnets.Calico, subnets.Master, subnets.Worker})
	if err != nil {
		return net.IPNet{}, microerror.Mask(err)
	}

	return subnet, err
}

// determineSubnet compute a subnet by wrapping decider in subnetQuantity and writing the resulting value in cidrFormat.
// cidrFormat must hold exactly one decimal verb (%d).
func determineSubnet(cidrFormat string, subnetQuantity uint, decider uint) net.IPNet {
	subnetIP := fmt.Sprintf(cidrFormat, int(decider%subnetQuantity))

	return net.IPNet{
		IP:   net.ParseIP(subnetIP),
		Mask: net.CIDRMask(16, 32),
	}
}
