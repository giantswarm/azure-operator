package setup

import (
	"context"

	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/integration/env"
)

// Teardown e2e testing environment.
func Teardown(c Config) error {
	ctx := context.Background()

	_, err := c.AzureClient.ResourceGroupsClient.Delete(ctx, env.ClusterID())
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
