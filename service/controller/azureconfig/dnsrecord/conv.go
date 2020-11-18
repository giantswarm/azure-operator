package dnsrecord

import (
	"github.com/giantswarm/microerror"
)

func toDNSRecords(v interface{}) (dnsRecords, error) {
	if v == nil {
		return dnsRecords{}, nil
	}

	r, ok := v.(dnsRecords)
	if !ok {
		return dnsRecords{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", r, v)
	}

	return r, nil
}
