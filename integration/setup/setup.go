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

	if os.Getenv("KEEP_RESOURCES") != "true" {
		teardown.Teardown(c, g, h)
	}

	os.Exit(r)
}

// Setup e2e testing environment.
func Setup(c *client.AzureClientSet, g *framework.Guest, h *framework.Host) error {
	var v int
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
		err = h.InstallStableOperator("cert-operator", "certconfig", template.CertOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}

		err = h.InstallCertResource()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = h.InstallBranchOperator("azure-operator", "azureconfig", template.AzureOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}

		err = installAzureResource()
		if err != nil {
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
