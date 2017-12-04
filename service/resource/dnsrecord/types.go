package dnsrecord

import (
	"reflect"

	"github.com/giantswarm/azure-operator/service/key"
	"github.com/giantswarm/azuretpr"
)

// TODO unexport
type NSRecord struct {
	RelativeName string
	Zone         string
	NameServers  []string
}

// TODO unexport
type DNSRecords []NSRecord

func (rs DNSRecords) Contains(r NSRecord) bool {
	for _, e := range rs {
		if reflect.DeepEqual(r, e) {
			return true
		}
	}
	return false
}

// newPartialDNSRecords creates DNSRecords without NameServers filled.
func newPartialDNSRecords(obj azuretpr.CustomObject) DNSRecords {
	all := DNSRecords{
		// api.
		{
			RelativeName: key.RelativeDNSAPIRecord(obj),
			Zone:         obj.Spec.Azure.DNSZones.API,
		},
		// etcd.
		{
			RelativeName: key.RelativeDNSEtcdRecord(obj),
			Zone:         obj.Spec.Azure.DNSZones.Etcd,
		},
		// ingress.
		{
			RelativeName: key.RelativeDNSIngressRecord(obj),
			Zone:         obj.Spec.Azure.DNSZones.Ingress,
		},
	}

	var unique DNSRecords
	for _, r := range all {
		if !unique.Contains(r) {
			unique = append(unique, r)
		}
	}

	return unique
}
