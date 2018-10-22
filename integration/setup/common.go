package setup

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/microerror"
)

// common installs components required to run the operator.
func common(config Config) error {
	ctx := context.Background()

	{
		err := config.K8s.EnsureNamespaceCreated(ctx, namespace)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		c := chartvalues.E2ESetupVaultConfig{
			Vault: chartvalues.E2ESetupVaultConfigVault{
				Token: env.VaultToken(),
			},
		}

		values, err := chartvalues.NewE2ESetupVault(c)
		if err != nil {
			return microerror.Mask(err)
		}

		err = config.Release.Install(ctx, "e2esetup-vault", release.NewStableVersion(), values, config.Release.Condition().PodExists(ctx, "default", "app=vault"))
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
		err := config.Host.InstallCertResource()
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		c := chartvalues.NodeOperatorConfig{
			RegistryPullSecret: env.RegistryPullSecret(),
		}

		values, err := chartvalues.NewNodeOperator(c)
		if err != nil {
			return microerror.Mask(err)
		}

		err = config.Release.InstallOperator(ctx, "node-operator", release.NewStableVersion(), values, corev1alpha1.NewNodeConfigCRD())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
