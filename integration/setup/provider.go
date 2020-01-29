package setup

import (
	"context"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/integration/env"
)

// provider installs the operator and tenant cluster CR.
func provider(ctx context.Context, config Config) error {
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
							Tenant: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYamlServiceTenant{
								Ignition: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYamlServiceTenantIgnition{
									Debug: chartvalues.AzureOperatorConfigSecretAzureOperatorSecretYamlServiceTenantIgnitionDebug{
										Enabled:    env.IgnitionDebugEnabled(),
										LogsPrefix: env.IgnitionDebugLogsPrefix(),
										LogsToken:  env.IgnitionDebugLogsToken(),
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

		err = config.Release.InstallOperator(ctx, "azure-operator", release.NewVersion(env.CircleSHA()), values, providerv1alpha1.NewAzureConfigCRD())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		c := chartvalues.CredentialdConfig{
			Azure: chartvalues.CredentialdConfigAzure{
				CredentialDefault: chartvalues.CredentialdConfigAzureCredentialDefault{
					ClientID:       env.AzureClientID(),
					ClientSecret:   env.AzureClientSecret(),
					SubscriptionID: env.AzureSubscriptionID(),
					TenantID:       env.AzureTenantID(),
				},
			},
			Deployment: chartvalues.CredentialdConfigDeployment{
				Replicas: 0,
			},
			RegistryPullSecret: env.RegistryPullSecret(),
		}

		values, err := chartvalues.NewCredentiald(c)
		if err != nil {
			return microerror.Mask(err)
		}

		err = config.Release.Install(ctx, "credentiald", release.NewStableVersion(), values, config.Release.Condition().SecretExists(ctx, namespace, "credential-default"))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		c := chartvalues.APIExtensionsAzureConfigE2EConfig{
			Azure: chartvalues.APIExtensionsAzureConfigE2EConfigAzure{
				AvailabilityZones: env.AzureAvailabilityZones(),
				CalicoSubnetCIDR:  env.AzureCalicoSubnetCIDR(),
				CIDR:              env.AzureCIDR(),
				Location:          env.AzureLocation(),
				MasterSubnetCIDR:  env.AzureMasterSubnetCIDR(),
				VPNSubnetCIDR:     env.AzureVPNSubnetCIDR(),
				WorkerSubnetCIDR:  env.AzureWorkerSubnetCIDR(),
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

		err = config.Release.Install(ctx, "apiextensions-azure-config-e2e", release.NewStableVersion(), values)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
