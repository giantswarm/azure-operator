package setup

import (
	"context"
	"fmt"

	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"

	"github.com/giantswarm/azure-operator/v4/e2e/env"
	"github.com/giantswarm/azure-operator/v4/e2e/key"
	"github.com/giantswarm/azure-operator/v4/pkg/project"
)

const (
	CatalogStorageURL     = "https://giantswarm.github.io/control-plane-catalog"
	TestCatalogStorageURL = "https://giantswarm.github.io/control-plane-test-catalog"
)

func pullLatestChart(ctx context.Context, config Config, chartName string, catalogURL string) (string, error) {
	var err error

	var latestRelease string
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("calculating latest %#q release version", chartName))

		o := func() error {
			latestRelease, err = appcatalog.GetLatestVersion(ctx, catalogURL, chartName)

			if latestRelease == "" {
				return invalidAppVersionError
			}

			return nil
		}
		n := backoff.NewNotifier(config.Logger, ctx)
		b := backoff.NewConstant(backoff.ShortMaxWait, backoff.ShortMaxInterval)
		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			return "", microerror.Mask(err)
		}
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("latest %#q release is %#q", chartName, latestRelease))
	}

	var latestReleaseChartPackagePath string
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("getting tarball URL for latest %#q release", chartName))
		latestReleaseTarballURL, err := appcatalog.NewTarballURL(catalogURL, chartName, latestRelease)
		if err != nil {
			return "", microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball URL for latest %#q release is %#q", chartName, latestReleaseTarballURL))
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("pulling tarball for latest %#q release", chartName))
		latestReleaseChartPackagePath, err = config.HelmClient.PullChartTarball(ctx, latestReleaseTarballURL)
		if err != nil {
			return "", microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for latest %#q release is %#q", chartName, latestReleaseChartPackagePath))
	}

	return latestReleaseChartPackagePath, err
}

func pullChartPackageUnderTest(ctx context.Context, config Config) (string, error) {
	config.Logger.LogCtx(ctx, "level", "debug", "message", "getting tarball URL for azure-operator tested version")
	operatorTarballURL, err := appcatalog.NewTarballURL(TestCatalogStorageURL, project.Name(), env.GetLatestOperatorRelease())
	if err != nil {
		return "", microerror.Mask(err)
	}
	config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball URL for azure-operator tested version is %#q", operatorTarballURL))

	config.Logger.LogCtx(ctx, "level", "debug", "message", "pulling tarball for azure-operator tested version")
	operatorTarballPath, err := config.HelmClient.PullChartTarball(ctx, operatorTarballURL)
	if err != nil {
		return "", microerror.Mask(err)
	}
	config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for azure-operator tested version is %#q", operatorTarballPath))

	return operatorTarballPath, err
}

func installLatestReleaseChartPackage(ctx context.Context, config Config, chartName, values string, catalogURL string) error {
	chartPackagePath, err := pullLatestChart(ctx, config, chartName, catalogURL)
	if err != nil {
		return microerror.Mask(err)
	}
	return installChart(ctx, config, chartName, values, chartPackagePath)
}

func installChartPackageBeingTested(ctx context.Context, config Config, values string) error {
	var err error
	chartPackagePath := env.OperatorHelmTarballPath()
	if chartPackagePath == "" {
		chartPackagePath, err = pullChartPackageUnderTest(ctx, config)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	helmReleaseName := fmt.Sprintf("%s-wip", project.Name())
	return installChart(ctx, config, helmReleaseName, values, chartPackagePath)
}

func installChart(ctx context.Context, config Config, releaseName, values, chartPackagePath string) error {
	defer func() {
		fs := afero.NewOsFs()
		err := fs.Remove(chartPackagePath)
		if err != nil {
			config.Logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %#q failed", chartPackagePath), "stack", fmt.Sprintf("%#v", err))
		}
	}()

	config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing %#q", releaseName))

	rawValues, err := valuesStrToMap(values)
	if err != nil {
		return microerror.Mask(err)
	}

	installOptions := helmclient.InstallOptions{
		Namespace:   key.Namespace(),
		ReleaseName: releaseName,
		Wait:        true,
	}

	err = config.HelmClient.InstallReleaseFromTarball(ctx,
		chartPackagePath,
		key.Namespace(),
		rawValues,
		installOptions)
	if err != nil {
		return microerror.Mask(err)
	}
	config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed %#q", releaseName))

	return err
}

func valuesStrToMap(values string) (map[string]interface{}, error) {
	rawValues, err := helmclient.MergeValues(map[string][]byte{"dest": []byte(values)}, map[string][]byte{"src": []byte{}})
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return rawValues, nil
}
