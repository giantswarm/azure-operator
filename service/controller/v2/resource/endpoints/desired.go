package endpoints

import (
	"context"

	"k8s.io/api/core/v2"
	apismetav2 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	masterNICPrivateIPs, err := r.getMasterNICPrivateIPs(key.ClusterID(customObject), key.MasterVMSSName(customObject))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	endpoints := &v2.Endpoints{
		ObjectMeta: apismetav2.ObjectMeta{
			Name:      "master",
			Namespace: key.ClusterID(customObject),
			Labels: map[string]string{
				"app":                        "master",
				"cluster":                    key.ClusterID(customObject),
				"customer":                   key.ClusterCustomer(customObject),
				"giantswarm.io/cluster":      key.ClusterID(customObject),
				"giantswarm.io/organization": key.ClusterCustomer(customObject),
				"giantswarm.io/managed-by":   "azure-operator",
			},
		},
	}

	for _, ip := range masterNICPrivateIPs {
		endpoints.Subsets = append(endpoints.Subsets, v2.EndpointSubset{
			Addresses: []v2.EndpointAddress{
				{
					IP: ip,
				},
			},
			Ports: []v2.EndpointPort{
				{
					Port: httpsPort,
				},
			},
		})
	}

	return endpoints, nil
}
