package dnsrecord

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/v3/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

// ApplyCreateChange is never called. We do not like it. It is not idempotent.
func (r *Resource) ApplyCreateChange(ctx context.Context, obj, change interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	c, err := toDNSRecords(change)
	if err != nil {
		return microerror.Mask(err)
	}

	return r.applyCreateChange(ctx, cr, c)
}

func (r *Resource) applyCreateChange(ctx context.Context, obj providerv1alpha1.AzureConfig, change dnsRecords) error {
	return nil
}
