package dnsrecord

import (
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"
)

func toCustomObject(v interface{}) (azuretpr.CustomObject, error) {
	if v == nil {
		return azuretpr.CustomObject{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &azuretpr.CustomObject{}, v)
	}

	customObjectPointer, ok := v.(*azuretpr.CustomObject)
	if !ok {
		return azuretpr.CustomObject{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &azuretpr.CustomObject{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func toDNSRecords(v interface{}) (DNSRecords, error) {
	if v == nil {
		return DNSRecords{}, nil
	}

	r, ok := v.(DNSRecords)
	if !ok {
		return DNSRecords{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", r, v)
	}

	return r, nil
}
