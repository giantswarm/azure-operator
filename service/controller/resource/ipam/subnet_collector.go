package ipam

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"sync"

	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/ipam"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"golang.org/x/sync/errgroup"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

type SubnetCollectorConfig struct {
	G8sClient versioned.Interface
	Logger    micrologger.Logger

	NetworkRange net.IPNet
}

type SubnetCollector struct {
	g8sClient versioned.Interface
	logger    micrologger.Logger

	networkRange net.IPNet
}

func NewSubnetCollector(config SubnetCollectorConfig) (*SubnetCollector, error) {
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if reflect.DeepEqual(config.NetworkRange, net.IPNet{}) {
		return nil, microerror.Maskf(invalidConfigError, "%T.NetworkRange must not be empty", config)
	}

	c := &SubnetCollector{
		g8sClient: config.G8sClient,
		logger:    config.Logger,

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

		subnets, err := c.getSubnetsFromSubscription(ctx)
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
	azureConfigList, err := c.g8sClient.ProviderV1alpha1().AzureConfigs(metav1.NamespaceAll).List(metav1.ListOptions{})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var results []net.IPNet
	for _, ac := range azureConfigList.Items {
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

func (c *SubnetCollector) getSubnetsFromSubscription(ctx context.Context) ([]net.IPNet, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	groupsClient := cc.AzureClientSet.GroupsClient
	vnetClient := cc.AzureClientSet.VirtualNetworkClient

	iterator, err := groupsClient.ListComplete(ctx, "tagName eq 'GiantSwarmCluster'", nil)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var ret []net.IPNet

	for iterator.NotDone() {
		group := iterator.Value()

		fmt.Printf("Group %s is interesting\n", *group.Name)

		// Search a VNET with the expected name.
		vnetName := fmt.Sprintf("%s-VirtualNetwork", *group.Name)

		vnet, err := vnetClient.Get(ctx, *group.Name, vnetName, "")
		if key.IsNotFound(err) {
			// VNET with desired name not found, ignore this resource group.
			continue
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		for _, cidr := range *vnet.AddressSpace.AddressPrefixes {
			_, n, err := net.ParseCIDR(cidr)
			if err != nil {
				return nil, microerror.Mask(err)
			}

			ret = append(ret, *n)
		}

		err = iterator.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return ret, nil
}
