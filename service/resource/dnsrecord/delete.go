package dnsrecord

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/dns"
	"github.com/giantswarm/azure-operator/service/key"
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/framework"
)

// ApplyDeleteChange deletes the resource group via the Azure API.
func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, change interface{}) error {
	o, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Maskf(err, "deleting host cluster DNS records")
	}
	c, err := toDNSRecords(change)
	if err != nil {
		return microerror.Maskf(err, "deleting host cluster DNS records")
	}

	return r.applyDeleteChange(ctx, o, c)
}

func (r *Resource) applyDeleteChange(ctx context.Context, obj azuretpr.CustomObject, change dnsRecords) error {
	r.logger.LogCtx(ctx, "debug", "deleting host cluster DNS records")

	if len(change) == 0 {
		r.logger.LogCtx(ctx, "debug", "deleting host cluster DNS records: already deleted")
		return nil
	}

	recordSetsClient, err := r.getDNSRecordSetsClient()
	if err != nil {
		return microerror.Maskf(err, "deleting host cluster DNS records")
	}

	for _, record := range change {
		r.logger.LogCtx(ctx, "debug", fmt.Sprintf("deleting host cluster DNS record=%#v", record))

		_, err := recordSetsClient.Delete(record.ZoneRG, record.Zone, record.RelativeName, dns.NS, "")
		if err != nil {
			return microerror.Maskf(err, fmt.Sprintf("deleting host cluster DNS record=%#v", record))
		}

		r.logger.LogCtx(ctx, "debug", fmt.Sprintf("deleting host cluster DNS record=%#v: deleted", record))
	}

	r.logger.LogCtx(ctx, "debug", "deleting host cluster DNS records: deleted")
	return nil
}

// NewDeletePatch returns the patch deleting resource group for this cluster if
// it is needed.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	o, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}
	c, err := toDNSRecords(currentState)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}
	d, err := toDNSRecords(desiredState)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}

	return r.newDeletePatch(ctx, o, c, d)
}

func (r *Resource) newDeletePatch(ctx context.Context, obj azuretpr.CustomObject, currentState, desiredState dnsRecords) (*framework.Patch, error) {
	patch := framework.NewPatch()

	deleteChange, err := r.newDeleteChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Maskf(err, "NewDeletePatch")
	}

	patch.SetDeleteChange(deleteChange)
	return patch, nil
}

func (r *Resource) newDeleteChange(ctx context.Context, obj azuretpr.CustomObject, currentState, desiredState dnsRecords) (dnsRecords, error) {
	return currentState, nil
}
