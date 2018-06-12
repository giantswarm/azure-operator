package network

import (
	"fmt"
	"net"

	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"
)

const (
	e2eNetwork        = "11.%d.0.0"
	e2eSubnetQuantity = 256

	azureCalicoSubnetMask = 17
	azureMasterSubnetMask = 24
	azureWorkerSubnetMask = 24
	azureVPNSubnetMask    = 27
)

type azureCIDR struct {
	AzureCIDR        string
	CalicoSubnetCIDR string
	MasterSubnetCIDR string
	VPNSubnetCIDR    string
	WorkerSubnetCIDR string
}

func ComputeCIDR(buildNumber uint) (*azureCIDR, error) {
	cidrs := new(azureCIDR)

	azureNetwork := determineSubnet(e2eNetwork, e2eSubnetQuantity, buildNumber)
	cidrs.AzureCIDR = azureNetwork.String()

	azureMasterSubnet, err := ipamFree(azureNetwork, azureMasterSubnetMask)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	cidrs.MasterSubnetCIDR = azureMasterSubnet.String()

	azureWorkerSubnet, err := ipamFree(azureNetwork, azureWorkerSubnetMask, *azureMasterSubnet)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	cidrs.WorkerSubnetCIDR = azureWorkerSubnet.String()

	azureVPNSubnet, err := ipamFree(azureNetwork, azureVPNSubnetMask, *azureMasterSubnet, *azureWorkerSubnet)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	cidrs.VPNSubnetCIDR = azureVPNSubnet.String()

	azureCalicoSubnet, err := ipamFree(azureNetwork, azureCalicoSubnetMask, *azureMasterSubnet, *azureWorkerSubnet, *azureVPNSubnet)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	cidrs.CalicoSubnetCIDR = azureCalicoSubnet.String()

	return cidrs, nil
}

// ipamFree wrap call to ipam.Free, and use 32 bits CIDRMask.
func ipamFree(network net.IPNet, mask int, allocatedSubnet ...net.IPNet) (subnet *net.IPNet, err error) {
	s, err := ipam.Free(network, net.CIDRMask(mask, 32), allocatedSubnet)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &s, nil
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
