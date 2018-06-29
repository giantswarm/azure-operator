package network

import (
	"context"
	"net"

	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

const (
	ipv4Mask = 32
)

// Compute network subnets within network from CR.
func ComputeFromCR(ctx context.Context, obj interface{}, networkSetting setting.AzureNetwork) (*Subnets, error) {
	azureConfig, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vnetCIDR := key.VnetCIDR(azureConfig)
	_, vnet, err := net.ParseCIDR(vnetCIDR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	subnets, err := Compute(*vnet, networkSetting)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return subnets, nil
}

// Compute network subnets.
func Compute(network net.IPNet, networkSetting setting.AzureNetwork) (subnets *Subnets, err error) {
	subnets = new(Subnets)

	subnets.Parent = network

	masterSubnetMask := net.CIDRMask(networkSetting.MasterSubnetMask, ipv4Mask)
	subnets.Master, err = ipam.Free(network, masterSubnetMask, nil)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	workerSubnetMask := net.CIDRMask(networkSetting.WorkerSubnetMask, ipv4Mask)
	subnets.Worker, err = ipam.Free(network, workerSubnetMask, []net.IPNet{subnets.Master})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vpnSubnetMask := net.CIDRMask(networkSetting.VPNSubnetMask, ipv4Mask)
	subnets.VPN, err = ipam.Free(network, vpnSubnetMask, []net.IPNet{subnets.Master, subnets.Worker})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	calicoSubnetMask := net.CIDRMask(networkSetting.CalicoSubnetMask, ipv4Mask)
	subnets.Calico, err = ipam.Free(network, calicoSubnetMask, []net.IPNet{subnets.Master, subnets.Worker, subnets.VPN})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return subnets, nil
}
