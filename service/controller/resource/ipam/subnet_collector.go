package ipam

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/ipam"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/client"
	"github.com/giantswarm/azure-operator/v4/pkg/credential"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

type SubnetCollectorConfig struct {
	GSClientCredentialsConfig auth.ClientCredentialsConfig
	K8sClient                 k8sclient.Interface
	InstallationName          string
	Logger                    micrologger.Logger

	NetworkRange net.IPNet
}

type SubnetCollector struct {
	gsClientCredentialsConfig auth.ClientCredentialsConfig
	k8sclient                 k8sclient.Interface
	installationName          string
	logger                    micrologger.Logger

	networkRange net.IPNet
}

func NewSubnetCollector(config SubnetCollectorConfig) (*SubnetCollector, error) {
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

	c := &SubnetCollector{
		k8sclient:        config.K8sClient,
		installationName: config.InstallationName,
		logger:           config.Logger,

		networkRange: config.NetworkRange,
	}

	return c, nil
}

func (c *SubnetCollector) Collect(ctx context.Context) ([]net.IPNet, error) {
	var err error
	var mutex sync.Mutex
	var reservedSubnets []net.IPNet

	g := &errgroup.Group{}

	g.Go(func() error {
		c.logger.LogCtx(ctx, "level", "debug", "message", "finding allocated subnets from AzureConfig CRs")

		subnets, err := c.getSubnetsFromAzureConfigs(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
		mutex.Lock()
		reservedSubnets = append(reservedSubnets, subnets...)
		mutex.Unlock()

		c.logger.LogCtx(ctx, "level", "debug", "message", "found allocated subnets from AzureConfig CRs")

		return nil
	})

	g.Go(func() error {
		c.logger.LogCtx(ctx, "level", "debug", "message", "finding allocated subnets from all resource groups in the subscription")

		subnets, err := c.getSubnetsFromAllSubscriptions(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
		mutex.Lock()
		reservedSubnets = append(reservedSubnets, subnets...)
		mutex.Unlock()

		c.logger.LogCtx(ctx, "level", "debug", "message", "found allocated subnets from all resource groups in the subscription")

		return nil
	})

	err = g.Wait()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	reservedSubnets = ipam.CanonicalizeSubnets(c.networkRange, reservedSubnets)

	return reservedSubnets, nil
}

func (c *SubnetCollector) getSubnetsFromAzureConfigs(ctx context.Context) ([]net.IPNet, error) {
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

func (c *SubnetCollector) getSubnetsFromAllSubscriptions(ctx context.Context) ([]net.IPNet, error) {
	tenantClusterList, err := c.getAllTenantClusters(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var doneSubscriptions []string
	var ret []net.IPNet
	for _, cluster := range tenantClusterList.Items {
		organizationAzureClientCredentialsConfig, subscriptionID, partnerID, err := credential.GetOrganizationAzureCredentials(c.k8sclient, cluster, c.gsClientCredentialsConfig.TenantID)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		organizationAzureClientSet, err := client.NewAzureClientSet(organizationAzureClientCredentialsConfig, subscriptionID, partnerID)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		// We want to check only once per subscription.
		if inArray(doneSubscriptions, subscriptionID) {
			continue
		}

		nets, err := c.getSubnetsFromSubscription(ctx, organizationAzureClientSet)
		if err != nil {
			// We can't use this Azure credentials. Might be wrong in the Secret file.
			// We shouldn't block the network calculation for this reason.
			c.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("Error getting used subnets for subscription %s: %s", subscriptionID, err))
			continue
		}

		doneSubscriptions = append(doneSubscriptions, subscriptionID)
		ret = append(ret, nets...)
	}

	return ret, nil
}

func (c *SubnetCollector) getSubnetsFromSubscription(ctx context.Context, clientSet *client.AzureClientSet) ([]net.IPNet, error) {
	groupsClient := clientSet.GroupsClient
	vnetClient := clientSet.VirtualNetworkClient

	// Look for all resource groups that have a tag named 'GiantSwarmInstallation' with installation name as value.
	iterator, err := groupsClient.ListComplete(ctx, fmt.Sprintf("tagName eq 'GiantSwarmInstallation' and tagValue eq '%s'", c.installationName), nil)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var ret []net.IPNet

	for iterator.NotDone() {
		group := iterator.Value()

		// Search a VNET with any of the expected names.
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

func (c *SubnetCollector) getAllTenantClusters(ctx context.Context) (*v1alpha1.AzureConfigList, error) {
	tenantClusterList := &v1alpha1.AzureConfigList{}
	err := c.k8sclient.CtrlClient().List(ctx, tenantClusterList, client2.InNamespace(metav1.NamespaceAll))

	return tenantClusterList, err
}
