package vmsku

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v4/client"
)

const (
	// CapabilitySupported is the value returned by this API from Azure when the capability is supported
	CapabilitySupported = "True"

	CapabilityAcceleratedNetworking = "AcceleratedNetworkingEnabled"
)

type Config struct {
	ClientFactory *client.Factory
	Location      string
}

type Interface struct {
	clientFactory *client.Factory
	location      string
	skus          map[string]*compute.ResourceSku
}

func New(config Config) (*Interface, error) {
	if config.ClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClientFactory must not be empty", config)
	}
	if config.Location == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Location must not be empty", config)
	}
	return &Interface{
		clientFactory: config.ClientFactory,
		location:      config.Location,
	}, nil
}

func (v *Interface) HasCapability(ctx context.Context, vmType string, name string) (bool, error) {
	if len(v.skus) == 0 {
		err := v.initCache(ctx)
		if err != nil {
			return false, microerror.Mask(err)
		}
	}
	vmsku, found := v.skus[vmType]
	if !found {
		return false, microerror.Mask(skuNotFoundError)
	}
	if vmsku.Capabilities != nil {
		for _, capability := range *vmsku.Capabilities {
			if capability.Name != nil && *capability.Name == name {
				if capability.Value != nil && strings.EqualFold(*capability.Value, string(CapabilitySupported)) {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (v *Interface) initCache(ctx context.Context) error {
	cl, err := v.clientFactory.GetResourceSkusClient("giantswarm", "credential-default")
	if err != nil {
		return microerror.Mask(err)
	}

	iterator, err := cl.ListComplete(ctx, fmt.Sprintf("location eq '%s'", v.location))
	if err != nil {
		return microerror.Mask(err)
	}

	skus := map[string]*compute.ResourceSku{}

	for iterator.NotDone() {
		sku := iterator.Value()

		skus[*sku.Name] = &sku

		err := iterator.NextWithContext(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
