package azureclusterconditions

import (
	"context"
	"reflect"

	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/giantswarm/azure-operator/v7/service/controller/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, cr interface{}) error {
	var err error
	azureCluster, err := key.ToAzureCluster(cr)
	if err != nil {
		return microerror.Mask(err)
	}

	oldStatus := azureCluster.Status

	// ensure Ready condition
	err = r.ensureReadyCondition(ctx, &azureCluster)
	if err != nil {
		return microerror.Mask(err)
	}

	if !reflect.DeepEqual(oldStatus, azureCluster.Status) {
		r.logger.Debugf(ctx, "status is changed, updating AzureCluster CR")
		err = r.ctrlClient.Status().Update(ctx, &azureCluster)
		if apierrors.IsConflict(err) {
			r.logger.Debugf(ctx, "conflict trying to save object in k8s API concurrently")
			r.logger.Debugf(ctx, "canceling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
