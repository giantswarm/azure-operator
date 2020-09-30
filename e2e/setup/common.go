package setup

import (
	"context"
	"fmt"

	releasev1alpha1 "github.com/giantswarm/apiextensions/v2/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	v1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// common installs components required to run the operator.
func common(ctx context.Context, config Config, giantSwarmRelease releasev1alpha1.Release) error {
	{
		err := config.K8s.EnsureNamespaceCreated(ctx, namespace)
		if err != nil {
			return microerror.Mask(err)
		}
	}
	{
		err := config.K8s.EnsureNamespaceCreated(ctx, OrganizationNamespace)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Install Vault.
	{
		err := installVault(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Ensure draughtsman configmap and secret.
	{
		err := ensureDraughtsman(ctx, config)
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

	// Install Azure Apps Collection.
	{
		err := installAzureAppsCollection(ctx, config)
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
