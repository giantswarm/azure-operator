package setup

import (
	"context"

	"github.com/giantswarm/microerror"
)

// Teardown e2e testing environment.
func Teardown(c Config) error {
	ctx := context.Background()

	// TODO the deletion detection should rather happen based on the cluster
	// status or even based on the CR being gone.
	//
	//     https://github.com/giantswarm/giantswarm/issues/3839
	//
	err := c.Host.DeleteGuestCluster(ctx, "azure")
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
