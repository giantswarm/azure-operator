package dnsrecord

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v4/pkg/resource/crud"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// ApplyDeleteChange deletes the resource group via the Azure API.
func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, change interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	dnsRecords, err := toDNSRecords(change)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(dnsRecords) != 0 {
		for _, record := range dnsRecords {
			r.logger.Debugf(ctx, "deleting host cluster DNS record '%s'", record.RelativeName)

			cpRecordSetsClient, err := r.mcAzureClientFactory.GetDnsRecordSetsClient(ctx, key.ClusterID(&cr))
			if err != nil {
				return microerror.Mask(err)
			}

			_, err = cpRecordSetsClient.Delete(ctx, record.ZoneRG, record.Zone, record.RelativeName, dns.NS, "")
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.Debugf(ctx, "deleted host cluster DNS record '%s'", record.RelativeName)
		}
	} else {
		r.logger.Debugf(ctx, "not deleting host cluster DNS records")
	}

	return nil
}

// NewDeletePatch returns the patch deleting resource group for this cluster if
// it is needed.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	cr, err := key.ToCustomResource(obj)
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

	return r.newDeletePatch(ctx, cr, c, d)
}

func (r *Resource) newDeletePatch(ctx context.Context, obj providerv1alpha1.AzureConfig, currentState, desiredState dnsRecords) (*crud.Patch, error) {
	patch := crud.NewPatch()

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
