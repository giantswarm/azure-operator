package vmsku

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/giantswarm/microerror"
)

const (
	// CapabilitySupported is the value returned by this API from Azure when the capability is supported
	CapabilitySupported = "True"

	CapabilityAcceleratedNetworking = "AcceleratedNetworkingEnabled"
)

type VMSKU struct {
	SKU *compute.ResourceSku
}

func New(ctx context.Context, client *compute.ResourceSkusClient, vmType string) (*VMSKU, error) {
	iterator, err := client.ListComplete(ctx, "")
	if err != nil {
		return nil, microerror.Mask(err)
	}

	for iterator.NotDone() {
		sku := iterator.Value()

		if *sku.Name == vmType {
			return &VMSKU{
				SKU: &sku,
			}, nil
		}

		err := iterator.NextWithContext(ctx)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	return nil, microerror.Mask(skuNotFoundError)
}

func (v *VMSKU) HasCapability(name string) bool {
	if v.SKU.Capabilities != nil {
		for _, capability := range *v.SKU.Capabilities {
			if capability.Name != nil && *capability.Name == name {
				if capability.Value != nil && strings.EqualFold(*capability.Value, string(CapabilitySupported)) {
					return true
				}
			}
		}
	}
	return false
}
