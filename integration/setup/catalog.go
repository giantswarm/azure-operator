package setup

import (
	"context"
	"fmt"

	"github.com/giantswarm/appcatalog"
	"github.com/giantswarm/microerror"
	"github.com/spf13/afero"
	"k8s.io/helm/pkg/helm"

	"github.com/giantswarm/azure-operator/integration/env"
	"github.com/giantswarm/azure-operator/integration/key"
	"github.com/giantswarm/azure-operator/pkg/project"
)

const (
	CatalogStorageURL     = "https://giantswarm.github.io/control-plane-catalog/"
	TestCatalogStorageURL = "https://giantswarm.github.io/control-plane-test-catalog/"
)

var (
	latestOperatorRelease string
)

func GetLatestOperatorRelease() string {
	return latestOperatorRelease
}

func init() {
	fmt.Printf("calculating latest %#q release\n", project.Name())

	var err error
	latestOperatorRelease, err = appcatalog.GetLatestVersion(context.Background(), CatalogStorageURL, project.Name())
	if err != nil {
		panic(fmt.Sprintln("cannot calculate latest operator release from app catalog"))
	}

	fmt.Printf("latest %#q release is %#q\n", project.Name(), latestOperatorRelease)
}

func pullLatestReleaseChartPackage(ctx context.Context, config Config, chartName string) (string, error) {
	var err error

	var latestRelease string
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("calculating latest %s release version", chartName))
		latestRelease, err = appcatalog.GetLatestVersion(ctx, CatalogStorageURL, chartName)
		if err != nil {
			return "", microerror.Mask(err)
		}
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("latest %s release is %s", chartName, latestRelease))
	}

	var latestReleaseChartPackagePath string
	{
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("getting tarball URL for latest %s release", chartName))
		latestReleaseTarballURL, err := appcatalog.NewTarballURL(CatalogStorageURL, chartName, latestRelease)
		if err != nil {
			return "", microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball URL for latest %s release is %s", chartName, latestReleaseTarballURL))
		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("pulling tarball for latest %s release", chartName))
		latestReleaseChartPackagePath, err = config.HelmClient.PullChartTarball(ctx, latestReleaseTarballURL)
		if err != nil {
			return "", microerror.Mask(err)
		}

		config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("tarball path for latest %s release is %s", chartName, latestReleaseChartPackagePath))
	}

	return latestReleaseChartPackagePath, err
}

func pullChartPackageUnderTest(ctx context.Context, config Config) (string, error) {
	config.Logger.LogCtx(ctx, "level", "debug", "message", "getting tarball URL for azure-operator tested version")
	operatorTarballURL, err := appcatalog.NewTarballURL(TestCatalogStorageURL, project.Name(), fmt.Sprintf("%s-%s", latestOperatorRelease, env.CircleSHA()))
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

func installLatestReleaseChartPackage(ctx context.Context, config Config, chartName, values string) error {
	chartPackagePath, err := pullLatestReleaseChartPackage(ctx, config, chartName)
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

	releaseName := fmt.Sprintf("%s-wip", project.Name())
	return installChart(ctx, config, releaseName, values, chartPackagePath)
}

func installChart(ctx context.Context, config Config, releaseName, values, chartPackagePath string) error {
	defer func() {
		fs := afero.NewOsFs()
		err := fs.Remove(chartPackagePath)
		if err != nil {
			config.Logger.LogCtx(ctx, "level", "error", "message", fmt.Sprintf("deletion of %s failed", chartPackagePath), "stack", fmt.Sprintf("%#v", err))
		}
	}()

	config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing %s", releaseName))
	err := config.HelmClient.InstallReleaseFromTarball(ctx,
		chartPackagePath,
		key.Namespace(),
		helm.ReleaseName(releaseName),
		helm.ValueOverrides([]byte(values)),
		helm.InstallWait(true))
	if err != nil {
		return microerror.Mask(err)
	}
	config.Logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed %s", releaseName))

	return err
}
