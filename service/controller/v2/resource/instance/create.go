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
	{
		c, err := r.getVMsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		g := key.ResourceGroupName(customObject)
		s := key.WorkerVMSSName(customObject)
		result, err := c.List(ctx, g, s, "", "", "")
		if err != nil {
			return microerror.Mask(err)
		}

		fmt.Printf("\n")
		fmt.Printf("\n")
		fmt.Printf("\n")
		fmt.Printf("NotDone: %#v\n", result.NotDone())
		fmt.Printf("\n")

		for _, v := range result.Values() {
			fmt.Printf("InstanceID: %#v\n", *v.InstanceID)
			fmt.Printf("LatestModelApplied: %#v\n", *v.LatestModelApplied)
			fmt.Printf("\n")
		}

		fmt.Printf("NotDone: %#v\n", result.NotDone())
		fmt.Printf("\n")
		fmt.Printf("\n")
		fmt.Printf("\n")
	}
	// TODO find the first instance not having the latest scale set model applied
	// TODO trigger update for found instance

	return nil
}
