package masters

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	capzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	m, err := meta.Accessor(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	// Ensure that AzureMachines for the cluster are deleted.
	{
		o := client.MatchingLabels{
			capiv1alpha3.ClusterLabelName: key.ClusterID(m),
		}
		mList := new(capzv1alpha3.AzureMachineList)
		err = r.ctrlClient.List(ctx, mList, o)
		if err != nil {
			return microerror.Mask(err)
		}

		for _, m := range mList.Items {
			err = r.ctrlClient.Delete(ctx, &m)
			if errors.IsNotFound(err) {
				continue
			} else if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	return nil
}
