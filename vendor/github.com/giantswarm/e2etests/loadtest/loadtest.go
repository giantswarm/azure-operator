package loadtest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1"
	"github.com/giantswarm/apprclient"
	"github.com/giantswarm/backoff"
	"github.com/giantswarm/helmclient"
	"github.com/giantswarm/k8sclient"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	yaml "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/helm/pkg/helm"
)

type Config struct {
	ApprClient   apprclient.Interface
	CPClients    *k8sclient.Clients
	CPHelmClient *helmclient.Client
	Logger       micrologger.Logger
	TCClients    *k8sclient.Clients
	TCHelmClient *helmclient.Client

	ClusterID            string
	CommonDomain         string
	StormForgerAuthToken string
}

type LoadTest struct {
	apprClient   apprclient.Interface
	cpClients    *k8sclient.Clients
	cpHelmClient *helmclient.Client
	logger       micrologger.Logger
	tcClients    *k8sclient.Clients
	tcHelmClient *helmclient.Client

	clusterID            string
	commonDomain         string
	stormForgerAuthToken string
}

func New(config Config) (*LoadTest, error) {
	if config.ApprClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.ApprClient must not be empty", config)
	}
	if config.CPClients == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CPClients must not be empty", config)
	}
	if config.CPHelmClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CPHelmClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.TCClients == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.TCClients must not be empty", config)
	}
	if config.TCHelmClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.TCHelmClient must not be empty", config)
	}

	if config.ClusterID == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.ClusterID must not be empty", config)
	}
	if config.CommonDomain == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.CommonDomain must not be empty", config)
	}
	if config.StormForgerAuthToken == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.StormForgerAuthToken must not be empty", config)
	}

	s := &LoadTest{
		apprClient:   config.ApprClient,
		cpClients:    config.CPClients,
		cpHelmClient: config.CPHelmClient,
		logger:       config.Logger,
		tcClients:    config.TCClients,
		tcHelmClient: config.TCHelmClient,

		clusterID:            config.ClusterID,
		commonDomain:         config.CommonDomain,
		stormForgerAuthToken: config.StormForgerAuthToken,
	}

	return s, nil
}

func (l *LoadTest) Test(ctx context.Context) error {
	var err error

	var loadTestEndpoint string
	{
		loadTestEndpoint = fmt.Sprintf("loadtest-app.%s.%s", l.clusterID, l.commonDomain)

		l.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("loadtest-app endpoint is %#q", loadTestEndpoint))
	}

	{
		l.logger.LogCtx(ctx, "level", "debug", "message", "waiting for tenant cluster kubernetes API to be up")

		err = l.waitForAPIUp(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		l.logger.LogCtx(ctx, "level", "debug", "message", "waited for tenant cluster kubernetes API to be up")
	}

	{
		l.logger.LogCtx(ctx, "level", "debug", "message", "installing loadtest app")

		err = l.installTestApp(ctx, loadTestEndpoint)
		if err != nil {
			return microerror.Mask(err)
		}

		l.logger.LogCtx(ctx, "level", "debug", "message", "installed loadtest app")
	}

	{
		l.logger.LogCtx(ctx, "level", "debug", "message", "waiting for loadtest app to be ready")

		err = l.waitForLoadTestApp(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		l.logger.LogCtx(ctx, "level", "debug", "message", "loadtest app is ready")
	}

	{
		l.logger.LogCtx(ctx, "level", "debug", "message", "enabling HPA for Nginx Ingress Controller")

		err = l.enableIngressControllerHPA(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		l.logger.LogCtx(ctx, "level", "debug", "message", "enabled HPA for Nginx Ingress Controller")
	}

	{
		l.logger.LogCtx(ctx, "level", "debug", "message", "installing loadtest job")

		err = l.installLoadTestJob(ctx, loadTestEndpoint)
		if err != nil {
			return microerror.Mask(err)
		}

		l.logger.LogCtx(ctx, "level", "debug", "message", "installed loadtest job")
	}

	var jsonResults []byte

	{
		l.logger.LogCtx(ctx, "level", "debug", "message", "waiting for loadtest job to complete")

		jsonResults, err = l.waitForLoadTestJob(ctx)
		if err != nil {
			return microerror.Mask(err)
		}

		l.logger.LogCtx(ctx, "level", "debug", "message", "loadtest job is complete")
	}

	{
		l.logger.LogCtx(ctx, "level", "debug", "message", "checking loadtest results")

		err = l.checkLoadTestResults(ctx, jsonResults)
		if err != nil {
			return microerror.Mask(err)
		}

		l.logger.LogCtx(ctx, "level", "debug", "message", "checked loadtest results")
	}

	return nil
}

// checkLoadTestResults parses the load test results JSON and determines if the
// test was successful or not.
func (l *LoadTest) checkLoadTestResults(ctx context.Context, jsonResults []byte) error {
	var err error

	l.logger.LogCtx(ctx, "level", "debug", "message", "checking loadtest results")

	l.logger.LogCtx(ctx, "level", "debug", "message", jsonResults)

	var results LoadTestResults

	err = json.Unmarshal(jsonResults, &results)
	if err != nil {
		return microerror.Mask(err)
	}

	apdexScore := results.Data.Attributes.BasicStatistics.Apdex75

	if apdexScore < ApdexPassThreshold {
		return microerror.Maskf(failedLoadTestError, "apdex score of %f is less than %f", apdexScore, ApdexPassThreshold)
	}

	l.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("load test passed: apdex score of %f is >= %f", apdexScore, ApdexPassThreshold))

	return nil
}

// enableIngressControllerHPA enables HPA via the user configmap and updates
// the chartconfig CR so chart-operator reconciles the config change.
func (l *LoadTest) enableIngressControllerHPA(ctx context.Context) error {
	var err error

	l.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waiting for %#q configmap to be created", UserConfigMapName))

	o := func() error {
		lo := metav1.ListOptions{
			LabelSelector: fmt.Sprintf("app=%s,cluster-operator.giantswarm.io/configmap-type=user", UserConfigMapName),
		}
		l, err := l.tcClients.K8sClient().CoreV1().ConfigMaps(metav1.NamespaceDefault).List(lo)
		if err != nil {
			return microerror.Mask(err)
		}
		if len(l.Items) != 1 {
			return microerror.Maskf(waitError, "want %d configmaps found %d", 1, len(l.Items))
		}

		return nil
	}

	b := backoff.NewConstant(10*time.Minute, 15*time.Second)
	n := func(err error, delay time.Duration) {
		l.logger.LogCtx(ctx, "level", "debug", "message", err.Error())
	}

	err = backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	l.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("waited for %#q configmap to be created", UserConfigMapName))

	var data []byte

	{
		l.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("patching %#q configmap", UserConfigMapName))

		values := map[string]interface{}{
			"autoscaling-enabled": true,
		}

		data, err = yaml.Marshal(values)
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = l.tcClients.K8sClient().CoreV1().ConfigMaps(metav1.NamespaceSystem).Patch(UserConfigMapName, types.StrategicMergePatchType, data)
		if err != nil {
			return microerror.Mask(err)
		}

		l.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("patched %#q configmap", UserConfigMapName))
	}

	var cr *v1alpha1.ChartConfig

	{
		l.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updating %#q chartconfig CR", CustomResourceName))

		cr, err = l.tcClients.G8sClient().CoreV1alpha1().ChartConfigs(CustomResourceNamespace).Get(CustomResourceName, metav1.GetOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		// Set dummy annotation to trigger an update event.
		annotations := cr.Annotations
		annotations["test"] = "test"
		cr.Annotations = annotations

		_, err = l.tcClients.G8sClient().CoreV1alpha1().ChartConfigs(CustomResourceNamespace).Update(cr)
		if err != nil {
			return microerror.Mask(err)
		}

		l.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("updated %#q chartconfig CR", CustomResourceName))
	}

	return nil
}

// installChart is a helper method for installing helm charts.
func (l *LoadTest) installChart(ctx context.Context, helmClient *helmclient.Client, chartName string, jsonValues []byte) error {
	var err error
	var tarballPath string

	{
		l.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installing %#q", chartName))

		tarballPath, err = l.apprClient.PullChartTarball(ctx, chartName, ChartChannel)
		if err != nil {
			return microerror.Mask(err)
		}

		err = helmClient.InstallReleaseFromTarball(ctx, tarballPath, ChartNamespace, helm.ValueOverrides(jsonValues))
		if err != nil {
			return microerror.Mask(err)
		}

		l.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("installed %#q", chartName))
	}

	return nil
}

// installLoadTestJob installs a chart that creates a job that uses the
// Stormforger CLI to trigger the load test.
func (l *LoadTest) installLoadTestJob(ctx context.Context, loadTestEndpoint string) error {
	var err error

	var jsonValues []byte
	{
		values := map[string]interface{}{
			"auth": map[string]string{
				"token": l.stormForgerAuthToken,
			},
			"test": map[string]string{
				"endpoint": loadTestEndpoint,
				"name":     TestName,
			},
		}

		jsonValues, err = json.Marshal(values)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = l.installChart(ctx, l.cpHelmClient, JobChartName, jsonValues)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

// installLoadTestApp installs a chart that deploys the Stormforger test app
// in the tenant cluster as the test workload for the load test.
func (l *LoadTest) installTestApp(ctx context.Context, loadTestEndpoint string) error {
	var err error

	var jsonValues []byte
	{
		values := map[string]interface{}{
			"ingress": map[string]interface{}{
				"hosts": []string{
					loadTestEndpoint,
				},
			},
		}

		jsonValues, err = json.Marshal(values)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = l.tcHelmClient.EnsureTillerInstalled(ctx)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	{
		err = l.installChart(ctx, l.tcHelmClient, AppChartName, jsonValues)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}

func (l *LoadTest) waitForAPIUp(ctx context.Context) error {
	l.logger.LogCtx(ctx, "level", "debug", "message", "waiting for k8s API to be up")

	o := func() error {
		_, err := l.tcClients.K8sClient().CoreV1().Services(metav1.NamespaceDefault).Get("kubernetes", metav1.GetOptions{})
		if err != nil {
			return microerror.Maskf(waitError, err.Error())
		}

		return nil
	}
	b := backoff.NewConstant(40*time.Minute, 30*time.Second)
	n := func(err error, delay time.Duration) {
		l.logger.LogCtx(ctx, "level", "debug", "stack", err.Error())
	}

	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	l.logger.LogCtx(ctx, "level", "debug", "message", "k8s API is up")

	return nil
}

// waitForLoadTestApp waits for all pods of the test app to be ready.
func (l *LoadTest) waitForLoadTestApp(ctx context.Context) error {
	l.logger.LogCtx(ctx, "level", "debug", "message", "waiting for loadtest-app deployment to be ready")

	o := func() error {
		lo := metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=loadtest-app",
		}
		l, err := l.tcClients.K8sClient().AppsV1().Deployments(metav1.NamespaceDefault).List(lo)
		if err != nil {
			return microerror.Mask(err)
		}
		if len(l.Items) != 1 {
			return microerror.Maskf(waitError, "want %d deployments found %d", 1, len(l.Items))
		}

		deploy := l.Items[0]
		if *deploy.Spec.Replicas == deploy.Status.ReadyReplicas {
			return microerror.Maskf(waitError, "want %d ready pods found %d", deploy.Spec.Replicas, deploy.Status.ReadyReplicas)
		}

		return nil
	}

	b := backoff.NewConstant(2*time.Minute, 15*time.Second)
	n := func(err error, delay time.Duration) {
		l.logger.LogCtx(ctx, "level", "debug", "message", err.Error())
	}

	err := backoff.RetryNotify(o, b, n)
	if err != nil {
		return microerror.Mask(err)
	}

	l.logger.LogCtx(ctx, "level", "debug", "message", "waited for loadtest-app deployment to be ready")

	return nil
}

// waitForLoadTestJob waits for the job running the Stormforger CLI to
// complete and then gets the pod logs which contains the results JSON. The CLI
// is configured to wait for the load test to complete.
func (l *LoadTest) waitForLoadTestJob(ctx context.Context) ([]byte, error) {
	var podCount = 1
	var podName = ""

	{
		l.logger.Log("level", "debug", "message", "waiting for stormforger-cli job")

		o := func() error {
			lo := metav1.ListOptions{
				FieldSelector: "status.phase=Succeeded",
				LabelSelector: "app.kubernetes.io/name=stormforger-cli",
			}
			l, err := l.cpClients.K8sClient().CoreV1().Pods(metav1.NamespaceDefault).List(lo)
			if err != nil {
				return microerror.Mask(err)
			}

			if len(l.Items) == podCount {
				podName = l.Items[0].Name

				return nil
			}

			return microerror.Maskf(waitError, "want %d Succeeded pods found %d", podCount, len(l.Items))
		}

		b := backoff.NewConstant(20*time.Minute, 30*time.Second)
		n := func(err error, delay time.Duration) {
			l.logger.Log("level", "debug", "message", err.Error())
		}

		err := backoff.RetryNotify(o, b, n)
		if err != nil {
			return nil, microerror.Mask(err)
		}

		l.logger.Log("level", "debug", "message", "waited for stormforger-cli job")
	}

	var results []byte

	{
		l.logger.Log("level", "debug", "message", "getting results from pod logs")

		req := l.cpClients.K8sClient().CoreV1().Pods(metav1.NamespaceDefault).GetLogs(podName, &corev1.PodLogOptions{})

		readCloser, err := req.Stream()
		if err != nil {
			return nil, err
		}

		defer readCloser.Close()

		buf := new(bytes.Buffer)

		_, err = io.Copy(buf, readCloser)
		if err != nil {
			return nil, err
		}

		results = buf.Bytes()

		l.logger.Log("level", "debug", "message", "got results from pod logs")
	}

	return results, nil
}
