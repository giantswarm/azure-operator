package dnsrecord

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/arm/dns"
	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/key"
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	o, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Maskf(err, "GetCurrentState")
	}

	return r.getCurrentState(ctx, o)
}

func (r *Resource) getCurrentState(ctx context.Context, obj azuretpr.CustomObject) (dnsRecords, error) {
	recordSetsClient, err := r.getDNSRecordSetsClient()
	if err != nil {
		return nil, microerror.Maskf(err, "GetCurrentState")
	}

	current := newPartialDNSRecords(obj)

	for i, record := range current {
		resp, err := recordSetsClient.Get(record.ZoneRG, record.Zone, record.RelativeName, dns.NS)
		if client.ResponseWasNotFound(resp.Response) {
			continue
		} else if err != nil {
			return nil, microerror.Maskf(err, "GetCurrentState: getting record=%#v", record)
		}

		var nameServers []string
		for _, ns := range *resp.NsRecords {
			nameServers = append(nameServers, *ns.Nsdname)
		}

		current[i].NameServers = nameServers
	}

	return current, nil
}
