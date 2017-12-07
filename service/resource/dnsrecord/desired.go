package dnsrecord

import (
	"context"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/key"
	"github.com/giantswarm/azuretpr"
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

func (r *Resource) getDesiredState(ctx context.Context, obj azuretpr.CustomObject) (dnsRecords, error) {
	zonesClient, err := r.getDNSZonesClient()
	if err != nil {
		return nil, microerror.Maskf(err, "GetDesiredState")
	}

	desired := newPartialDNSRecords(obj)

	for i, record := range desired {
		zone := record.RelativeName + "." + record.Zone
		resp, err := zonesClient.Get(key.ResourceGroupName(obj), zone)
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
