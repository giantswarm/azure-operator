package setup

import (
	"context"
	"time"

	corev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v3/integration/env"
	"github.com/giantswarm/azure-operator/v3/integration/key"
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
		_, err := config.K8sClients.K8sClient().CoreV1().Secrets("giantswarm").Create(credentialDefault())
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err := installCertOperator(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring drainerconfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, corev1alpha1.NewDrainerConfigCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured drainerconfig CRD exists")
	}

	{
		err := installNodeOperator(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring Release CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, releasev1alpha1.NewReleaseCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured Release CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring ReleaseCycle CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, releasev1alpha1.NewReleaseCycleCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured ReleaseCycle CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring Release exists", "release", env.VersionBundleVersion())
		_, err := config.K8sClients.G8sClient().ReleaseV1alpha1().Releases().Create(&releasev1alpha1.Release{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "v1.0.0",
				Namespace: "default",
				Labels: map[string]string{
					"giantswarm.io/managed-by": "release-operator",
					"giantswarm.io/provider":   "azure",
				},
			},
			Spec: releasev1alpha1.ReleaseSpec{
				Apps: []releasev1alpha1.ReleaseSpecApp{},
				Components: []releasev1alpha1.ReleaseSpecComponent{
					{
						Name:    "azure-operator",
						Version: env.VersionBundleVersion(),
					},
					{
						Name:    "calico",
						Version: "3.10.1",
					},
					{
						Name:    "containerlinux",
						Version: "2345.3.0",
					},
					{
						Name:    "coredns",
						Version: "1.6.5",
					},
					{
						Name:    "etcd",
						Version: "3.3.17",
					},
					{
						Name:    "kubernetes",
						Version: "1.16.8",
					},
				},
				Date:  &metav1.Time{Time: time.Unix(10, 0)},
				State: "active",
			},
		})
		if err != nil {
			return microerror.Mask(err)
		}
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured Release exists", "release", env.VersionBundleVersion())
	}

	return nil
}

func credentialDefault() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "credential-default",
			Namespace: "giantswarm",
			Labels: map[string]string{
				"app":                        "credentiald",
				"giantswarm.io/managed-by":   "credentiald",
				"giantswarm.io/organization": "giantswarm",
				"giantswarm.io/service-type": "system",
			},
		},
		Data: map[string][]byte{
			"azure.azureoperator.clientid":       []byte(env.AzureClientID()),
			"azure.azureoperator.clientsecret":   []byte(env.AzureClientSecret()),
			"azure.azureoperator.subscriptionid": []byte(env.AzureSubscriptionID()),
			"azure.azureoperator.tenantid":       []byte(env.AzureTenantID()),
		},
		Type: "Opaque",
	}
}
