package ipam

import (
	"context"
	"net"

	"github.com/giantswarm/microerror"
	capzV1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/pkg/helpers"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

type AzureClusterSubnetCollectorConfig struct {
	AzureClientFactory *client.Factory
	Client             ctrl.Client
}

// AzureClusterSubnetCollector is a Collector implementation that collects all subnets that are
// already allocated in tenant cluster virtual network. See Collect function implementation and
// docs for more details.
type AzureClusterSubnetCollector struct {
	azureClientFactory *client.Factory
	client             ctrl.Client
}

func NewAzureClusterSubnetCollector(config AzureClusterSubnetCollectorConfig) (*AzureClusterSubnetCollector, error) {
	if config.AzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureClientFactory must not be empty", config)
	}
	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	c := &AzureClusterSubnetCollector{
		azureClientFactory: config.AzureClientFactory,
		client:             config.Client,
	}

	return c, nil
}

// Collect function returns all subnets that are already allocated in tenant cluster virtual
// network. These include subnets set in AzureCluster CR and all subnets that are created in tenant
// cluster's Azure virtual network.
//
// Why getting both of these?
//   - Subnets from AzureCluster CR: these are desired subnets for the tenant cluster, they might
//     be already deployed in Azure or not.
//   - Subnets in Azure virtual network: In addition to subnets from AzureCluster CR that should be
//     eventually deployed here, there might be some other subnets that are created outside of
//     tenant cluster.
func (c *AzureClusterSubnetCollector) Collect(ctx context.Context, obj interface{}) ([]net.IPNet, error) {
	var err error
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Get AzureCluster CR where the subnets are stored.
	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, c.client, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var azureClusterCRSubnets []net.IPNet
	{
		// Collect subnets from AzureCluster CR.
		azureClusterCRSubnets, err = c.collectSubnetsFromAzureClusterCR(ctx, azureCluster)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var actualAzureVNetSubnets []net.IPNet
	{
		// Collect subnets from the actual Azure VNet via Azure API.
		actualAzureVNetSubnets, err = c.collectSubnetsFromAzureVNet(ctx, azureCluster)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	allocatedSubnets := append(azureClusterCRSubnets, actualAzureVNetSubnets...)
	return allocatedSubnets, nil
}

// collectSubnetsFromAzureClusterCR returns all subnets specified in AzureCluster CR.
func (c *AzureClusterSubnetCollector) collectSubnetsFromAzureClusterCR(_ context.Context, azureCluster *capzV1alpha3.AzureCluster) ([]net.IPNet, error) {
	azureClusterCRSubnets := make([]net.IPNet, len(azureCluster.Spec.NetworkSpec.Subnets))

	// TODO: add check for .Spec.NetworkSpec.Subnets field
	for _, subnet := range azureCluster.Spec.NetworkSpec.Subnets {
		_, subnetIPNet, err := net.ParseCIDR(subnet.CidrBlock)
		if err != nil {
			return nil, microerror.Mask(err)
		}
		azureClusterCRSubnets = append(azureClusterCRSubnets, *subnetIPNet)
	}

	return azureClusterCRSubnets, nil
}

// collectSubnetsFromAzureVNet returns all subnets that are deployed in Azure virtual network.
func (c *AzureClusterSubnetCollector) collectSubnetsFromAzureVNet(ctx context.Context, azureCluster *capzV1alpha3.AzureCluster) ([]net.IPNet, error) {
	// TODO: add to docs that "giantswarm.io/organization" must be set on AzureCluster
	credentials, err := helpers.GetCredentialSecretFromMetadata(ctx, c.client, azureCluster.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	subnetsClient, err := c.azureClientFactory.GetSubnetsClient(credentials.Namespace, credentials.Name)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// cluster ID is tenant cluster resource group name
	resourceGroupName := azureCluster.Name
	vnetName := azureCluster.Spec.NetworkSpec.Vnet.Name // TODO: check if .Spec.NetworkSpec.Vnet.Name is set
	resultPage, err := subnetsClient.List(ctx, resourceGroupName, vnetName)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var subnets []net.IPNet

	for resultPage.NotDone() {
		azureSubnets := resultPage.Values()

		for _, azureSubnet := range azureSubnets {
			_, subnetIPNet, err := net.ParseCIDR(*azureSubnet.AddressPrefix)
			if err != nil {
				return nil, microerror.Mask(err)
			}
			subnets = append(subnets, *subnetIPNet)
		}

		err = resultPage.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return nil, nil
}
