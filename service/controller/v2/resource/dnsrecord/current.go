package dnsrecord

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2017-10-01/dns"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	o, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r.getCurrentState(ctx, o)
}

func (r *Resource) getCurrentState(ctx context.Context, obj providerv1alpha1.AzureConfig) (dnsRecords, error) {
	recordSetsClient, err := r.getDNSRecordSetsClient()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	current := newPartialDNSRecords(obj)

	for i, record := range current {
		resp, err := recordSetsClient.Get(ctx, record.ZoneRG, record.Zone, record.RelativeName, dns.NS)
		if client.ResponseWasNotFound(resp.Response) {
			continue
		} else if err != nil {
			return nil, microerror.Mask(err)
		}

		var nameServers []string
		for _, ns := range *resp.NsRecords {
			nameServers = append(nameServers, *ns.Nsdname)
		}

		current[i].NameServers = nameServers
	}

	return current, nil
}
