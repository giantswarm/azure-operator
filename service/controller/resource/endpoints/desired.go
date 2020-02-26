package endpoints

import (
	"context"

	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	masterNICPrivateIPs, err := r.getMasterNICPrivateIPs(ctx, key.ClusterID(customObject), key.MasterVMSSName(customObject))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	endpoints := &v1.Endpoints{
		ObjectMeta: apismetav1.ObjectMeta{
			Name:      "master",
			Namespace: key.ClusterID(customObject),
			Labels: map[string]string{
				key.LabelApp:           "master",
				key.LabelCluster:       key.ClusterID(customObject),
				key.LabelCustomer:      key.ClusterCustomer(customObject),
				key.LegacyLabelCluster: key.ClusterID(customObject),
				key.LabelManagedBy:     "azure-operator",
				key.LabelOrganization:  key.ClusterCustomer(customObject),
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
