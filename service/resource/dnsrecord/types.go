package dnsrecord

import (
	"reflect"

	"github.com/giantswarm/azure-operator/service/key"
	"github.com/giantswarm/azuretpr"
)

type nsRecord struct {
	// RelativeName is the newly created DNS zone name relative to the
	// parent zone.
	RelativeName string
	// Zone is a parent DNS zone name in which NS record is created.
	Zone string
	// ZoneRG is the parent DNS zone resource group name.
	ZoneRG string
	// NameServers are entires for the NS record to be created in the
	// parent DNS zone.
	NameServers []string
}

type dnsRecords []nsRecord

func (rs dnsRecords) Contains(r nsRecord) bool {
	for _, e := range rs {
		if reflect.DeepEqual(r, e) {
			return true
		}
	}
	return false
}

// newPartialDNSRecords creates DNSRecords without NameServers filled.
func newPartialDNSRecords(obj azuretpr.CustomObject) dnsRecords {
	all := dnsRecords{
		// api.
		{
			RelativeName: key.DNSZonePrefixAPI(obj),
			Zone:         key.DNSZoneAPI(obj),
			ZoneRG:       key.DNSZoneResourceGroupAPI(obj),
		},
		// etcd.
		{
			RelativeName: key.DNSZonePrefixEtcd(obj),
			Zone:         key.DNSZoneEtcd(obj),
			ZoneRG:       key.DNSZoneResourceGroupEtcd(obj),
		},
		// ingress.
		{
			RelativeName: key.DNSZonePrefixIngress(obj),
			Zone:         key.DNSZoneIngress(obj),
			ZoneRG:       key.DNSZoneResourceGroupIngress(obj),
		},
	}

	var unique dnsRecords
	for _, r := range all {
		if !unique.Contains(r) {
			unique = append(unique, r)
		}
	}

	return unique
}
