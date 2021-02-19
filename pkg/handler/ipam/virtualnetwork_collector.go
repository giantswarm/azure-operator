package ipam

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/ipam"
	"github.com/giantswarm/k8sclient/v5/pkg/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/azureclient/credentialsawarefactory"
	"github.com/giantswarm/azure-operator/v5/service/collector"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

type VirtualNetworkCollectorConfig struct {
	AzureMetricsCollector collector.AzureAPIMetrics
	WCAzureClientFactory  credentialsawarefactory.Interface
	InstallationName      string
	K8sClient             k8sclient.Interface
	Logger                micrologger.Logger

	NetworkRange  net.IPNet
	ReservedCIDRs []net.IPNet
}

type VirtualNetworkCollector struct {
	azureMetricsCollector collector.AzureAPIMetrics
	wcAzureClientFactory  credentialsawarefactory.Interface
	installationName      string
	k8sclient             k8sclient.Interface
	logger                micrologger.Logger

	networkRange  net.IPNet
	reservedCIDRs []net.IPNet
}

func NewVirtualNetworkCollector(config VirtualNetworkCollectorConfig) (*VirtualNetworkCollector, error) {
	if config.AzureMetricsCollector == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureMetricsCollector must not be empty", config)
	}
	if config.WCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientFactory must not be empty", config)
	}
	if config.K8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.K8sClient must not be empty", config)
	}
	if config.InstallationName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstallationName must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if reflect.DeepEqual(config.NetworkRange, net.IPNet{}) {
		return nil, microerror.Maskf(invalidConfigError, "%T.NetworkRange must not be empty", config)
	}

	c := &VirtualNetworkCollector{
		azureMetricsCollector: config.AzureMetricsCollector,
		wcAzureClientFactory:  config.WCAzureClientFactory,
		k8sclient:             config.K8sClient,
		installationName:      config.InstallationName,
		logger:                config.Logger,

		networkRange:  config.NetworkRange,
		reservedCIDRs: config.ReservedCIDRs,
	}

	return c, nil
}

func (c *VirtualNetworkCollector) Collect(ctx context.Context, _ interface{}) ([]net.IPNet, error) {
	var err error
	var mutex sync.Mutex
	var reservedVirtualNetworks []net.IPNet
	reservedVirtualNetworks = append(reservedVirtualNetworks, c.reservedCIDRs...)

	g := &errgroup.Group{}

	g.Go(func() error {
		c.logger.Debugf(ctx, "finding allocated virtual networks from AzureConfig CRs")

		virtualNetworks, err := c.getVirtualNetworksFromAzureConfigs(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
		mutex.Lock()
		reservedVirtualNetworks = append(reservedVirtualNetworks, virtualNetworks...)
		mutex.Unlock()

		c.logger.Debugf(ctx, "found allocated virtual networks from AzureConfig CRs")

		return nil
	})

	g.Go(func() error {
		c.logger.Debugf(ctx, "finding allocated virtual networks from all resource groups in the subscription")

		virtualNetworks, err := c.getVirtualNetworksFromAllSubscriptions(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
		mutex.Lock()
		reservedVirtualNetworks = append(reservedVirtualNetworks, virtualNetworks...)
		mutex.Unlock()

		c.logger.Debugf(ctx, "found allocated virtual networks from all resource groups in the subscription")

		return nil
	})

	err = g.Wait()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	reservedVirtualNetworks = ipam.CanonicalizeSubnets(c.networkRange, reservedVirtualNetworks)

	return reservedVirtualNetworks, nil
}

func (c *VirtualNetworkCollector) getVirtualNetworksFromAzureConfigs(ctx context.Context) ([]net.IPNet, error) {
	tenantClusterList, err := c.getAllTenantClusters(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var results []net.IPNet
	for _, ac := range tenantClusterList.Items {
		cidr := key.AzureConfigNetworkCIDR(ac)
		if cidr == "" {
			continue
		}

		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		results = append(results, *n)
	}

	return results, nil
}

func (c *VirtualNetworkCollector) getVirtualNetworksFromAllSubscriptions(ctx context.Context) ([]net.IPNet, error) {
	tenantClusterList, err := c.getAllTenantClusters(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var doneSubscriptions []string
	var ret []net.IPNet
	for _, cluster := range tenantClusterList.Items {
		subscriptionID, err := c.wcAzureClientFactory.GetSubscriptionID(ctx, key.ClusterID(&cluster))
		if err != nil {
			return nil, microerror.Mask(err)
		}

		// We want to check only once per subscription.
		if inArray(doneSubscriptions, subscriptionID) {
			continue
		}

		nets, err := c.getVirtualNetworksFromSubscription(ctx, key.ClusterID(&cluster))
		if err != nil {
			// We can't use this Azure credentials. Might be wrong in the Secret file.
			// We shouldn't block the network calculation for this reason.
			c.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("Error getting used virtual networks for subscription %s: %s", subscriptionID, err))
			continue
		}

		doneSubscriptions = append(doneSubscriptions, subscriptionID)
		ret = append(ret, nets...)
	}

	return ret, nil
}

func (c *VirtualNetworkCollector) getVirtualNetworksFromSubscription(ctx context.Context, clusterID string) ([]net.IPNet, error) {
	groupsClient, err := c.wcAzureClientFactory.GetGroupsClient(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	vnetClient, err := c.wcAzureClientFactory.GetVirtualNetworksClient(ctx, clusterID)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Look for all resource groups that have a tag named 'GiantSwarmInstallation' with installation name as value.
	iterator, err := groupsClient.ListComplete(ctx, fmt.Sprintf("tagName eq 'GiantSwarmInstallation' and tagValue eq '%s'", c.installationName), nil)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var ret []net.IPNet

	for iterator.NotDone() {
		group := iterator.Value()

		// Search a VNET with any of the expected names.
		// Note: One we move to fully utilize CAPI/CAPZ types only (without AzureConfig), we should
		// not expect that tenant cluster virtual networks follow any specific naming convention.
		// We could list VNets by tags (we currently don't set tags to VNets).
		vnetCandidates := []string{
			fmt.Sprintf("%s-VirtualNetwork", *group.Name),
			c.installationName,
		}

		for _, vnetName := range vnetCandidates {
			vnet, err := vnetClient.Get(ctx, *group.Name, vnetName, "")
			if IsNotFound(err) {
				// VNET with desired name not found, ignore this resource group.
			} else if err != nil {
				return nil, microerror.Mask(err)
			} else {
				for _, cidr := range *vnet.AddressSpace.AddressPrefixes {
					_, n, err := net.ParseCIDR(cidr)
					if err != nil {
						return nil, microerror.Mask(err)
					}

					ret = append(ret, *n)
				}
			}
		}

		err = iterator.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return ret, nil
}

func inArray(a []string, s string) bool {
	for _, x := range a {
		if x == s {
			return true
		}
	}

	return false
}

func (c *VirtualNetworkCollector) getAllTenantClusters(ctx context.Context) (*v1alpha1.AzureConfigList, error) {
	tenantClusterList := &v1alpha1.AzureConfigList{}
	err := c.k8sclient.CtrlClient().List(ctx, tenantClusterList, client2.InNamespace(metav1.NamespaceAll))

	return tenantClusterList, err
}
