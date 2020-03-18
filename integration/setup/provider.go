package setup

import (
	"context"

	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/integration/env"
)

// provider installs the operator and tenant cluster CR.
func provider(ctx context.Context, config Config) error {
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

	return nil
}
