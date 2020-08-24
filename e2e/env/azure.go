package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/giantswarm/employees/pkg/employees"

	"github.com/giantswarm/azure-operator/v4/e2e/network"
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
		employeesSet, err := employees.NewEmployeesSet()
		if err != nil {
			panic("Error trying to read employees SSH keys")
		}

		sshPublicKey = strings.Join(employeesSet.GetAllKeys(), "\n")
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

func SSHPublicKey() string {
	return sshPublicKey
}
