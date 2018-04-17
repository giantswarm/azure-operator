package network

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net"
	"os"
	"strconv"

	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"
)

const (
	EnvCircleBuildNumber     = "CIRCLE_BUILD_NUM"
	EnvAzureCIDR             = "AZURE_CIDR"
	EnvAzureCalicoSubnetCIDR = "AZURE_CALICO_SUBNET_CIDR"
	EnvAzureMasterSubnetCIDR = "AZURE_MASTER_SUBNET_CIDR"
	EnvAzureWorkerSubnetCIDR = "AZURE_WORKER_SUBNET_CIDR"

	EnvE2ECIDR = "E2E_CIDR"
	EnvE2EMask = "E2E_MASK"

	e2eMasterSubnetMask = 24
	e2eWorkerSubnetMask = 24
	e2eCalicoSubnetMask = 17
)

func init() {
	var err error

	var azureSubnet *net.IPNet
	{
		azureSubnet, err = AzureCIDR()
		if err != nil {
			panic(err)
		}

		os.Setenv(EnvAzureCIDR, azureSubnet.String())
		log.Println("azureCIDR", azureSubnet.String())
	}

	azureMasterSubnet, err := NextSubnet(*azureSubnet, e2eMasterSubnetMask)
	if err != nil {
		panic(err)
	}
	os.Setenv(EnvAzureMasterSubnetCIDR, azureMasterSubnet.String())
	log.Println("azureMasterCIDR", azureMasterSubnet.String())

	azureWorkerSubnet, err := NextSubnet(*azureSubnet, e2eWorkerSubnetMask, *azureMasterSubnet)
	if err != nil {
		panic(err)
	}
	os.Setenv(EnvAzureWorkerSubnetCIDR, azureWorkerSubnet.String())
	log.Println("azureWorkerCIDR", azureWorkerSubnet.String())

	azureCalicoSubnet, err := NextSubnet(*azureSubnet, e2eCalicoSubnetMask, *azureMasterSubnet, *azureWorkerSubnet)
	if err != nil {
		panic(err)
	}
	os.Setenv(EnvAzureCalicoSubnetCIDR, azureCalicoSubnet.String())
	log.Println("azureCalicoSubnet", azureCalicoSubnet.String())
}

func NextSubnet(network net.IPNet, mask int, allocatedSubnet ...net.IPNet) (*net.IPNet, error) {
	subnet, err := ipam.Free(network, net.CIDRMask(mask, 32), allocatedSubnet)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return &subnet, nil
}

func AzureCIDR() (azureSubnet *net.IPNet, err error) {
	azureCDIR := os.Getenv(EnvAzureCIDR)
	if azureCDIR == "" {
		e2eCIDR := os.Getenv(EnvE2ECIDR)

		e2eMask, err := strconv.Atoi(os.Getenv(EnvE2EMask))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		buildNumber, err := strconv.Atoi(os.Getenv(EnvCircleBuildNumber))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		_, e2eSubnet, err := net.ParseCIDR(e2eCIDR)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		ones, _ := e2eSubnet.Mask.Size()
		subnetSize := e2eMask - ones
		if subnetSize <= 0 {
			return nil, microerror.Mask(fmt.Errorf("subnet: %v, requested: %v", e2eSubnet.Mask, e2eMask))
		}
		subnetQuantity := int(math.Pow(2, float64(subnetSize)))

		subnetShift := uint(32 - e2eMask)
		subnet := int(buildNumber%subnetQuantity) << subnetShift

		networkShift := uint(32 - ones)
		network := ipToDecimal(e2eSubnet.IP) >> networkShift << networkShift

		network |= subnet

		azureSubnet = &net.IPNet{
			IP:   decimalToIP(network),
			Mask: net.CIDRMask(e2eMask, 32),
		}
	} else {
		_, azureSubnet, err = net.ParseCIDR(azureCDIR)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return azureSubnet, nil
}

// ipToDecimal converts a net.IP to an int.
// stolen from github.com/giantswarm/ipam
func ipToDecimal(ip net.IP) int {
	t := ip
	if len(ip) == 16 {
		t = ip[12:16]
	}

	return int(binary.BigEndian.Uint32(t))
}

// decimalToIP converts an int to a net.IP.
// stolen from github.com/giantswarm/ipam
func decimalToIP(ip int) net.IP {
	t := make(net.IP, 4)
	binary.BigEndian.PutUint32(t, uint32(ip))

	return t
}
