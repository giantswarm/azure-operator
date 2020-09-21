package setup

import (
	"context"
	"fmt"
	"time"

	corev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/core/v1alpha1"
	v12 "k8s.io/api/rbac/v1"
	v1 "k8s.io/api/scheduling/v1"

	"github.com/giantswarm/apiextensions/v2/pkg/apis/application/v1alpha1"
	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/apiextensions/v2/pkg/crd"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/e2e-harness/v2/pkg/release"
	"github.com/giantswarm/e2etemplates/pkg/chartvalues"
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/v4/e2e/env"
	"github.com/giantswarm/azure-operator/v4/e2e/key"
)

const (
	ClusterIPRange = "172.31.0.0/16"
)

// common installs components required to run the operator.
func common(ctx context.Context, config Config, giantSwarmRelease releasev1alpha1.Release) error {
	{
		err := config.K8s.EnsureNamespaceCreated(ctx, namespace)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Ensure CRDs.
	{
		err := ensureCRDs(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		// Steup RBAC.
		{
			clusterRoleBinding := v12.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: "jwt-reviewer",
				},
				Subjects: []v12.Subject{
					{
						Kind:      "ServiceAccount",
						Name:      "default",
						Namespace: "default",
					},
				},
				RoleRef: v12.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "cluster-admin",
				},
			}

			_, err := config.K8sClients.K8sClient().RbacV1().ClusterRoleBindings().Create(ctx, &clusterRoleBinding, metav1.CreateOptions{})
			if err != nil {
				return microerror.Mask(err)
			}
		}

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

	// Ensure draughtsman config.
	{
		err := ensureDraughtsman(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Deploy App Operator v2 for control plane.
	{
		err := installAppOperator(ctx, config, "2.2.0")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Deploy Chart Operator.
	{
		err := installChartOperator(ctx, config, "2.3.1")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Ensure app catalogs.
	{
		err := ensureAppCatalogs(ctx, config, "2.2.0")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Install node operator.
	{
		err := installNodeOperator(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Install release operator.
	{
		err := installReleaseOperator(ctx, config, "2.1.0")
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Install cluster service.
	{
		err := installClusterService(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func ensureCRDs(ctx context.Context, config Config) error {
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring appcatalog CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("application.giantswarm.io", "AppCatalog"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured appcatalog CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring App CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, v1alpha1.NewAppCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured App CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring Chart CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, v1alpha1.NewChartCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured Chart CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring Spark CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, corev1alpha1.NewSparkCRD(), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured Spark CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring CertConfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("core.giantswarm.io", "CertConfig"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured CertConfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring drainerconfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("core.giantswarm.io", "DrainerConfig"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured drainerconfig CRD exists")
	}

	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensuring storageconfig CRD exists")

		err := config.K8sClients.CRDClient().EnsureCreated(ctx, crd.LoadV1("core.giantswarm.io", "StorageConfig"), backoff.NewMaxRetries(7, 1*time.Second))
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", "ensured storageconfig CRD exists")
	}

	return nil
}

func installNodeOperator(ctx context.Context, config Config) error {
	{
		nodeOperatorValues := `Installation:
  V1:
    Registry:
      Domain: quay.io
`
		err := installLatestReleaseChartPackage(ctx, config, "node-operator", nodeOperatorValues, CatalogStorageURL)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func installClusterService(ctx context.Context, config Config) error {
	// Steup RBAC.
	{
		clusterRoleBinding := v12.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "jwt-reviewer-cluster-service",
			},
			Subjects: []v12.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "cluster-service",
					Namespace: "giantswarm",
				},
			},
			RoleRef: v12.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "ClusterRole",
				Name:     "cluster-admin",
			},
		}

		_, err := config.K8sClients.K8sClient().RbacV1().ClusterRoleBindings().Create(ctx, &clusterRoleBinding, metav1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		clusterServiceValues := `Registry:
      Domain: quay.io
`
		err := installLatestReleaseChartPackage(ctx, config, "cluster-service", getDraughtsmanMergedConfig(clusterServiceValues), CatalogStorageURL)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func installAppOperator(ctx context.Context, config Config, version string) error {
	{
		chartName := "app-operator"
		tarballURL := fmt.Sprintf("https://giantswarm.github.com/control-plane-catalog/%s-%s.tgz", chartName, version)
		chartPackagePath, err := config.HelmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for %#q release is %#q", chartName, chartPackagePath))
		err = installChart(ctx, config, "app-operator-unique", "", chartPackagePath)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	return nil
}

func installChartOperator(ctx context.Context, config Config, version string) error {
	{
		// Ensure priority class.
		priorityClass := v1.PriorityClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "giantswarm-critical",
			},
			Value:       1000000000,
			Description: "This priority class is used by giantswarm kubernetes components.",
		}

		_, err := config.K8sClients.K8sClient().SchedulingV1().PriorityClasses().Create(ctx, &priorityClass, metav1.CreateOptions{})
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		chartName := "chart-operator"
		tarballURL := fmt.Sprintf("https://giantswarm.github.com/control-plane-catalog/%s-%s.tgz", chartName, version)
		chartPackagePath, err := config.HelmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for %#q release is %#q", chartName, chartPackagePath))
		err = installChart(ctx, config, fmt.Sprintf("%s-%s", chartName, version), "", chartPackagePath)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	return nil
}

func installReleaseOperator(ctx context.Context, config Config, version string) error {
	{
		chartName := "release-operator"
		tarballURL := fmt.Sprintf("https://giantswarm.github.com/control-plane-catalog/%s-%s.tgz", chartName, version)
		chartPackagePath, err := config.HelmClient.PullChartTarball(ctx, tarballURL)
		if err != nil {
			return microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for %#q release is %#q", chartName, chartPackagePath))
		err = installChart(ctx, config, fmt.Sprintf("%s-%s", chartName, version), "", chartPackagePath)
		if err != nil {
			return microerror.Mask(err)
		}
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
