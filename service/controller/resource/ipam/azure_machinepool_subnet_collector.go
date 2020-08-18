package ipam

import (
	"context"
	"net"
	"sync"

	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"
	"golang.org/x/sync/errgroup"
	capzV1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/pkg/helpers"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

type AzureMachinePoolSubnetCollectorConfig struct {
	AzureClientFactory *client.Factory
	Client             ctrl.Client
}

// AzureMachinePoolSubnetCollector is a Collector implementation that collects all subnets that are
// already allocated in tenant cluster virtual network. See Collect function implementation and
// docs for more details.
type AzureMachinePoolSubnetCollector struct {
	azureClientFactory *client.Factory
	client             ctrl.Client
}

func NewAzureMachineSubnetCollector(config AzureMachinePoolSubnetCollectorConfig) (*AzureMachinePoolSubnetCollector, error) {
	if config.AzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureClientFactory must not be empty", config)
	}
	if config.Client == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Client must not be empty", config)
	}

	c := &AzureMachinePoolSubnetCollector{
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
//     tenant cluster. For existing pre-node-pool clusters, legacy subnets, if they still exist,
//     will be collected here.
func (c *AzureMachinePoolSubnetCollector) Collect(ctx context.Context, obj interface{}) ([]net.IPNet, error) {
	var err error
	var mutex sync.Mutex
	var reservedSubnets []net.IPNet

	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Get AzureCluster CR where the subnets are stored.
	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, c.client, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Get TC's VNet CIDR. We need it to check the collected subnets later, but we fetch it now, in
	// order to fail fast in case of an error.
	_, tenantClusterVNetNetworkRange, err := net.ParseCIDR(azureCluster.Spec.NetworkSpec.Vnet.CidrBlock)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	g := &errgroup.Group{}

	g.Go(func() error {
		// Collect subnets from AzureCluster CR.
		azureClusterCRSubnets, err := c.collectSubnetsFromAzureClusterCR(ctx, azureCluster)
		if err != nil {
			return microerror.Mask(err)
		}

		mutex.Lock()
		reservedSubnets = append(reservedSubnets, azureClusterCRSubnets...)
		mutex.Unlock()

		return nil
	})

	g.Go(func() error {
		// Collect subnets from the actual Azure VNet via Azure API.
		actualAzureVNetSubnets, err := c.collectSubnetsFromAzureVNet(ctx, azureCluster)
		if err != nil {
			return microerror.Mask(err)
		}

		mutex.Lock()
		reservedSubnets = append(reservedSubnets, actualAzureVNetSubnets...)
		mutex.Unlock()

		return nil
	})

	err = g.Wait()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Check if subnets actually belong to the tenant cluster VNet and filter out the duplicates.
	reservedSubnets = ipam.CanonicalizeSubnets(*tenantClusterVNetNetworkRange, reservedSubnets)

	return reservedSubnets, nil
}

// collectSubnetsFromAzureClusterCR returns all subnets specified in AzureCluster CR.
func (c *AzureMachinePoolSubnetCollector) collectSubnetsFromAzureClusterCR(_ context.Context, azureCluster *capzV1alpha3.AzureCluster) ([]net.IPNet, error) {
	azureClusterCRSubnets := make([]net.IPNet, len(azureCluster.Spec.NetworkSpec.Subnets))

	// Collect all the subnets from AzureCluster.Spec.NetworkSpec.Subnets field. If the Subnets
	// field is not set, this function will simply return nil.
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
func (c *AzureMachinePoolSubnetCollector) collectSubnetsFromAzureVNet(ctx context.Context, azureCluster *capzV1alpha3.AzureCluster) ([]net.IPNet, error) {
	// Reads "giantswarm.io/organization" label from AzureCluster CR, and then uses organization
	// name to get Azure credentials.
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

	// Not assuming VNet name here, keeping it flexible. In order to keep it correct and stable, we
	// should have a webhook for enforcing a VNet name convention.
	if azureCluster.Spec.NetworkSpec.Vnet.Name == "" {
		return nil, microerror.Maskf(invalidObjectError, "AzureCluster.Spec.NetworkSpec.Vnet.Name must be set")
	}

	vnetName := azureCluster.Spec.NetworkSpec.Vnet.Name
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

	return subnets, nil
}
