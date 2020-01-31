package basicapp

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/e2etests/basicapp/legacyresource"
)

type Config struct {
	Clients    Clients
	HelmClient *helmclient.Client
	Logger     micrologger.Logger

	App            Chart
	ChartResources ChartResources
}

type BasicApp struct {
	clients    Clients
	helmClient *helmclient.Client
	logger     micrologger.Logger
	resource   *legacyresource.Resource

	chart          Chart
	chartResources ChartResources
}

func New(config Config) (*BasicApp, error) {
	var err error

	if config.HelmClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.HelmClient must not be empty", config)
	}
	if config.Clients == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Clients must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	err = config.App.Validate()
	if err != nil {
		return nil, microerror.Mask(err)
	}
	err = config.ChartResources.Validate()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var resource *legacyresource.Resource
	{
		c := legacyresource.Config{
			HelmClient: config.HelmClient,
			Logger:     config.Logger,
			Namespace:  config.App.Namespace,
		}

		resource, err = legacyresource.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	b := &BasicApp{
		clients:    config.Clients,
		helmClient: config.HelmClient,
		logger:     config.Logger,
		resource:   resource,

		chart:          config.App,
		chartResources: config.ChartResources,
	}

	return b, nil
}

func (b *BasicApp) Test(ctx context.Context) error {
	var err error

	{
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing chart %#q", b.chart.Name))

		err = b.resource.Install(b.chart.Name, b.chart.URL, b.chart.ChartValues)
		if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed chart %#q", b.chart.Name))
	}

	{
		b.logger.LogCtx(ctx, "level", "debug", "message", "waiting for deployed status")

		err = b.resource.WaitForStatus(b.chart.Name, "DEPLOYED")
		if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", "chart is deployed")
	}

	{
		b.logger.LogCtx(ctx, "level", "debug", "message", "checking resources")

		for _, ds := range b.chartResources.DaemonSets {
			b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking daemonset %#q", ds.Name))

			err = b.checkDaemonSet(ds)
			if err != nil {
				return microerror.Mask(err)
			}

			b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("daemonset %#q is correct", ds.Name))
		}

		for _, d := range b.chartResources.Deployments {
			b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking deployment %#q", d.Name))

			err = b.checkDeployment(ctx, d)
			if err != nil {
				return microerror.Mask(err)
			}

			b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment %#q is correct", d.Name))
		}

		for _, s := range b.chartResources.Services {
			b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("checking service %#q", s.Name))

			err = b.checkService(ctx, s)
			if err != nil {
				return microerror.Mask(err)
			}

			b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("service %#q is correct", s.Name))
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", "resources are correct")
	}

	{
		if !b.chart.RunReleaseTests {
			return nil
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", "running release tests")

		err = b.helmClient.RunReleaseTest(ctx, b.chart.Name)
		if err != nil {
			return microerror.Mask(err)
		}

		b.logger.LogCtx(ctx, "level", "debug", "message", "release tests passed")
	}

	return nil
}

// checkDaemonSet ensures that key properties of the daemonset are correct.
func (b *BasicApp) checkDaemonSet(expectedDaemonSet DaemonSet) error {
	ds, err := b.clients.K8sClient().AppsV1().DaemonSets(expectedDaemonSet.Namespace).Get(expectedDaemonSet.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return microerror.Maskf(notFoundError, "daemonset %#q", expectedDaemonSet.Name)
	} else if err != nil {
		return microerror.Mask(err)
	}

	err = b.checkLabels("daemonset labels", expectedDaemonSet.Labels, ds.ObjectMeta.Labels)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.checkLabels("daemonset matchLabels", expectedDaemonSet.MatchLabels, ds.Spec.Selector.MatchLabels)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.checkLabels("daemonset pod labels", expectedDaemonSet.Labels, ds.Spec.Template.ObjectMeta.Labels)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// checkDeployment ensures that key properties of the deployment are correct.
func (b *BasicApp) checkDeployment(ctx context.Context, expectedDeployment Deployment) error {

	o := func() error {
		err := b.checkDeploymentReady(ctx, expectedDeployment)
		if err != nil {
			return microerror.Mask(err)
		}

		return nil
	}

	off := backoff.NewConstant(30*time.Second, 5*time.Second)
	n := func(err error, delay time.Duration) {
		b.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("%#q deployment is not ready retrying in %s", expectedDeployment.Name, delay), "stack", fmt.Sprintf("%#v", err))
	}

	err := backoff.RetryNotify(o, off, n)
	if err != nil {
		return microerror.Mask(err)
	}

	ds, err := b.clients.K8sClient().AppsV1().Deployments(expectedDeployment.Namespace).Get(expectedDeployment.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return microerror.Maskf(notFoundError, "deployment: %#q", expectedDeployment.Name)
	} else if err != nil {
		return microerror.Mask(err)
	}

	err = b.checkLabels("deployment labels", expectedDeployment.DeploymentLabels, ds.ObjectMeta.Labels)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.checkLabels("deployment matchLabels", expectedDeployment.MatchLabels, ds.Spec.Selector.MatchLabels)
	if err != nil {
		return microerror.Mask(err)
	}

	err = b.checkLabels("deployment pod labels", expectedDeployment.PodLabels, ds.Spec.Template.ObjectMeta.Labels)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// checkDeploymentReady checks for the specified deployment that the number of
// ready replicas matches the desired state.
func (b *BasicApp) checkDeploymentReady(ctx context.Context, expectedDeployment Deployment) error {
	deploy, err := b.clients.K8sClient().AppsV1().Deployments(expectedDeployment.Namespace).Get(expectedDeployment.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return microerror.Maskf(notReadyError, "deployment %#q in not found", expectedDeployment.Name, expectedDeployment.Namespace)
	} else if err != nil {
		return microerror.Mask(err)
	}

	if deploy.Status.ReadyReplicas != *deploy.Spec.Replicas {
		return microerror.Maskf(notReadyError, "deployment %#q want %d replicas %d ready", expectedDeployment.Name, *deploy.Spec.Replicas, deploy.Status.ReadyReplicas)
	}

	// Deployment is ready.
	return nil
}

func (b *BasicApp) checkLabels(labelType string, expectedLabels, labels map[string]string) error {
	if !reflect.DeepEqual(expectedLabels, labels) {
		b.logger.Log("level", "debug", "message", fmt.Sprintf("expected %s: %v got: %v", labelType, expectedLabels, labels))
		return microerror.Maskf(invalidLabelsError, "%s do not match expected labels", labelType)
	}

	return nil
}

// checkService ensures that key properties of the service are correct.
func (b *BasicApp) checkService(ctx context.Context, expectedService Service) error {

	s, err := b.clients.K8sClient().CoreV1().Services(expectedService.Namespace).Get(expectedService.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return microerror.Maskf(notFoundError, "service: %#q", expectedService.Name)
	} else if err != nil {
		return microerror.Mask(err)
	}

	err = b.checkLabels("service labels", expectedService.Labels, s.ObjectMeta.Labels)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
