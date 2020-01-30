package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/giantswarm/azure-operator/integration/network"
)

const (
	EnvVarAzureAZs              = "AZURE_AZS"
	EnvVarAzureCIDR             = "AZURE_CIDR"
	EnvVarAzureCalicoSubnetCIDR = "AZURE_CALICO_SUBNET_CIDR"
	EnvVarAzureMasterSubnetCIDR = "AZURE_MASTER_SUBNET_CIDR"
	EnvVarAzureVPNSubnetCIDR    = "AZURE_VPN_SUBNET_CIDR"
	EnvVarAzureWorkerSubnetCIDR = "AZURE_WORKER_SUBNET_CIDR"

	EnvVarAzureClientID       = "AZURE_CLIENTID"
	EnvVarAzureClientSecret   = "AZURE_CLIENTSECRET"
	EnvVarAzureLocation       = "AZURE_LOCATION"
	EnvVarAzureSubscriptionID = "AZURE_SUBSCRIPTIONID"
	EnvVarAzureTenantID       = "AZURE_TENANTID"

	EnvVarCommonDomainResourceGroup = "COMMON_DOMAIN_RESOURCE_GROUP"
	EnvVarCircleBuildNumber         = "CIRCLE_BUILD_NUM"
)

var (
	azureClientID       string
	azureClientSecret   string
	azureLocation       string
	azureSubscriptionID string
	azureTenantID       string

	azureCIDR             string
	azureCalicoSubnetCIDR string
	azureMasterSubnetCIDR string
	azureVPNSubnetCIDR    string
	azureWorkerSubnetCIDR string

	commonDomainResourceGroup string
	sshPublicKey              string
)

func init() {
	azureClientID = os.Getenv(EnvVarAzureClientID)
	if azureClientID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureClientID))
	}

	azureClientSecret = os.Getenv(EnvVarAzureClientSecret)
	if azureClientSecret == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureClientSecret))
	}

	azureLocation = os.Getenv(EnvVarAzureLocation)
	if azureLocation == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureLocation))
	}

	azureSubscriptionID = os.Getenv(EnvVarAzureSubscriptionID)
	if azureSubscriptionID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureSubscriptionID))
	}

	azureTenantID = os.Getenv(EnvVarAzureTenantID)
	if azureTenantID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureTenantID))
	}

	commonDomainResourceGroup = os.Getenv(EnvVarCommonDomainResourceGroup)
	if commonDomainResourceGroup == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarCommonDomainResourceGroup))
	}

	sshPublicKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDBSSJCLkZWhOvs6blotU+fWbrTmC7fOwOm0+w01Ww/YN3j3j1vCrvji1A4Yonr89ePQEQKfZsYcYFodQI/D3Uzu9rOFy0dCMQfvL/J6N8LkNtmooh3J2p061829MurAdD+TVsNGrD2FZGm5Ab4NiyDXIGAYCaHL6BHP16ipBglYjLQt6jVyzdTbYspkRi1QrsNFN3gIv9V47qQSvoNEsC97gvumKzCSQ/EwJzFoIlqVkZZHZTXvGwnZrAVXB69t9Y8OJ5zA6cYFAKR0O7lEiMpebdLNGkZgMA6t2PADxfT78PHkYXLR/4tchVuOSopssJqgSs7JgIktEE14xKyNyoLKIyBBo3xwywnDySsL8R2zG4Ytw1luo79pnSpIzTvfwrNhd7Cg//OYzyDCty+XUEUQx2JfOBx5Qb1OFw71WA+zYqjbworOsy2ZZ9UAy8ryjiaeT8L2ZRGuhdicD6kkL3Lxg5UeNIxS2FLNwgepZ4D8Vo6Yxe+VOZl524ffoOJSHQ0Gz8uE76hXMNEcn4t8HVkbR4sCMgLn2YbwJ2dJcROj4w80O4qgtN1vsL16r4gt9o6euml8LbmnJz6MtGdMczSO7kHRxirtEHMTtYbT1wNgUAzimbScRggBpUz5gbz+NRE1Xgnf4A5yNMRy+JOWtLVUozJlcGSiQkVcexzdb27yQ=="

	// azureCDIR must be provided along with other CIDRs,
	// otherwise we compute CIDRs base on EnvVarCircleBuildNumber value.
	azureCDIR := os.Getenv(EnvVarAzureCIDR)
	if azureCDIR == "" {
		buildNumber, err := strconv.ParseUint(os.Getenv(EnvVarCircleBuildNumber), 10, 32)
		if err != nil {
			panic(err)
		}

		subnets, err := network.ComputeSubnets(uint(buildNumber))
		if err != nil {
			panic(err)
		}

		azureCIDR = subnets.Parent.String()
		azureCalicoSubnetCIDR = subnets.Calico.String()
		azureMasterSubnetCIDR = subnets.Master.String()
		azureVPNSubnetCIDR = subnets.VPN.String()
		azureWorkerSubnetCIDR = subnets.Worker.String()
	}
}

func AzureAvailabilityZones() []int {
	azureAvailabilityZones := os.Getenv(EnvVarAzureAZs)
	if azureAvailabilityZones == "" {
		return []int{}
	}

	azs := strings.Split(strings.TrimSpace(azureAvailabilityZones), " ")
	zones := make([]int, len(azs))

	for i, s := range azs {
		zone, err := strconv.Atoi(s)
		if err != nil {
			panic(fmt.Sprintf("AvailabilityZones valid numbers are 1, "+
				"2, 3. Your '%s' env var contains %s",
				EnvVarAzureAZs, azureAvailabilityZones))
		}
		if zone < 1 || zone > 3 {
			panic(fmt.Sprintf("AvailabilityZones valid numbers are 1, "+
				"2, 3. Your '%s' env var contains %s",
				EnvVarAzureAZs, azureAvailabilityZones))
		}
		zones[i] = zone
	}
	return zones
}

func AzureCalicoSubnetCIDR() string {
	return azureCalicoSubnetCIDR
}

func AzureClientID() string {
	return azureClientID
}

func AzureClientSecret() string {
	return azureClientSecret
}

func AzureCIDR() string {
	return azureCIDR
}

func AzureLocation() string {
	return azureLocation
}

func AzureMasterSubnetCIDR() string {
	return azureMasterSubnetCIDR
}

func AzureSubscriptionID() string {
	return azureSubscriptionID
}

func AzureTenantID() string {
	return azureTenantID
}

func AzureVPNSubnetCIDR() string {
	return azureVPNSubnetCIDR
}

func AzureWorkerSubnetCIDR() string {
	return azureWorkerSubnetCIDR
}

func CommonDomainResourceGroup() string {
	return commonDomainResourceGroup
}

func SSHPublicKey() string {
	return sshPublicKey
}
