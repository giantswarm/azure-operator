package dnsrecord

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/arm/dns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/azure-operator/service/key"
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/framework"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, change interface{}) error {
	o, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Maskf(err, "ensuring host cluster DNS records")
	}

	c, err := toDNSRecords(change)
	if err != nil {
		return microerror.Maskf(err, "ensuring host cluster DNS records")
	}

	return r.applyUpdateChange(ctx, o, c)
}

func (r *Resource) applyUpdateChange(ctx context.Context, obj azuretpr.CustomObject, change dnsRecords) error {
	r.logger.LogCtx(ctx, "debug", "ensuring host cluster DNS records")

	recordSetsClient, err := r.getDNSRecordSetsClient()
	if err != nil {
		return microerror.Maskf(err, "ensuring host cluster DNS records")
	}

	if len(change) == 0 {
		r.logger.LogCtx(ctx, "debug", "ensuring host cluster DNS records: already ensured")
		return nil
	}

	for _, record := range change {
		r.logger.LogCtx(ctx, "debug", fmt.Sprintf("ensuring host cluster DNS record=%#v", record))

		var params dns.RecordSet
		{
			var nameServers []dns.NsRecord
			for _, ns := range record.NameServers {
				nameServers = append(nameServers, dns.NsRecord{Nsdname: to.StringPtr(ns)})
			}
			params.RecordSetProperties = &dns.RecordSetProperties{
				TTL:       to.Int64Ptr(300),
				NsRecords: &nameServers,
			}
		}

		_, err := recordSetsClient.CreateOrUpdate(record.ZoneRG, record.Zone, record.RelativeName, dns.NS, params, "", "")
		if err != nil {
			return microerror.Maskf(err, fmt.Sprintf("ensuring host cluster DNS record=%#v", record))
		}

		r.logger.LogCtx(ctx, "debug", fmt.Sprintf("ensuring host cluster DNS record=%#v: ensured", record))
	}

	r.logger.LogCtx(ctx, "debug", "ensuring host cluster DNS records: ensured")
	return nil
}

// NewUpdatePatch returns the patch creating resource group for this cluster if
// it is needed.
func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	o, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}
	c, err := toDNSRecords(currentState)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}
	d, err := toDNSRecords(desiredState)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}

	return r.newUpdatePatch(ctx, o, c, d)
}

func (r *Resource) newUpdatePatch(ctx context.Context, obj azuretpr.CustomObject, currentState, desiredState dnsRecords) (*framework.Patch, error) {
	patch := framework.NewPatch()

	updateChange, err := r.newUpdateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Maskf(err, "NewUpdatePatch")
	}

	patch.SetUpdateChange(updateChange)
	return patch, nil
}

func (r *Resource) newUpdateChange(ctx context.Context, obj azuretpr.CustomObject, currentState, desiredState dnsRecords) (dnsRecords, error) {
	var change dnsRecords
	for _, d := range desiredState {
		if !currentState.Contains(d) {
			change = append(change, d)
		}
	}

	return change, nil
}
