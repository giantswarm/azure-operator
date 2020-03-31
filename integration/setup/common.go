package setup

import (
	"context"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/microerror"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/key"
)

// common installs components required to run the operator.
func common(ctx context.Context, config Config) error {
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

		err = config.Release.Install(ctx, key.VaultReleaseName(), release.NewStableVersion(), values, config.Release.Condition().PodExists(ctx, "default", "app=vault"))
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		c := chartvalues.CertOperatorConfig{
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

		err = config.Release.InstallOperator(ctx, key.CertOperatorReleaseName(), release.NewStableVersion(), values, corev1alpha1.NewCertConfigCRD())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err := ensureCertConfigsInstalled(ctx, env.ClusterID(), config)
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

		err = config.Release.InstallOperator(ctx, key.NodeOperatorReleaseName(), release.NewStableVersion(), values, corev1alpha1.NewDrainerConfigCRD())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err := ensureCRDCreated(ctx, config, *releasev1alpha1.NewReleaseCRD())
		if err != nil {
			return microerror.Mask(err)
		}

		err = ensureCRDCreated(ctx, config, *releasev1alpha1.NewReleaseCycleCRD())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func ensureCRDCreated(ctx context.Context, config Config, crd apiextensionsv1beta1.CustomResourceDefinition) error {
	o := func() error {
		_, err := config.K8sClients.ExtClient().ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return microerror.Mask(err)
		} else if err != nil {
			return backoff.Permanent(microerror.Mask(err))
		}
		return nil
	}
	b := backoff.NewExponential(backoff.ShortMaxWait, backoff.ShortMaxInterval)
	n := backoff.NewNotifier(config.Logger, ctx)
	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
