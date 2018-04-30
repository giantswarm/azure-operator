package dnsrecord

import (
	"context"

	providerv2alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
	"github.com/giantswarm/microerror"
)

// GetDesiredState returns the desired resource group for this cluster.
func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	o, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Maskf(err, "GetDesiredState")
	}

	return r.getDesiredState(ctx, o)
}

func (r *Resource) getDesiredState(ctx context.Context, obj providerv2alpha1.AzureConfig) (dnsRecords, error) {
	zonesClient, err := r.getDNSZonesClient()
	if err != nil {
		return nil, microerror.Maskf(err, "GetDesiredState")
	}

	desired := newPartialDNSRecords(obj)

	for i, record := range desired {
		zone := record.RelativeName + "." + record.Zone
		resp, err := zonesClient.Get(ctx, key.ResourceGroupName(obj), zone)
		if client.ResponseWasNotFound(resp.Response) {
			return dnsRecords{}, nil
		} else if err != nil {
			return nil, microerror.Maskf(err, "GetDesiredState: getting zone=%q", zone)
		}

		var nameServers []string
		for _, ns := range *resp.NameServers {
			nameServers = append(nameServers, ns)
		}

		desired[i].NameServers = nameServers
	}

	return desired, nil
}
