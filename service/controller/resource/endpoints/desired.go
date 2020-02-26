package endpoints

import (
	"context"

	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	masterNICPrivateIPs, err := r.getMasterNICPrivateIPs(ctx, key.ClusterID(cr), key.MasterVMSSName(cr))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	endpoints := &v1.Endpoints{
		ObjectMeta: apismetav1.ObjectMeta{
			Name:      "master",
			Namespace: key.ClusterID(cr),
			Labels: map[string]string{
				"app":                        "master",
				"cluster":                    key.ClusterID(cr),
				"customer":                   key.ClusterCustomer(cr),
				"giantswarm.io/cluster":      key.ClusterID(cr),
				"giantswarm.io/organization": key.ClusterCustomer(cr),
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
