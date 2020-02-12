// +build k8srequired

package multiaz

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/giantswarm/apprclient"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/spf13/afero"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_AZ(t *testing.T) {
	err := multiaz.Test(context.Background())
	if err != nil {
		t.Fatalf("%#v", err)
	}
}

type Config struct {
	Logger   micrologger.Logger
	Provider *Provider
}

type MultiAZ struct {
	logger   micrologger.Logger
	provider *Provider
}

func New(config Config) (*MultiAZ, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.Provider == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Provider must not be empty", config)
	}

	s := &MultiAZ{
		logger:   config.Logger,
		provider: config.Provider,
	}

	return s, nil
}

func (s *MultiAZ) Test(ctx context.Context) error {
	s.logger.LogCtx(ctx, "level", "debug", "message", "getting current availability zones")
	zones, err := s.provider.GetClusterAZs(ctx)
	if err != nil {
		return microerror.Mask(err)
	}
	s.logger.LogCtx(ctx, "level", "debug", "message", "found availability zones", "azs", zones)

	if len(zones) != 1 {
		return microerror.Maskf(executionFailedError, "The amount of AZ's used is not correct. Expected %d zones, got %d zones", 1, len(zones))
	}
	if zones[0] != "1" {
		return microerror.Maskf(executionFailedError, "The AZ used is not correct. Expected %s, got %s", "1", zones[0])
	}

	return nil
}

func (s *MultiAZ) TestApp(ctx context.Context) error {
	var err error

	{
		s.logger.LogCtx(ctx, "level", "debug", "message", "installing e2e-app")

		err = s.InstallTestApp(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		s.logger.LogCtx(ctx, "level", "debug", "message", "installed e2e-app")
	}

	{
		s.logger.LogCtx(ctx, "level", "debug", "message", "checking test app is installed")

		err = s.CheckTestAppIsInstalled(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		s.logger.LogCtx(ctx, "level", "debug", "message", "test app is installed")
	}

	{
		resp, err := http.Get(fmt.Sprintf("http://helloworld.%s.k8s.godsmack.westeurope.azure.gigantic.io/", s.provider.clusterID))
		if err != nil {
			return microerror.Mask(err)
		}
		if resp.StatusCode != http.StatusOK {
			return microerror.Mask(err)
		}
	}

	//{
	//	s.provider.k8sClient.CoreV1().RESTClient().Post().Resource("pods").
	//		Name("podName").
	//		Namespace("default").
	//		SubResource("exec")
	//}

	{
		pvcName := "e2e-pvc"
		pvc := &v1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pvcName,
				Namespace: "default",
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{v1.ReadWriteOnce},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceName(v1.ResourceStorage): resource.MustParse("1Gi"),
					},
				},
			},
		}
		_, err := s.provider.k8sClient.CoreV1().PersistentVolumeClaims("default").Create(pvc)
		if err != nil {
			return microerror.Mask(err)
		}

		o := func() error {
			pvc, err := s.provider.k8sClient.CoreV1().PersistentVolumeClaims("default").Get(pvcName, metav1.GetOptions{})
			if err != nil {
				return microerror.Mask(err)
			}

			if pvc.Status.Phase != v1.ClaimBound {
				return microerror.Mask(pvcIsNotBound)
			}

			return nil
		}

		b := backoff.NewConstant(backoff.ShortMaxWait, backoff.ShortMaxInterval)
		n := func(err error, delay time.Duration) {
			s.logger.Log("level", "debug", "message", err.Error())
		}

		err = backoff.RetryNotify(o, b, n)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (s *MultiAZ) InstallTestApp(ctx context.Context) error {
	var err error

	var apprClient *apprclient.Client
	{
		c := apprclient.Config{
			Fs:     afero.NewOsFs(),
			Logger: s.logger,

			Address:      CNRAddress,
			Organization: CNROrganization,
		}

		apprClient, err = apprclient.New(c)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	var helmClient *helmclient.Client
	{
		c := helmclient.Config{
			Logger:    s.logger,
			K8sClient: s.legacyFramework.K8sClient(),

			RestConfig: s.legacyFramework.RestConfig(),
		}

		helmClient, err = helmclient.New(c)
		if err != nil {
			return microerror.Mask(err)
		}

		err = helmClient.EnsureTillerInstalled(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	// Install the e2e app chart in the guest cluster.
	{
		s.logger.Log("level", "debug", "message", "installing e2e-app for testing")

		tarballPath, err := apprClient.PullChartTarball(ctx, ChartName, ChartChannel)
		if err != nil {
			return microerror.Mask(err)
		}

		err = helmClient.InstallReleaseFromTarball(ctx, tarballPath, ChartNamespace)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (s *MultiAZ) CheckTestAppIsInstalled(ctx context.Context) error {
	var podCount = 2

	s.logger.Log("level", "debug", "message", fmt.Sprintf("waiting for %d pods of the e2e-app to be up", podCount))

	o := func() error {
		lo := metav1.ListOptions{
			LabelSelector: "app=e2e-app",
		}
		l, err := s.legacyFramework.K8sClient().CoreV1().Pods(ChartNamespace).List(lo)
		if err != nil {
			return microerror.Mask(err)
		}
		if len(l.Items) != podCount {
			return microerror.Maskf(waitError, "want %d pods found %d", podCount, len(l.Items))
		}

		return nil
	}

	b := backoff.NewConstant(backoff.ShortMaxWait, backoff.ShortMaxInterval)
	n := func(err error, delay time.Duration) {
		s.logger.Log("level", "debug", "message", err.Error())
	}

	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	s.logger.Log("level", "debug", "message", fmt.Sprintf("found %d pods of the e2e-app", podCount))

	return nil
}
