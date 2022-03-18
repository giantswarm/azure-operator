package dnsrecord

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/dns/mgmt/2018-05-01/dns"
	providerv1alpha1 "github.com/giantswarm/apiextensions/v5/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/v5/client"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return r.getCurrentState(ctx, cr)
}

func (r *Resource) getCurrentState(ctx context.Context, obj providerv1alpha1.AzureConfig) (dnsRecords, error) {
	recordSetsClient, err := r.getDNSRecordSetsGuestClient(ctx)
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
