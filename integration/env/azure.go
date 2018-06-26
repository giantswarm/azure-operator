package env

import (
	"fmt"
	"os"
	"strconv"

	"github.com/giantswarm/azure-operator/integration/network"
)

const (
	EnvVarAzureCIDR             = "AZURE_CIDR"
	EnvVarAzureCalicoSubnetCIDR = "AZURE_CALICO_SUBNET_CIDR"
	EnvVarAzureMasterSubnetCIDR = "AZURE_MASTER_SUBNET_CIDR"
	EnvVarAzureVPNSubnetCIDR    = "AZURE_VPN_SUBNET_CIDR"
	EnvVarAzureWorkerSubnetCIDR = "AZURE_WORKER_SUBNET_CIDR"

	EnvVarAzureClientID       = "AZURE_CLIENTID"
	EnvVarAzureClientSecret   = "AZURE_CLIENTSECRET"
	EnvVarAzureSubscriptionID = "AZURE_SUBSCRIPTIONID"
	EnvVarAzureTenantID       = "AZURE_TENANTID"

	EnvVarAzureGuestClientID       = "AZURE_GUEST_CLIENTID"
	EnvVarAzureGuestClientSecret   = "AZURE_GUEST_CLIENTSECRET"
	EnvVarAzureGuestSubscriptionID = "AZURE_GUEST_SUBSCRIPTIONID"
	EnvVarAzureGuestTenantID       = "AZURE_GUEST_TENANTID"

	EnvVarCircleBuildNumber = "CIRCLE_BUILD_NUM"
)

var (
	azureClientID       string
	azureClientSecret   string
	azureSubscriptionID string
	azureTenantID       string

	azureGuestClientID       string
	azureGuestClientSecret   string
	azureGuestSubscriptionID string
	azureGuestTenantID       string
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

	azureSubscriptionID = os.Getenv(EnvVarAzureSubscriptionID)
	if azureSubscriptionID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureSubscriptionID))
	}

	azureTenantID = os.Getenv(EnvVarAzureTenantID)
	if azureTenantID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureTenantID))
	}

	azureGuestClientID = os.Getenv(EnvVarAzureGuestClientID)
	if azureGuestClientID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureGuestClientID))
	}

	azureGuestClientSecret = os.Getenv(EnvVarAzureGuestClientSecret)
	if azureGuestClientSecret == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureGuestClientSecret))
	}

	azureGuestSubscriptionID = os.Getenv(EnvVarAzureGuestSubscriptionID)
	if azureGuestSubscriptionID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureGuestSubscriptionID))
	}

	azureGuestTenantID = os.Getenv(EnvVarAzureGuestTenantID)
	if azureGuestTenantID == "" {
		panic(fmt.Sprintf("env var '%s' must not be empty", EnvVarAzureGuestTenantID))
	}

	// azureCDIR must be provided along with other CIDRs,
	// otherwise we compute CIDRs base on EnvVarCircleBuildNumber value.
	azureCDIR := os.Getenv(EnvVarAzureCIDR)
	if azureCDIR == "" {
		buildNumber, err := strconv.ParseUint(os.Getenv(EnvVarCircleBuildNumber), 10, 32)
		if err != nil {
			panic(err)
		}

		cidrs, err := network.ComputeCIDR(uint(buildNumber))
		if err != nil {
			panic(err)
		}

		os.Setenv(EnvVarAzureCIDR, cidrs.AzureCIDR)
		os.Setenv(EnvVarAzureMasterSubnetCIDR, cidrs.MasterSubnetCIDR)
		os.Setenv(EnvVarAzureVPNSubnetCIDR, cidrs.VPNSubnetCIDR)
		os.Setenv(EnvVarAzureWorkerSubnetCIDR, cidrs.WorkerSubnetCIDR)
		os.Setenv(EnvVarAzureCalicoSubnetCIDR, cidrs.CalicoSubnetCIDR)
	} else {
		if os.Getenv(EnvVarAzureCalicoSubnetCIDR) == "" {
			panic(fmt.Sprintf("env var '%s' must not be empty when AZURE_CIDR is set", EnvVarAzureCalicoSubnetCIDR))
		}
		if os.Getenv(EnvVarAzureMasterSubnetCIDR) == "" {
			panic(fmt.Sprintf("env var '%s' must not be empty when AZURE_CIDR is set", EnvVarAzureMasterSubnetCIDR))
		}
		if os.Getenv(EnvVarAzureVPNSubnetCIDR) == "" {
			panic(fmt.Sprintf("env var '%s' must not be empty when AZURE_CIDR is set", EnvVarAzureVPNSubnetCIDR))
		}
		if os.Getenv(EnvVarAzureWorkerSubnetCIDR) == "" {
			panic(fmt.Sprintf("env var '%s' must not be empty when AZURE_CIDR is set", EnvVarAzureWorkerSubnetCIDR))
		}
	}
}

func AzureClientID() string {
	return azureClientID
}

func AzureClientSecret() string {
	return azureClientSecret
}

func AzureSubscriptionID() string {
	return azureSubscriptionID
}

func AzureTenantID() string {
	return azureTenantID
}

func AzureGuestClientID() string {
	return azureGuestClientID
}

func AzureGuestClientSecret() string {
	return azureGuestClientSecret
}

func AzureGuestSubscriptionID() string {
	return azureGuestSubscriptionID
}

func AzureGuestTenantID() string {
	return azureGuestTenantID
}
