// +build k8srequired

package teardown

import (
	"os"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
)

// Teardown e2e testing environment.
func Teardown(c *client.AzureClientSet, g *framework.Guest, h *framework.Host) error {
	var err error

	{
		h.DeleteGuestCluster("azure-operator", "azureconfig", "deleting host vnet peering: deleted")

		// only do full teardown when not on CI
		if os.Getenv("CIRCLECI") == "true" {
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
	}

	{
		err = framework.HelmCmd("delete cert-resource-lab --purge")
		if err != nil {
			return microerror.Mask(err)
		}
		err = framework.HelmCmd("delete azure-resource-lab --purge")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
