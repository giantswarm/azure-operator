package cleanup

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-11-01/network"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v3/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v3/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "cleanup"
)

type Config struct {
	Logger micrologger.Logger
}

// Resource manages Azure resource groups.
type Resource struct {
	logger micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		logger: config.Logger,
	}

	return r, nil
}

// EnsureCreated ensures that old, unused resources from old releases get deleted.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Ensure the fronted IP configuration 'dummy-frontend' is deleted.
	loadBalancersClient, err := r.getLoadBalancersClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Getting the ingress controller load balancer")
	lb, err := loadBalancersClient.Get(ctx, key.ResourceGroupName(cr), key.IngressControllerLoadBalancerName, "")
	if err != nil {
		return microerror.Mask(err)
	}

	var cnfs []network.FrontendIPConfiguration
	for _, fc := range *lb.FrontendIPConfigurations {
		if *fc.Name != key.DummyFrontendConfigurationName {
			cnfs = append(cnfs, fc)
		}
	}

	if len(cnfs) != len(*lb.FrontendIPConfigurations) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "Ingress load balancer needs updating")

		lb.FrontendIPConfigurations = &cnfs

		r.logger.LogCtx(ctx, "level", "debug", "message", "Updating Ingress load balancer to remove the dummy frontend IP configuration")

		_, err := loadBalancersClient.CreateOrUpdate(ctx, key.ResourceGroupName(cr), *lb.Name, lb)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "Updated Ingress load balancer to remove the dummy frontend IP configuration")

		// Don't proceed with deleting the public IP address before the FrontendIP configuration is deleted.
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Dummy frontend configuration already delete from the ingress load balancer")

	// Ensure the Public IP 'dummy-pip' is deleted.
	publicIpClient, err := r.getPublicIPAddressesClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Checking if the dummy public IP address exists")
	_, err = publicIpClient.Delete(ctx, key.ResourceGroupName(cr), key.DummyPublicIpName)
	if IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "The dummy public IP address already deleted")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "Requested deletion of the dummy public IP address")

	return nil
}

// EnsureDeleted ensures the resource group is deleted via the Azure API.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getPublicIPAddressesClient(ctx context.Context) (*network.PublicIPAddressesClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.PublicIPAddressesClient, nil
}

func (r *Resource) getLoadBalancersClient(ctx context.Context) (*network.LoadBalancersClient, error) {
	return nil, nil
}
