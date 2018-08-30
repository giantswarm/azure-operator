package dnsrecord

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/controller"

	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

// ApplyDeleteChange deletes the resource group via the Azure API.
func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, change interface{}) error {
	obj, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	dnsRecords, err := toDNSRecords(change)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(dnsRecords) != 0 {
		recordSetsClient, err := r.getDNSRecordSetsHostClient()
		if err != nil {
			return microerror.Mask(err)
		}

		for _, record := range dnsRecords {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleting host cluster DNS record '%s'", record.RelativeName))

			_, err := recordSetsClient.Delete(ctx, record.ZoneRG, record.Zone, record.RelativeName, dns.NS, "")
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deleted host cluster DNS record '%s'", record.RelativeName))
		}
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "not deleting host cluster DNS records")
	}

	return nil
}

// NewDeletePatch returns the patch deleting resource group for this cluster if
// it is needed.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	o, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	c, err := toDNSRecords(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	d, err := toDNSRecords(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r.newDeletePatch(ctx, o, c, d)
}

func (r *Resource) newDeletePatch(ctx context.Context, obj providerv1alpha1.AzureConfig, currentState, desiredState dnsRecords) (*controller.Patch, error) {
	patch := controller.NewPatch()

	deleteChange, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch.SetDeleteChange(deleteChange)

	return patch, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj providerv1alpha1.AzureConfig, currentState, desiredState dnsRecords) (dnsRecords, error) {
	return currentState, nil
}
