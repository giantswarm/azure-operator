package endpoints

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/v3/key"
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

	endpoints := &v1.Endpoints{
		ObjectMeta: apismetav1.ObjectMeta{
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
		endpoints.Subsets = append(endpoints.Subsets, v1.EndpointSubset{
			Addresses: []v1.EndpointAddress{
				{
					IP: ip,
				},
			},
			Ports: []v1.EndpointPort{
				{
					Port: httpsPort,
				},
			},
		})
	}

	return endpoints, nil
}
