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
	"github.com/giantswarm/azure-operator/integration/teardown"
	"github.com/giantswarm/azure-operator/integration/template"
)

const (
	azureResourceValuesFile = "/tmp/azure-operator-values.yaml"
)

func WrapTestMain(c *client.AzureClientSet, g *framework.Guest, h *framework.Host, m *testing.M) {
	var v int
	var err error

	err = h.Setup()
	if err != nil {
		log.Printf("%#v\n", err)
		v = 1
	}

	err = Resources(c, g, h)
	if err != nil {
		log.Printf("%#v\n", err)
		v = 1
	}

	err = g.Setup()
	if err != nil {
		log.Printf("%#v\n", err)
		v = 1
	}

	if v == 0 {
		v = m.Run()
	}

	if os.Getenv("KEEP_RESOURCES") != "true" {
		h.DeleteGuestCluster("azureconfig", "azure-operator", "deleting host vnet peering: deleted")

		// only do full teardown when not on CI
		if os.Getenv("CIRCLECI") != "true" {
			err := teardown.Teardown(c, g, h)
			if err != nil {
				log.Printf("%#v\n", err)
				v = 1
			}
			h.Teardown()
		}
	}

	os.Exit(v)
}

func Resources(c *client.AzureClientSet, g *framework.Guest, h *framework.Host) error {
	{
		if err := h.InstallStableOperator("cert-operator", "certconfig", template.CertOperatorChartValues); err != nil {
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
