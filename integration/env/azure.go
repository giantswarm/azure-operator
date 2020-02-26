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
	EnvVarBastionPublicSSHKey       = "BASTION_PUBLIC_SSH_KEY"

	EnvVarCircleBuildNumber = "CIRCLE_BUILD_NUM"
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

	var ok bool
	sshPublicKey, ok = os.LookupEnv(EnvVarBastionPublicSSHKey)
	if !ok {
		fmt.Printf("No public SSH key found in '%s': no keys will be placed on the bastion server", EnvVarBastionPublicSSHKey)
	}

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
