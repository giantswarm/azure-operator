package env

import (
	"context"

	"github.com/giantswarm/azure-operator/integration/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

const (
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

	EnvVarAzureGuestClientID       = "AZURE_GUEST_CLIENTID"
	EnvVarAzureGuestClientSecret   = "AZURE_GUEST_CLIENTSECRET"
	EnvVarAzureGuestSubscriptionID = "AZURE_GUEST_SUBSCRIPTIONID"
	EnvVarAzureGuestTenantID       = "AZURE_GUEST_TENANTID"

	EnvVarCommonDomainResourceGroup = "COMMON_DOMAIN_RESOURCE_GROUP"
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

	CommonDomainResourceGroup string
}

type azureBuilderConfig struct {
	Logger micrologger.Logger

	CircleBuildNumber uint
}

type azureBuilder struct {
	logger micrologger.Logger

	circleBuildNumber uint
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

func (a *azureBuilder) Build(ctx context.Context) (Azure, error) {
	var (
		azureCIDR             string
		azureCalicoSubnetCIDR string
		azureMasterSubnetCIDR string
		azureVPNSubnetCIDR    string
		azureWorkerSubnetCIDR string

		azureClientID       string
		azureClientSecret   string
		azureLocation       string
		azureSubscriptionID string
		azureTenantID       string

		azureGuestClientID       string
		azureGuestClientSecret   string
		azureGuestSubscriptionID string
		azureGuestTenantID       string

		commonDomainResourceGroup string
	)

	err := getEnvs(
		getEnvOptional(EnvVarAzureCIDR, &azureCIDR),
		getEnvOptional(EnvVarAzureCalicoSubnetCIDR, &azureCalicoSubnetCIDR),
		getEnvOptional(EnvVarAzureMasterSubnetCIDR, &azureMasterSubnetCIDR),
		getEnvOptional(EnvVarAzureVPNSubnetCIDR, &azureVPNSubnetCIDR),
		getEnvOptional(EnvVarAzureWorkerSubnetCIDR, &azureWorkerSubnetCIDR),

		getEnvRequired(EnvVarAzureClientID, &azureClientID),
		getEnvRequired(EnvVarAzureClientSecret, &azureClientSecret),
		getEnvRequired(EnvVarAzureLocation, &azureLocation),
		getEnvRequired(EnvVarAzureSubscriptionID, &azureSubscriptionID),
		getEnvRequired(EnvVarAzureTenantID, &azureTenantID),

		getEnvRequired(EnvVarAzureGuestClientID, &azureGuestClientID),
		getEnvRequired(EnvVarAzureGuestClientSecret, &azureGuestClientSecret),
		getEnvRequired(EnvVarAzureGuestSubscriptionID, &azureGuestSubscriptionID),
		getEnvRequired(EnvVarAzureGuestTenantID, &azureGuestTenantID),

		getEnvRequired(EnvVarCommonDomainResourceGroup, commonDomainResourceGroup),
	)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}

	if azureCIDR == "" {
		if err != nil {
			return Azure{}, microerror.Mask(err)
		}

		subnets, err := network.ComputeSubnets(a.circleBuildNumber)
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

		CommonDomainResourceGroup: azureCommonDomainResourceGroup,
	}

	return azure, nil
}
