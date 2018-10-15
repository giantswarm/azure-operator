// +build k8srequired

package setup

import (
	"context"
	"log"
	"os"
	"testing"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/e2etemplates/pkg/e2etemplates"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/integration/env"
)

const (
	azureResourceValuesFile = "/tmp/azure-operator-values.yaml"
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
					CredentialDefault: chartvalues.AzureOperatorConfigSecretAzureOperatorCredentialDefault{
						ClientID:       env.AzureClientID(),
						ClientSecret:   env.AzureClientSecret(),
						SubscriptionID: env.AzureSubscriptionID(),
						TenantID:       env.AzureTenantID(),
					},
					SecretYaml: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYaml{
						Service: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYamlService{
							Azure: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYamlServiceAzure{
								ClientID:       env.AzureClientID(),
								ClientSecret:   env.AzureClientSecret(),
								SubscriptionID: env.AzureSubscriptionID(),
								TenantID:       env.AzureTenantID(),
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
	}

	{
		c := chartvalues.APIExtensionsAzureConfigE2EConfig{
			Azure: chartvalues.APIExtensionsAzureConfigE2EConfigAzure{
				CalicoSubnetCIDR: env.AzureCalicoSubnetCIDR(),
				CIDR:             env.AzureCIDR(),
				Location:         env.AzureLocation(),
				MasterSubnetCIDR: env.AzureMasterSubnetCIDR(),
				VPNSubnetCIDR:    env.AzureVPNSubnetCIDR(),
				WorkerSubnetCIDR: env.AzureWorkerSubnetCIDR(),
			},
			ClusterName:               env.ClusterID(),
			CommonDomain:              env.CommonDomain(),
			CommonDomainResourceGroup: env.CommonDomainResourceGroup(),
			VersionBundleVersion:      env.VersionBundleVersion(),
		}

		values, err := chartvalues.NewAPIExtensionsAzureConfigE2E(c)
		if err != nil {
			return microerror.Mask(err)
		}

		err = config.Release.Install(context.Background(), "apiextensions-azure-config-e2e", release.NewStableVersion(), values)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
