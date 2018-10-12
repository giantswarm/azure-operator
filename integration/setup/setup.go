// +build k8srequired

package setup

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/pkg/framework"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/e2etemplates/pkg/e2etemplates"
	"github.com/giantswarm/microerror"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/template"
	"github.com/giantswarm/azure-operator/service/controller/v3/credential"
)

const (
	azureResourceValuesFile = "/tmp/azure-operator-values.yaml"

	credentialName      = "credential-default"
	credentialNamespace = "giantswarm"
)

// WrapTestMain setup and teardown e2e testing environment.
func WrapTestMain(m *testing.M, c Config) {
	var r int

	err := Setup(c)
	if err != nil {
		log.Printf("%#v\n", err)
		r = 1
	} else {
		r = m.Run()
	}

	if env.KeepResources() != "true" {
		Teardown(c)
	}

	os.Exit(r)
}

// Setup e2e testing environment.
func Setup(c Config) error {
	var err error

	err = c.Host.Setup()
	if err != nil {
		return microerror.Mask(err)
	}

	err = Resources(c)
	if err != nil {
		return microerror.Mask(err)
	}

	err = c.Guest.Setup()
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// Resources install required charts.
func Resources(config Config) error {
	var err error

	{
		c := chartvalues.AzureOperatorConfig{
			Provider: chartvalues.AzureOperatorConfigProvider{
				Azure: chartvalues.AzureOperatorConfigProviderAzure{
					Location: env.AzureLocation(),
				},
			},
			Secret: chartvalues.AzureOperatorConfigSecret{
				AzureOperator: chartvalues.AzureOperatorConfigSecretAzureOperator{
					SecretYaml: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYaml{
						Service: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYamlService{
							Azure: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYamlServiceAzure{
								ClientID:      env.AzureClientID(),
								ClientSecret:  env.AzureClientSecret(),
								SubsciptionID: env.AzureSubscriptionID(),
								TenantID:      env.AzureTenantID(),
								Template: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYamlServiceAzureTemplate{
									URI: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYamlServiceAzureTemplateURI{
										Version: env.CircleSHA(),
									},
								},
							},
						},
					},
				},
				Registry: chartvalues.AzureOperatorConfigSecretRegistry{
					PullSecret: chartvalues.AzureOperatorConfigSecretRegistryPullSecret{
						DockerConfigJSON: env.RegistryPullSecret(),
					},
				},
			},
		}

		values, err := chartvalues.NewAzureOperator(c)
		if err != nil {
			return microerror.Mask(err)
		}

		err = config.Release.InstallOperator(context.Background(), "azure-operator", release.NewVersion(env.CircleSHA()), values, providerv1alpha1.NewAzureConfigCRD())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		c := chartvalues.CertOperatorConfig{
			ClusterName:        env.ClusterID(),
			CommonDomain:       env.CommonDomain(),
			RegistryPullSecret: env.RegistryPullSecret(),
			Vault: chartvalues.CertOperatorVault{
				Token: env.VaultToken(),
			},
		}

		values, err := chartvalues.NewCertOperator(c)
		if err != nil {
			return microerror.Mask(err)
		}

		err = config.Host.InstallStableOperator("cert-operator", "certconfig", values)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = config.Host.InstallStableOperator("node-operator", "drainerconfig", e2etemplates.NodeOperatorChartValues)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = config.Host.InstallCertResource()
		if err != nil {
			return microerror.Mask(err)
		}

		err = installCredential(config.Host)
		if err != nil {
			return microerror.Mask(err)
		}

		err = config.Host.InstallResource("apiextensions-azure-config-e2e", template.AzureConfigE2EChartValues, ":stable")
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
