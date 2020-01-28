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
					Location:        env.AzureLocation(),
					HostClusterCidr: "0.0.0.0/0",
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
			SSHUser:                   "test-user",
			SSHPublicKey:              "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQDBSSJCLkZWhOvs6blotU+fWbrTmC7fOwOm0+w01Ww/YN3j3j1vCrvji1A4Yonr89ePQEQKfZsYcYFodQI/D3Uzu9rOFy0dCMQfvL/J6N8LkNtmooh3J2p061829MurAdD+TVsNGrD2FZGm5Ab4NiyDXIGAYCaHL6BHP16ipBglYjLQt6jVyzdTbYspkRi1QrsNFN3gIv9V47qQSvoNEsC97gvumKzCSQ/EwJzFoIlqVkZZHZTXvGwnZrAVXB69t9Y8OJ5zA6cYFAKR0O7lEiMpebdLNGkZgMA6t2PADxfT78PHkYXLR/4tchVuOSopssJqgSs7JgIktEE14xKyNyoLKIyBBo3xwywnDySsL8R2zG4Ytw1luo79pnSpIzTvfwrNhd7Cg//OYzyDCty+XUEUQx2JfOBx5Qb1OFw71WA+zYqjbworOsy2ZZ9UAy8ryjiaeT8L2ZRGuhdicD6kkL3Lxg5UeNIxS2FLNwgepZ4D8Vo6Yxe+VOZl524ffoOJSHQ0Gz8uE76hXMNEcn4t8HVkbR4sCMgLn2YbwJ2dJcROj4w80O4qgtN1vsL16r4gt9o6euml8LbmnJz6MtGdMczSO7kHRxirtEHMTtYbT1wNgUAzimbScRggBpUz5gbz+NRE1Xgnf4A5yNMRy+JOWtLVUozJlcGSiQkVcexzdb27yQ==",
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
