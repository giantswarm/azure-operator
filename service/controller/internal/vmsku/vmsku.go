package vmsku

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/v5/client"
)

const (
	// CapabilitySupported is the value returned by this API from Azure when the capability is supported
	CapabilitySupported = "True"

	CapabilityAcceleratedNetworking = "AcceleratedNetworkingEnabled"
)

type Config struct {
	MCAzureClientFactory client.CredentialsAwareClientFactoryInterface
	Location             string
	Logger               micrologger.Logger
}

type VMSKUs struct {
	mcAzureClientFactory client.CredentialsAwareClientFactoryInterface
	location             string
	skus                 map[string]*compute.ResourceSku
	logger               micrologger.Logger
	initMutex            sync.Mutex
}

func New(config Config) (*VMSKUs, error) {
	if config.MCAzureClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.WCAzureClientFactory must not be empty", config)
	}
	if config.Location == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.Location must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	return &VMSKUs{
		mcAzureClientFactory: config.MCAzureClientFactory,
		location:             config.Location,
		logger:               config.Logger,
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
	v.initMutex.Lock()
	defer v.initMutex.Unlock()
	if len(v.skus) == 0 {
		err := v.initCache(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (v *VMSKUs) initCache(ctx context.Context) error {
	v.logger.Debugf(ctx, "Initializing cache for VMSKU")
	cl, err := v.mcAzureClientFactory.GetResourceSkusClient(ctx, "")
	if err != nil {
		return microerror.Mask(err)
	}

	filter := fmt.Sprintf("location eq '%s'", v.location)
	v.logger.Debugf(ctx, "Filter is: '%s'", filter)
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

	v.logger.Debugf(ctx, "Number of SKUs in cache: '%d'", len(skus))

	return nil
}
