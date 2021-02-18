package ipam

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"golang.org/x/sync/errgroup"
	capzV1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/client"
	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

type AzureMachinePoolSubnetCollectorConfig struct {
	WCAzureClientFactory client.CredentialsAwareClientFactoryInterface
	CtrlClient           ctrl.Client
	Logger               micrologger.Logger
}

// AzureMachinePoolSubnetCollector is a Collector implementation that collects all subnets that are
// already allocated in tenant cluster virtual network. See Collect function implementation and
// docs for more details.
type AzureMachinePoolSubnetCollector struct {
	wcAzureClientFactory client.CredentialsAwareClientFactoryInterface
	ctrlClient           ctrl.Client
	logger               micrologger.Logger
}

func NewAzureMachineSubnetCollector(config AzureMachinePoolSubnetCollectorConfig) (*AzureMachinePoolSubnetCollector, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	c := &AzureMachinePoolSubnetCollector{
		wcAzureClientFactory: config.WCAzureClientFactory,
		ctrlClient:           config.CtrlClient,
		logger:               config.Logger,
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
	azureCluster, err := helpers.GetAzureClusterFromMetadata(ctx, c.ctrlClient, azureMachinePool.ObjectMeta)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	if len(azureCluster.Spec.NetworkSpec.Vnet.CIDRBlocks) == 0 {
		// This can happen when the VNet for the tenant cluster is still not allocated (e.g. when
		// the cluster is still being created).
		errorMessage := "AzureCluster.Spec.NetworkSpec.Vnet.CIDRBlocks is not set yet"
		c.logger.LogCtx(ctx, "level", "warning", "message", errorMessage)
		return nil, microerror.Maskf(parentNetworkRangeStillNotKnown, errorMessage)
	}

	// Get TC's VNet CIDR. We need it to check the collected subnets later, but we fetch it now, in
	// order to fail fast in case of an error.
	_, tenantClusterVNetNetworkRange, err := net.ParseCIDR(azureCluster.Spec.NetworkSpec.Vnet.CIDRBlocks[0])
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
func (c *AzureMachinePoolSubnetCollector) collectSubnetsFromAzureClusterCR(ctx context.Context, azureCluster *capzV1alpha3.AzureCluster) ([]net.IPNet, error) {
	c.logger.Debugf(ctx, "finding allocated subnets in AzureCluster CR")
	azureClusterCRSubnets := make([]net.IPNet, len(azureCluster.Spec.NetworkSpec.Subnets))

	// Collect all the subnets from AzureCluster.Spec.NetworkSpec.Subnets field. If the Subnets
	// field is not set, this function will simply return nil.
	for _, subnet := range azureCluster.Spec.NetworkSpec.Subnets {
		if len(subnet.CIDRBlocks) == 0 {
			continue
		}

		_, subnetIPNet, err := net.ParseCIDR(subnet.CIDRBlocks[0])
		if err != nil {
			return nil, microerror.Mask(err)
		}
		azureClusterCRSubnets = append(azureClusterCRSubnets, *subnetIPNet)
	}

	c.logger.Debugf(ctx, "found %d allocated subnets in AzureCluster CR", len(azureClusterCRSubnets))
	return azureClusterCRSubnets, nil
}

// collectSubnetsFromAzureVNet returns all subnets that are deployed in Azure virtual network.
func (c *AzureMachinePoolSubnetCollector) collectSubnetsFromAzureVNet(ctx context.Context, azureCluster *capzV1alpha3.AzureCluster) ([]net.IPNet, error) {
	// Not assuming VNet name here, keeping it flexible. In order to keep it correct and stable, we
	// should have a webhook for enforcing a VNet name convention.
	if azureCluster.Spec.NetworkSpec.Vnet.Name == "" {
		errorMessage := "AzureCluster.Spec.NetworkSpec.Vnet.Name is not set"
		c.logger.LogCtx(ctx, "level", "warning", "message", errorMessage)
		return nil, microerror.Maskf(invalidObjectError, errorMessage)
	}

	c.logger.LogCtx(
		ctx,
		"level", "debug",
		"message", fmt.Sprintf("finding subnets created in Azure VNet %q", azureCluster.Spec.NetworkSpec.Vnet.Name))

	subnetsClient, err := c.wcAzureClientFactory.GetSubnetsClient(ctx, key.ClusterID(azureCluster))
	if err != nil {
		errorMessage := fmt.Sprintf("error while creating/getting Azure subnets client for cluster %q", azureCluster.Name)
		c.logger.LogCtx(ctx, "level", "warning", "message", errorMessage)
		return nil, microerror.Mask(err)
	}

	// cluster ID is tenant cluster resource group name
	resourceGroupName := azureCluster.Name
	vnetName := azureCluster.Spec.NetworkSpec.Vnet.Name
	resultPage, err := subnetsClient.List(ctx, resourceGroupName, vnetName)
	if err != nil {
		var errorMessage string
		if IsNotFound(err) {
			errorMessage = fmt.Sprintf("Azure VNet %q is not found", vnetName)
		} else {
			errorMessage = fmt.Sprintf("error while getting Azure subnets from VNet %q", vnetName)
		}
		c.logger.LogCtx(ctx, "level", "warning", "message", errorMessage)
		return nil, microerror.Mask(err)
	}

	var subnets []net.IPNet

	for resultPage.NotDone() {
		azureSubnets := resultPage.Values()

		for _, azureSubnet := range azureSubnets {
			_, subnetIPNet, err := net.ParseCIDR(*azureSubnet.AddressPrefix)
			if err != nil {
				errorMessage := fmt.Sprintf("error while parsing Azure subnet range %q", *azureSubnet.AddressPrefix)
				c.logger.LogCtx(ctx, "level", "warning", "message", errorMessage)
				return nil, microerror.Mask(err)
			}
			subnets = append(subnets, *subnetIPNet)
		}

		err = resultPage.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	c.logger.LogCtx(
		ctx,
		"level", "debug",
		"message", fmt.Sprintf("found subnets created in Azure VNet %q", azureCluster.Spec.NetworkSpec.Vnet.Name))

	return subnets, nil
}
