// +build k8srequired

package setup

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/e2etemplates/pkg/e2etemplates"
	"github.com/giantswarm/microerror"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/teardown"
	"github.com/giantswarm/azure-operator/integration/template"
	"github.com/giantswarm/azure-operator/service/controller/v3/credential"
)

const (
	azureResourceValuesFile = "/tmp/azure-operator-values.yaml"

	credentialName      = "credential-default"
	credentialNamespace = "giantswarm"
)

// WrapTestMain setup and teardown e2e testing environment.
func WrapTestMain(g *framework.Guest, h *framework.Host, m *testing.M) {
	var r int

	err := Setup(g, h)
	if err != nil {
		log.Printf("%#v\n", err)
		r = 1
	} else {
		r = m.Run()
	}

	if env.KeepResources() != "true" {
		teardown.Teardown(g, h)
	}

	os.Exit(r)
}

// Setup e2e testing environment.
func Setup(g *framework.Guest, h *framework.Host) error {
	var err error

	err = h.Setup()
	if err != nil {
		return microerror.Mask(err)
	}

	err = Resources(g, h)
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
func Resources(g *framework.Guest, h *framework.Host) error {
	var err error

	{
		err = h.InstallBranchOperator("azure-operator", "azureconfig", template.AzureOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}
		err = h.InstallStableOperator("cert-operator", "certconfig", e2etemplates.CertOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}
		err = h.InstallStableOperator("node-operator", "drainerconfig", e2etemplates.NodeOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = h.InstallCertResource()
		if err != nil {
			return microerror.Mask(err)
		}

		err = installCredential(h)
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

func installCredential(h *framework.Host) error {
	o := func() error {
		k8sClient := h.K8sClient()

		k8sClient.CoreV1().Secrets(credentialNamespace).Delete(credentialName, &metav1.DeleteOptions{})

		secret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: credentialName,
			},

			Data: map[string][]byte{
				credential.ClientIDKey:       []byte(env.AzureGuestClientID()),
				credential.ClientSecretKey:   []byte(env.AzureGuestClientSecret()),
				credential.SubscriptionIDKey: []byte(env.AzureGuestSubscriptionID()),
				credential.TenantIDKey:       []byte(env.AzureGuestTenantID()),
			},
		}

		_, err := k8sClient.CoreV1().Secrets(credentialNamespace).Create(secret)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}
	b := backoff.NewExponential(backoff.ShortMaxWait, backoff.ShortMaxInterval)
	n := func(err error, delay time.Duration) {
		log.Println("level", "debug", "message", err.Error())
	}

	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
