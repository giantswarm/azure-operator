// +build k8srequired

package teardown

import (
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/integration/env"
)

// Teardown e2e testing environment.
func Teardown(c *client.AzureClientSet, g *framework.Guest, h *framework.Host) error {
	var err error

	{
		// TODO the deletion detection should rather happen based on the cluster status or even based on the CR being gone
		h.DeleteGuestCluster("azure-operator", "azureconfig", "removed finalizer 'operatorkit.giantswarm.io/azure-operator'")

		// only do full teardown when not on CI
		if env.CircleCI() == "true" {
			return nil
		}
	}

	{
		err = framework.HelmCmd("delete azure-operator --purge")
		if err != nil {
			return microerror.Mask(err)
		}
		err = framework.HelmCmd("delete cert-operator --purge")
		if err != nil {
			return microerror.Mask(err)
		}
		err = framework.HelmCmd("delete node-operator --purge")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = framework.HelmCmd("delete cert-config-e2e --purge")
		if err != nil {
			return microerror.Mask(err)
		}
		err = framework.HelmCmd("delete apiextensions-azure-config-e2e --purge")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
