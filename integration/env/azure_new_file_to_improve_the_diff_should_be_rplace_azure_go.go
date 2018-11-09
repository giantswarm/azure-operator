package env

import (
	"os"
	"strconv"

	"github.com/giantswarm/azure-operator/integration/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

var (
	// TODO Consider if we need to pass it at all.
	azureCIDR = getEnv("AZURE_CIDR")
	// TODO Why do we still need all the subnets below?
	azureCalicoSubnetCIDR = mustGetEnv("AZURE_CALICO_SUBNET_CIDR")
	azureMasterSubnetCIDR = mustGetEnv("AZURE_MASTER_SUBNET_CIDR")
	azureVPNSubnetCIDR    = mustGetEnv("AZURE_VPN_SUBNET_CIDR")
	azureWorkerSubnetCIDR = mustGetEnv("AZURE_WORKER_SUBNET_CIDR")

	azureClientID       = mustGetEnv("AZURE_CLIENTID")
	azureClientSecret   = mustGetEnv("AZURE_CLIENTSECRET")
	azureLocation       = mustGetEnv("AZURE_LOCATION")
	azureSubscriptionID = mustGetEnv("AZURE_SUBSCRIPTIONID")
	azureTenantID       = mustGetEnv("AZURE_TENANTID")

	azureGuestClientID       = mustGetEnv("AZURE_GUEST_CLIENTID")
	azureGuestClientSecret   = mustGetEnv("AZURE_GUEST_CLIENTSECRET")
	azureGuestSubscriptionID = mustGetEnv("AZURE_GUEST_SUBSCRIPTIONID")
	azureGuestTenantID       = mustGetEnv("AZURE_GUEST_TENANTID")

	commonDomainResourceGroup = mustGetEnv("COMMON_DOMAIN_RESOURCE_GROUP")
)

type Azure struct {
	CIDR             string
	CalicoSubnetCIDR string
	MasterSubnetCIDR string
	VPNSubnetCIDR    string
	WorkerSubnetCIDR string

	ClientID       string
	ClientSecret   string
	Location       string
	SubscriptionID string
	TenantID       string

	GuestClientID       string
	GuestClientSecret   string
	GuestSubscriptionID string
	GuestTenantID       string

	commonDomainResourceGroup string
}

type azureBuilderConfig struct {
	Logger micrologger.Logger

	CircleBuildNumber int
}

type azureBuilder struct {
	logger micrologger.Logger

	circleBuildNumber int
}

func newAzureBuilder(config Config) (*azureBuilder, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.CircleBuildNumber == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.CircleBuildNumber must not be empty", config)
	}

	a := &azureBuilder{
		logger: config.Logger,
	}
}

func (a *azureBuilder) Build() (Azure, error) {
	if azureCIDR == "" {
		buildNumber, err := strconv.ParseUint(os.Getenv(EnvVarCircleBuildNumber), 10, 32)
		if err != nil {
			return Azure{}, microerror.Mask(err)
		}

		subnets, err := network.ComputeSubnets(uint(buildNumber))
		if err != nil {
			return Azure{}, microerror.Mask(err)
		}

		azureCidr = subnets.Parent.String()
		azureCalicoSubnetCIDR = subnets.Calico.String()
		azureMasterSubnetCIDR = subnets.Master.String()
		azureVPNSubnetCIDR = subnets.VPN.String()
		azureWorkerSubnetCIDR = subnets.Worker.String()
	}

	azure := Azure{
		CIDR:             azureCIDR,
		CalicoSubnetCIDR: azureCalicoSubnetCIDR,
		MasterSubnetCIDR: azureMasterSubnetCIDR,
		VPNSubnetCIDR:    azureVPNSubnetCIDR,
		WorkerSubnetCIDR: azureWorkerSubnetCIDR,

		ClientID:       azureClientID,
		ClientSecret:   azureClientSecret,
		Location:       azureLocation,
		SubscriptionID: azureSubscriptionID,
		TenantID:       azureTenantID,

		GuestClientID:       azureGuestClientID,
		GuestClientSecret:   azureGuestClientSecret,
		GuestLocation:       azureGuestLocation,
		GuestSubscriptionID: azureGuestSubscriptionID,
		GuestTenantID:       azureGuestTenantID,
	}

	return azure, nil
}
