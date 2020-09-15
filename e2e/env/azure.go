package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/giantswarm/azure-operator/v4/e2e/network"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
)

const (
	DefaultAzureLocation             = "westeurope"
	DefaultAzureVMSize               = "Standard_D4_v2"
	DefaultCommonDomainResourceGroup = "godsmack"

	EnvVarAzureAZs  = "AZURE_AZS"
	EnvVarAzureCIDR = "AZURE_CIDR"

	EnvVarAzureClientID       = "AZURE_CLIENTID"
	EnvVarAzureClientSecret   = "AZURE_CLIENTSECRET" // #nosec
	EnvVarAzureLocation       = "AZURE_LOCATION"
	EnvVarAzureSubscriptionID = "AZURE_SUBSCRIPTIONID"
	EnvVarAzureTenantID       = "AZURE_TENANTID"
	EnvVarAzureVMSize         = "AZURE_VMSIZE"

	EnvVarCommonDomainResourceGroup = "COMMON_DOMAIN_RESOURCE_GROUP"
	EnvVarBastionPublicSSHKey       = "BASTION_PUBLIC_SSH_KEY"

	EnvVarCircleBuildNumber = "CIRCLE_BUILD_NUM"

	EnvVarLatestOperatorRelease = "LATEST_OPERATOR_RELEASE"
)

var (
	azureClientID       string
	azureClientSecret   string
	azureLocation       string
	azureSubscriptionID string
	azureTenantID       string
	azureVMSize         string

	azureCIDR             string
	azureCalicoSubnetCIDR string
	azureMasterSubnetCIDR string
	azureWorkerSubnetCIDR string

	commonDomainResourceGroup string
	sshPublicKey              string

	latestOperatorRelease string
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
		azureLocation = DefaultAzureLocation
		fmt.Printf("No value found in '%s': using default value %s\n", EnvVarAzureLocation, DefaultAzureLocation)
	}

	azureSubscriptionID = os.Getenv(EnvVarAzureSubscriptionID)
	if azureSubscriptionID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureSubscriptionID))
	}

	azureTenantID = os.Getenv(EnvVarAzureTenantID)
	if azureTenantID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureTenantID))
	}

	azureVMSize = os.Getenv(EnvVarAzureVMSize)
	if azureVMSize == "" {
		fmt.Printf("No value found in '%s': using default value %s\n", EnvVarAzureVMSize, DefaultAzureVMSize)
		azureVMSize = DefaultAzureVMSize
	}

	commonDomainResourceGroup = os.Getenv(EnvVarCommonDomainResourceGroup)
	if commonDomainResourceGroup == "" {
		commonDomainResourceGroup = DefaultCommonDomainResourceGroup
		fmt.Printf("No value found in '%s': using default value %s\n", EnvVarCommonDomainResourceGroup, DefaultCommonDomainResourceGroup)
	}

	var ok bool
	sshPublicKey, ok = os.LookupEnv(EnvVarBastionPublicSSHKey)
	if !ok {
		fmt.Printf("No value found in '%s': default public key will be placed on the bastion server\n", EnvVarBastionPublicSSHKey)
		sshPublicKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDBSSJCLkZWhOvs6blotU+fWbrTmC7fOwOm0+w01Ww/YN3j3j1vCrvji1A4Yonr89ePQEQKfZsYcYFodQI/D3Uzu9rOFy0dCMQfvL/J6N8LkNtmooh3J2p061829MurAdD+TVsNGrD2FZGm5Ab4NiyDXIGAYCaHL6BHP16ipBglYjLQt6jVyzdTbYspkRi1QrsNFN3gIv9V47qQSvoNEsC97gvumKzCSQ/EwJzFoIlqVkZZHZTXvGwnZrAVXB69t9Y8OJ5zA6cYFAKR0O7lEiMpebdLNGkZgMA6t2PADxfT78PHkYXLR/4tchVuOSopssJqgSs7JgIktEE14xKyNyoLKIyBBo3xwywnDySsL8R2zG4Ytw1luo79pnSpIzTvfwrNhd7Cg//OYzyDCty+XUEUQx2JfOBx5Qb1OFw71WA+zYqjbworOsy2ZZ9UAy8ryjiaeT8L2ZRGuhdicD6kkL3Lxg5UeNIxS2FLNwgepZ4D8Vo6Yxe+VOZl524ffoOJSHQ0Gz8uE76hXMNEcn4t8HVkbR4sCMgLn2YbwJ2dJcROj4w80O4qgtN1vsL16r4gt9o6euml8LbmnJz6MtGdMczSO7kHRxirtEHMTtYbT1wNgUAzimbScRggBpUz5gbz+NRE1Xgnf4A5yNMRy+JOWtLVUozJlcGSiQkVcexzdb27yQ=="
	}

	// azureCDIR must be provided along with other CIDRs,
	// otherwise we compute CIDRs base on EnvVarCircleBuildNumber value.
	azureCDIR := os.Getenv(EnvVarAzureCIDR)
	if azureCDIR == "" {
		circleCIBuildNumber, ok := os.LookupEnv(EnvVarCircleBuildNumber)
		if !ok {
			circleCIBuildNumber = "1"
		}
		buildNumber, err := strconv.ParseUint(circleCIBuildNumber, 10, 32)
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
		azureWorkerSubnetCIDR = subnets.Worker.String()
	}

	var exists bool
	latestOperatorRelease, exists = os.LookupEnv(EnvVarLatestOperatorRelease)
	if !exists {
		panic(fmt.Sprintf("env var %#q must not be empty\n", EnvVarLatestOperatorRelease))
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

func AzureAvailabilityZonesAsStrings() []string {
	var azs []string

	for _, azInt := range AzureAvailabilityZones() {
		az := strconv.Itoa(azInt)
		azs = append(azs, az)
	}

	return azs
}

// AzureAvailabilityZonesCount returns expected number of availability zones for the cluster.
func AzureAvailabilityZonesCount() int {
	specifiedZones := AzureAvailabilityZones()
	specifiedCount := len(specifiedZones)

	switch specifiedCount {
	case 0:
		return 1
	case 1, 2, 3:
		return specifiedCount
	default:
		panic("AvailabilityZones valid numbers are 1, 2, 3.")
	}
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

func AzureVMSize() string {
	return azureVMSize
}

func AzureWorkerSubnetCIDR() string {
	return azureWorkerSubnetCIDR
}

func CommonDomainResourceGroup() string {
	return commonDomainResourceGroup
}

func GetLatestOperatorRelease() string {
	return latestOperatorRelease
}

func GetOperatorVersion() string {
	var operatorVersion string
	{
		// `operatorVersion` is the link between an operator and a `CustomResource`.
		// azure-operator with version `operatorVersion` will only reconcile `AzureConfig` labeled with `operatorVersion`.
		operatorVersion = project.Version()
		if TestDir() == "e2e/test/update" {
			// When testing the update process, we want the latest release of the operator to reconcile the `CustomResource` and create a cluster.
			// We can then update the label in the `CustomResource`, making the operator under test to reconcile it and update the cluster.
			operatorVersion = GetLatestOperatorRelease()
		}
	}

	return operatorVersion
}

func SSHPublicKey() string {
	return sshPublicKey
}
