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
	EnvCircleBuildNumber = "CIRCLE_BUILD_NUM"
	EnvE2ECIDR           = "E2E_CIDR"
	EnvE2EMask           = "E2E_MASK"

	EnvAzureCIDR             = "AZURE_CIDR"
	EnvAzureCalicoSubnetCIDR = "AZURE_CALICO_SUBNET_CIDR"
	EnvAzureMasterSubnetCIDR = "AZURE_MASTER_SUBNET_CIDR"
	EnvAzureWorkerSubnetCIDR = "AZURE_WORKER_SUBNET_CIDR"

	AzureCalicoSubnetMask = 17
	AzureMasterSubnetMask = 24
	AzureWorkerSubnetMask = 24
)

func init() {
	var err error

	var azureNetwork *net.IPNet
	{
		azureNetwork, err = AzureNetwork()
		if err != nil {
			panic(err)
		}
		os.Setenv(EnvAzureCIDR, azureNetwork.String())
		log.Println("azureCIDR", azureNetwork.String())
	}

	var azureMasterSubnet *net.IPNet
	{
		azureMasterSubnet, err = AzureSubnet(EnvAzureMasterSubnetCIDR, *azureNetwork, AzureMasterSubnetMask)
		if err != nil {
			panic(err)
		}
		os.Setenv(EnvAzureMasterSubnetCIDR, azureMasterSubnet.String())
		log.Println("azureMasterCIDR", azureMasterSubnet.String())
	}

	var azureWorkerSubnet *net.IPNet
	{
		azureWorkerSubnet, err = AzureSubnet(EnvAzureWorkerSubnetCIDR, *azureNetwork, AzureWorkerSubnetMask, *azureMasterSubnet)
		if err != nil {
			panic(err)
		}
		os.Setenv(EnvAzureWorkerSubnetCIDR, azureWorkerSubnet.String())
		log.Println("azureWorkerCIDR", azureWorkerSubnet.String())
	}

	var azureCalicoSubnet *net.IPNet
	{
		azureCalicoSubnet, err = AzureSubnet(EnvAzureCalicoSubnetCIDR, *azureNetwork, AzureCalicoSubnetMask, *azureMasterSubnet, *azureWorkerSubnet)
		if err != nil {
			panic(err)
		}
		os.Setenv(EnvAzureCalicoSubnetCIDR, azureCalicoSubnet.String())
		log.Println("azureCalicoSubnet", azureCalicoSubnet.String())
	}
}

// AzureNetwork either return network from CIDR found at EnvAzureCIDR environement variable or
// determine network using EnvE2ECIDR, EnvE2EMask and EnvCircleBuildNumber.
func AzureNetwork() (azureNetwork *net.IPNet, err error) {
	azureCDIR := os.Getenv(EnvAzureCIDR)
	if azureCDIR == "" {
		e2eCIDR := os.Getenv(EnvE2ECIDR)

		e2eMask, err := strconv.ParseUint(os.Getenv(EnvE2EMask), 10, 32)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		buildNumber, err := strconv.ParseUint(os.Getenv(EnvCircleBuildNumber), 10, 32)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		azureNetwork, err = DetermineSubnet(e2eCIDR, uint(e2eMask), uint(buildNumber))
		if err != nil {
			return nil, microerror.Mask(err)
		}
	} else {
		_, azureNetwork, err = net.ParseCIDR(azureCDIR)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return azureNetwork, nil
}

// AzureSubnet either return subnet from CIDR found at envVar environement variable or
// compute the next free subnet using network, mask and allocatedSubnet (if any).
func AzureSubnet(envVar string, network net.IPNet, mask int, allocatedSubnet ...net.IPNet) (subnet *net.IPNet, err error) {
	subnetCIDR := os.Getenv(envVar)
	if subnetCIDR == "" {
		s, err := ipam.Free(network, net.CIDRMask(mask, 32), allocatedSubnet)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		subnet = &s
	} else {
		_, subnet, err = net.ParseCIDR(subnetCIDR)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return subnet, nil
}

// DetermineSubnet is deterministic, it will always return the same network given the same input.
// It compute the number of available networks between cidr and mask.
// And use `decider mod available networks` to determine which one to pick.
// e.g. DetermineSubnet("10.255.0.0/9", 16, 42)  >  10.170.0.0/16
func DetermineSubnet(cidr string, mask, decider uint) (*net.IPNet, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	ones, _ := network.Mask.Size()
	subnetSize := mask - uint(ones)
	if subnetSize <= 0 {
		return nil, microerror.Mask(fmt.Errorf("subnet: %v, requested: %v", network.Mask, mask))
	}
	subnetQuantity := int(math.Pow(2, float64(subnetSize)))

	subnet := int(decider % uint(subnetQuantity))
	// uncomment following line to start
	// subnet = possibilities - 1 - subnet
	subnetShift := uint(32 - mask)
	subnet <<= subnetShift

	ip := ipToDecimal(network.IP)
	ipShift := uint(32 - ones)
	ip >>= ipShift
	ip <<= ipShift

	ip |= subnet

	return &net.IPNet{
		IP:   decimalToIP(ip),
		Mask: net.CIDRMask(int(mask), 32),
	}, nil
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
