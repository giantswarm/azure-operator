// +build k8srequired

package teardown

import (
	"context"
	"fmt"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/integration/env"
)

const (
	provider = "azure"
)

// Teardown e2e testing environment.
func Teardown(g *framework.Guest, h *framework.Host) error {
	ctx := context.Background()

	var err error

	{
		// TODO the deletion detection should rather happen based on the cluster
		// status or even based on the CR being gone.
		//
		//     https://github.com/giantswarm/giantswarm/issues/3839
		//
		h.DeleteGuestCluster(ctx, provider)

		// only do full teardown when not on CI
		if env.CircleCI() == "true" {
			return nil
		}
	}

	{
		err = framework.HelmCmd(fmt.Sprintf("delete %s-azure-operator --purge", h.TargetNamespace()))
		if err != nil {
			return microerror.Mask(err)
		}
		err = framework.HelmCmd(fmt.Sprintf("delete %s-cert-operator --purge", h.TargetNamespace()))
		if err != nil {
			return microerror.Mask(err)
		}
		err = framework.HelmCmd(fmt.Sprintf("delete %s-node-operator --purge", h.TargetNamespace()))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = framework.HelmCmd(fmt.Sprintf("delete %s-cert-config-e2e --purge", h.TargetNamespace()))
		if err != nil {
			return microerror.Mask(err)
		}
		err = framework.HelmCmd(fmt.Sprintf("delete %s-apiextensions-azure-config-e2e --purge", h.TargetNamespace()))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
