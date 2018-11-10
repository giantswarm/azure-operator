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
	GuestLocation       string
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

func newAzureBuilder(config azureBuilderConfig) (*azureBuilder, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if config.CircleBuildNumber == 0 {
		return nil, microerror.Maskf(invalidConfigError, "%T.CircleBuildNumber must not be empty", config)
	}

	a := &azureBuilder{
		logger: config.Logger,
	}

	return a, nil
}

func (a *azureBuilder) Build(ctx context.Context) (Azure, error) {
	azureCIDR, err := getEnvVarOptional(EnvVarAzureCIDR)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureCalicoSubnetCIDR, err := getEnvVarOptional(EnvVarAzureCalicoSubnetCIDR)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureMasterSubnetCIDR, err := getEnvVarOptional(EnvVarAzureMasterSubnetCIDR)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureVPNSubnetCIDR, err := getEnvVarOptional(EnvVarAzureVPNSubnetCIDR)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureWorkerSubnetCIDR, err := getEnvVarOptional(EnvVarAzureWorkerSubnetCIDR)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}

	azureClientID, err := getEnvVarRequired(EnvVarAzureClientID)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureClientSecret, err := getEnvVarRequired(EnvVarAzureClientSecret)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureLocation, err := getEnvVarRequired(EnvVarAzureLocation)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureSubscriptionID, err := getEnvVarRequired(EnvVarAzureSubscriptionID)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureTenantID, err := getEnvVarRequired(EnvVarAzureTenantID)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}

	azureGuestClientID, err := getEnvVarRequired(EnvVarAzureGuestClientID)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureGuestClientSecret, err := getEnvVarRequired(EnvVarAzureGuestClientSecret)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureGuestSubscriptionID, err := getEnvVarRequired(EnvVarAzureGuestSubscriptionID)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}
	azureGuestTenantID, err := getEnvVarRequired(EnvVarAzureGuestTenantID)
	if err != nil {
		return Azure{}, microerror.Mask(err)
	}

	commonDomainResourceGroup, err := getEnvVarRequired(EnvVarCommonDomainResourceGroup)
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

		azureCIDR = subnets.Parent.String()
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
		GuestLocation:       azureLocation,
		GuestSubscriptionID: azureGuestSubscriptionID,
		GuestTenantID:       azureGuestTenantID,

		CommonDomainResourceGroup: commonDomainResourceGroup,
	}

	return azure, nil
}
