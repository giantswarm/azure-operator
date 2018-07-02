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
	ipv4MaskSize = 32
)

// ComputeFromCR computes subnets using network found in CR.
func ComputeFromCR(ctx context.Context, obj interface{}, subnetsSetting setting.AzureNetwork) (*Subnets, error) {
	azureConfig, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vnetCIDR := key.VnetCIDR(azureConfig)
	_, vnet, err := net.ParseCIDR(vnetCIDR)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	subnets, err := Compute(*vnet, subnetsSetting)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return subnets, nil
}

// Compute computes subnets within network based on subnetsSetting.
//
// subnets computation rely on ipam.Free and use ipv4MaskSize as IPMask length.
func Compute(network net.IPNet, subnetsSetting setting.AzureNetwork) (subnets *Subnets, err error) {
	subnets = new(Subnets)

	subnets.Parent = network

	_, subnets.Calico, err = ipam.Half(network)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	masterSubnetMask := net.CIDRMask(subnetsSetting.MasterSubnetMask, ipv4MaskSize)
	subnets.Master, err = ipam.Free(network, masterSubnetMask, []net.IPNet{subnets.Calico})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	workerSubnetMask := net.CIDRMask(subnetsSetting.WorkerSubnetMask, ipv4MaskSize)
	subnets.Worker, err = ipam.Free(network, workerSubnetMask, []net.IPNet{subnets.Calico, subnets.Master})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	vpnSubnetMask := net.CIDRMask(subnetsSetting.VPNSubnetMask, ipv4MaskSize)
	subnets.VPN, err = ipam.Free(network, vpnSubnetMask, []net.IPNet{subnets.Calico, subnets.Master, subnets.Worker})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return subnets, nil
}
