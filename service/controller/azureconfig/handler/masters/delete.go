package masters

import (
	"context"

	"github.com/giantswarm/microerror"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
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
			capi.ClusterLabelName: key.ClusterID(m),
		}
		mList := new(capz.AzureMachineList)
		err = r.ctrlClient.List(ctx, mList, o)
		if err != nil {
			return microerror.Mask(err)
		}

		for i := range mList.Items {
			err = r.ctrlClient.Delete(ctx, &mList.Items[i])
			if errors.IsNotFound(err) {
				continue
			} else if err != nil {
				return microerror.Mask(err)
			}
		}
	}

	return nil
}
