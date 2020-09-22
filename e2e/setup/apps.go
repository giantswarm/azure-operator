package setup

import (
	"context"
	"fmt"

	"github.com/giantswarm/microerror"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// The installAzureAppsCollection simulates the `opsctl install azure-app-collection` command by installing the CP
// apps needed by the e2e tests using helm.
func installAzureAppsCollection(ctx context.Context, config Config) error {
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

func installClusterService(ctx context.Context, config Config) error {
	// Steup RBAC.
	{
		clusterRoleBinding := v12.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "jwt-reviewer-cluster-service",
			},
			Subjects: []rbacv1.Subject{
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
