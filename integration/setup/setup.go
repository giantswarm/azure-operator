// +build k8srequired

package setup

import (
	"log"
	"os"
	"testing"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/e2etemplates/pkg/e2etemplates"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/teardown"
	"github.com/giantswarm/azure-operator/integration/template"
)

const (
	azureResourceValuesFile = "/tmp/azure-operator-values.yaml"
)

// WrapTestMain setup and teardown e2e testing environment.
func WrapTestMain(c *client.AzureClientSet, g *framework.Guest, h *framework.Host, m *testing.M) {
	var r int

	err := Setup(c, g, h)
	if err != nil {
		log.Printf("%#v\n", err)
		r = 1
	} else {
		r = m.Run()
	}

	if env.KeepResources() != "true" {
		teardown.Teardown(c, g, h)
	}

	os.Exit(r)
}

// Setup e2e testing environment.
func Setup(c *client.AzureClientSet, g *framework.Guest, h *framework.Host) error {
	var err error

	err = h.Setup()
	if err != nil {
		return microerror.Mask(err)
	}

	err = Resources(c, g, h)
	if err != nil {
		return microerror.Mask(err)
	}

	err = g.Setup()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Resources install required charts.
func Resources(c *client.AzureClientSet, g *framework.Guest, h *framework.Host) error {
	var err error

	{
		err = h.InstallStableOperator("cert-operator", "certconfig", e2etemplates.CertOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}

		err = h.InstallBranchOperator("azure-operator", "azureconfig", template.AzureOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = h.InstallResource("apiextensions-cert-config-e2e", template.CertConfigE2EChartValues, ":stable")
		if err != nil {
			return microerror.Mask(err)
		}

		err = h.InstallResource("apiextensions-azure-config-e2e", template.AzureConfigE2EChartValues, ":stable")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
