package vmsku

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

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
	Logger        micrologger.Logger
}

type VMSKUs struct {
	clientFactory *client.Factory
	location      string
	skus          map[string]*compute.ResourceSku
	logger        micrologger.Logger
	mux           sync.Mutex
}

func New(config Config) (*VMSKUs, error) {
	if config.ClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClientFactory must not be empty", config)
	}
	if config.Location == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Location must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	return &VMSKUs{
		clientFactory: config.ClientFactory,
		location:      config.Location,
		logger:        config.Logger,
	}, nil
}

func (v *VMSKUs) HasCapability(ctx context.Context, vmType string, name string) (bool, error) {
	err := v.ensureInitialized(ctx)
	if err != nil {
		return false, microerror.Mask(err)
	}
	vmsku, found := v.skus[vmType]
	if !found {
		return false, microerror.Maskf(skuNotFoundError, vmType)
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

func (v *VMSKUs) ensureInitialized(ctx context.Context) error {
	v.mux.Lock()
	defer v.mux.Unlock()
	if len(v.skus) == 0 {
		err := v.initCache(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (v *VMSKUs) getResourcesSkusClient() (*compute.ResourceSkusClient, error) {
	cl, err := v.clientFactory.GetResourceSkusClient("giantswarm", "credential-default")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cl, nil
}

func (v *VMSKUs) initCache(ctx context.Context) error {
	v.logger.LogCtx(ctx, "level", "debug", "message", "Initializing cache for VMSKU")
	cl, err := v.getResourcesSkusClient()
	if err != nil {
		return microerror.Mask(err)
	}

	filter := fmt.Sprintf("location eq '%s'", v.location)
	v.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Filter is: '%s'", filter))
	iterator, err := cl.ListComplete(ctx, filter)
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

	v.skus = skus

	v.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Number of SKUs in cache: '%d'", len(skus)))

	return nil
}
