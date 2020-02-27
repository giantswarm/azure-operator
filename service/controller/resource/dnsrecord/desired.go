package dnsrecord

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/key"

	"github.com/giantswarm/azure-operator/client"
)

// GetDesiredState returns the desired resource group for this cluster.
func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r.getDesiredState(ctx, cr)
}

func (r *Resource) getDesiredState(ctx context.Context, obj providerv1alpha1.AzureConfig) (dnsRecords, error) {
	zonesClient, err := r.getDNSZonesGuestClient(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	desired := newPartialDNSRecords(obj)

	for i, record := range desired {
		zone := record.RelativeName + "." + record.Zone
		resp, err := zonesClient.Get(ctx, key.ResourceGroupName(obj), zone)
		if client.ResponseWasNotFound(resp.Response) {
			return dnsRecords{}, nil
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		var nameServers []string
		for _, ns := range *resp.NameServers {
			nameServers = append(nameServers, ns)
		}

		desired[i].NameServers = nameServers
	}

	return desired, nil
}
