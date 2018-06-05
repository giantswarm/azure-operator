package instance

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("EnsureCreated called for cluster ID '%s'", key.ClusterID(customObject)))

	// TODO list all instances
	// TODO find the first instance not having the latest scale set model applied
	// TODO trigger update for found instance

	return nil
}
