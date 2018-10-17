// +build k8srequired

package setup

import (
	"context"
	"fmt"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/integration/env"
)

// Teardown e2e testing environment.
func Teardown(c Config) error {
	ctx := context.Background()

	var err error

	{
		// TODO the deletion detection should rather happen based on the cluster
		// status or even based on the CR being gone.
		//
		//     https://github.com/giantswarm/giantswarm/issues/3839
		//
		c.Host.DeleteGuestCluster(ctx, "azure")

		// only do full teardown when not on CI
		if env.CircleCI() == "true" {
			return nil
		}
	}

	{
		err = framework.HelmCmd(fmt.Sprintf("delete %s-azure-operator --purge", c.Host.TargetNamespace()))
		if err != nil {
			return microerror.Mask(err)
		}
		err = framework.HelmCmd(fmt.Sprintf("delete %s-cert-operator --purge", c.Host.TargetNamespace()))
		if err != nil {
			return microerror.Mask(err)
		}
		err = framework.HelmCmd(fmt.Sprintf("delete %s-node-operator --purge", c.Host.TargetNamespace()))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = framework.HelmCmd(fmt.Sprintf("delete %s-cert-config-e2e --purge", c.Host.TargetNamespace()))
		if err != nil {
			return microerror.Mask(err)
		}
		err = framework.HelmCmd(fmt.Sprintf("delete %s-apiextensions-azure-config-e2e --purge", c.Host.TargetNamespace()))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
