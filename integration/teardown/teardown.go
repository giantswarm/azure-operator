// +build k8srequired

package teardown

import (
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
)

func Teardown(c *client.AzureClientSet, g *framework.Guest, h *framework.Host) error {
	var err error

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
