package azure

import (
	"os"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/giantswarm/microerror"
)

const (
	envVarAzureClientID       = "AZURE_CLIENTID"
	envVarAzureClientSecret   = "AZURE_CLIENTSECRET"
	envVarAzureSubscriptionID = "AZURE_SUBSCRIPTIONID"
	envVarAzureTenantID       = "AZURE_TENANTID"
)

var (
	azureClientID       string
	azureClientSecret   string
	azureSubscriptionID string
	azureTenantID       string
)

type Client struct {
	InterfacesClient                *network.InterfacesClient
	IPAddressesClient               *network.PublicIPAddressesClient
	ResourceGroupsClient            *resources.GroupsClient
	SecurityGroupsClient            *network.SecurityGroupsClient
	SecurityRulesClient             *network.SecurityRulesClient
	SubnetsClient                   *network.SubnetsClient
	VirtualMachineScaleSetsClient   *compute.VirtualMachineScaleSetsClient
	VirtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient
	VirtualMachinesClient           *compute.VirtualMachinesClient
	VirtualNetworksClient           *network.VirtualNetworksClient
}

func NewClient() (*Client, error) {
	a := &Client{}

	{
		azureClientID = os.Getenv(envVarAzureClientID)
		if azureClientID == "" {
			return nil, microerror.Maskf(invalidConfigError, "%s must be set", envVarAzureClientID)
		}

		azureClientSecret = os.Getenv(envVarAzureClientSecret)
		if azureClientSecret == "" {
			return nil, microerror.Maskf(invalidConfigError, "%s must be set", envVarAzureClientSecret)
		}

		azureSubscriptionID = os.Getenv(envVarAzureSubscriptionID)
		if azureSubscriptionID == "" {
			return nil, microerror.Maskf(invalidConfigError, "%s must be set", envVarAzureSubscriptionID)
		}

		azureTenantID = os.Getenv(envVarAzureTenantID)
		if azureTenantID == "" {
			return nil, microerror.Maskf(invalidConfigError, "%s must be set", envVarAzureTenantID)
		}

		env, err := azure.EnvironmentFromName(azure.PublicCloud.Name)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		oauthConfig, err := adal.NewOAuthConfig(env.ActiveDirectoryEndpoint, azureTenantID)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		servicePrincipalToken, err := adal.NewServicePrincipalToken(*oauthConfig, azureClientID, azureClientSecret, env.ServiceManagementEndpoint)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		virtualMachineScaleSetsClient := compute.NewVirtualMachineScaleSetsClient(azureSubscriptionID)
		virtualMachineScaleSetsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

		a.VirtualMachineScaleSetsClient = &virtualMachineScaleSetsClient

		virtualMachineScaleSetVMsClient := compute.NewVirtualMachineScaleSetVMsClient(azureSubscriptionID)
		virtualMachineScaleSetVMsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

		a.VirtualMachineScaleSetVMsClient = &virtualMachineScaleSetVMsClient

		virtualNetworksClient := network.NewVirtualNetworksClient(azureSubscriptionID)
		virtualNetworksClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

		a.VirtualNetworksClient = &virtualNetworksClient

		subnetsClient := network.NewSubnetsClient(azureSubscriptionID)
		subnetsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

		a.SubnetsClient = &subnetsClient

		securityRulesClient := network.NewSecurityRulesClient(azureSubscriptionID)
		securityRulesClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

		a.SecurityRulesClient = &securityRulesClient

		securityGroupsClient := network.NewSecurityGroupsClient(azureSubscriptionID)
		securityGroupsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

		a.SecurityGroupsClient = &securityGroupsClient

		ipAddressesClient := network.NewPublicIPAddressesClient(azureSubscriptionID)
		ipAddressesClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

		a.IPAddressesClient = &ipAddressesClient

		virtualMachinesClient := compute.NewVirtualMachinesClient(azureSubscriptionID)
		virtualMachinesClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

		a.VirtualMachinesClient = &virtualMachinesClient

		interfacesClient := network.NewInterfacesClient(azureSubscriptionID)
		interfacesClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

		a.InterfacesClient = &interfacesClient

		groupsClient := resources.NewGroupsClient(azureSubscriptionID)
		groupsClient.Authorizer = autorest.NewBearerAuthorizer(servicePrincipalToken)

		a.ResourceGroupsClient = &groupsClient
	}

	return a, nil
}
