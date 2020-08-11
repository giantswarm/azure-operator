package dnsrecord

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/operatorkit/v2/pkg/resource/crud"
)

func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, change interface{}) error {
	c, err := toDNSRecords(change)
	if err != nil {
		return microerror.Mask(err)
	}

	return r.applyUpdateChange(ctx, c)
}

func (r *Resource) applyUpdateChange(ctx context.Context, change dnsRecords) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring host cluster DNS records")

	if len(change) == 0 {
		r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring host cluster DNS records: already ensured")
		return nil
	}

	for _, record := range change {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring host cluster DNS record=%#v", record))

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

		_, err := r.cpRecordSetsClient.CreateOrUpdate(ctx, record.ZoneRG, record.Zone, record.RelativeName, dns.NS, params, "", "")
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("ensuring host cluster DNS record=%#v: ensured", record))
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring host cluster DNS records: ensured")
	return nil
}

// NewUpdatePatch returns the patch creating resource group for this cluster if
// it is needed.
func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*crud.Patch, error) {
	c, err := toDNSRecords(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	d, err := toDNSRecords(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r.newUpdatePatch(c, d)
}

func (r *Resource) newUpdatePatch(currentState, desiredState dnsRecords) (*crud.Patch, error) {
	patch := crud.NewPatch()

	updateChange := r.newUpdateChange(currentState, desiredState)

	patch.SetUpdateChange(updateChange)
	return patch, nil
}

func (r *Resource) newUpdateChange(currentState, desiredState dnsRecords) dnsRecords {
	var change dnsRecords
	for _, d := range desiredState {
		if !currentState.Contains(d) {
			change = append(change, d)
		}
	}

	return change
}
