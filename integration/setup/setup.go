// +build k8srequired

package setup

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/integration/template"
)

const (
	azureResourceValuesFile = "/tmp/azure-operator-values.yaml"
)

func WrapTestMain(c *client.AzureClientSet, g *framework.Guest, h *framework.Host, m *testing.M) {
	if err := h.Setup(); err != nil {
		log.Fatalf("%#v\n", err)
	}

	if err := Resources(c, g, h); err != nil {
		log.Fatalf("%#v\n", err)
	}

	if err := g.Setup(); err != nil {
		log.Fatalf("%#v\n", err)
	}

	os.Exit(m.Run())
}

func Resources(c *client.AzureClientSet, g *framework.Guest, h *framework.Host) error {
	{
		if err := h.InstallCertOperator(); err != nil {
			return microerror.Mask(err)
		}

		if err := h.InstallCertResource(); err != nil {
			return microerror.Mask(err)
		}
	}

	{
		if err := h.InstallBranchOperator("azure-operator", "azureconfig", template.AzureOperatorChartValues); err != nil {
			return microerror.Mask(err)
		}

		if err := installAzureResource(); err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func installAzureResource() error {
	azureResourceChartValuesEnv := os.ExpandEnv(template.AzureResourceChartValues)
	d := []byte(azureResourceChartValuesEnv)

	err := ioutil.WriteFile(azureResourceValuesFile, d, 0644)
	if err != nil {
		return microerror.Mask(err)
	}

	err = framework.HelmCmd("registry install quay.io/giantswarm/azure-resource-chart -- -n azure-resource-lab --values " + azureResourceValuesFile)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
