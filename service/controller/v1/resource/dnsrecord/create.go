package dnsrecord

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v1/key"
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

func (r *Resource) applyCreateChange(ctx context.Context, obj providerv1alpha1.AzureConfig, change dnsRecords) error {
	return nil
}
