package endpoints

import (
	"context"

	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/core/v1"
	apismetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v7/pkg/project"
	"github.com/giantswarm/azure-operator/v7/service/controller/key"
)

func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	masterNICPrivateIPs, err := r.getMasterNICPrivateIPs(ctx, key.ClusterID(&cr), key.MasterVMSSName(cr))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	endpoints := &v1.Endpoints{
		ObjectMeta: apismetav1.ObjectMeta{
			Name:      "master",
			Namespace: key.ClusterID(&cr),
			Labels: map[string]string{
				key.LabelApp:           "master",
				key.LabelCluster:       key.ClusterID(&cr),
				key.LabelCustomer:      key.ClusterCustomer(cr),
				key.LegacyLabelCluster: key.ClusterID(&cr),
				key.LabelManagedBy:     project.Name(),
				key.LabelOrganization:  key.ClusterCustomer(cr),
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
