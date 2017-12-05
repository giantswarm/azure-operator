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
