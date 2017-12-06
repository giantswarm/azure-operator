package dnsrecord

import (
	"context"

	"github.com/giantswarm/azure-operator/service/key"
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"
)

// ApplyCreateChange is never called. We do not like it. It is not idempotent.
func (r *Resource) ApplyCreateChange(ctx context.Context, obj, change interface{}) error {
	o, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Maskf(err, "creating DNS records in host cluster")
	}

	c, err := toDNSRecords(change)
	if err != nil {
		return microerror.Maskf(err, "creating DNS records in host cluster")
	}

	return r.applyCreateChange(ctx, o, c)
}

func (r *Resource) applyCreateChange(ctx context.Context, obj azuretpr.CustomObject, change dnsRecords) error {
	return nil
}
