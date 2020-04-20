package deployment

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/apiextensions/pkg/clientset/versioned"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/azure-operator/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/debugger"
	"github.com/giantswarm/azure-operator/service/controller/key"
	"github.com/giantswarm/azure-operator/service/controller/setting"
)

const (
	// Name is the identifier of the resource.
	Name = "deployment"
)

const (
	DeploymentTemplateChecksum   = "TemplateChecksum"
	DeploymentParametersChecksum = "ParametersChecksum"
	mainDeploymentName           = "cluster-main-template"
)

type Config struct {
	Debugger  *debugger.Debugger
	G8sClient versioned.Interface
	Logger    micrologger.Logger

	Azure setting.Azure
	// TemplateVersion is the ARM template version. Currently is the name
	// of the git branch in which the version is stored.
	TemplateVersion string
}

type Resource struct {
	debugger  *debugger.Debugger
	g8sClient versioned.Interface
	logger    micrologger.Logger

	azure           setting.Azure
	templateVersion string
}

func New(config Config) (*Resource, error) {
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.G8sClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.G8sClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TemplateVersion must not be empty", config)
	}

	r := &Resource{
		debugger:  config.Debugger,
		g8sClient: config.G8sClient,
		logger:    config.Logger,

		azure:           config.Azure,
		templateVersion: config.TemplateVersion,
	}

	return r, nil
}

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	var deployment azureresource.Deployment

	d, err := deploymentsClient.Get(ctx, key.ClusterID(cr), mainDeploymentName)
	if IsNotFound(err) {
		params := map[string]interface{}{
			"initialProvisioning": "Yes",
		}
		deployment, err = r.newDeployment(ctx, cr, params)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		s := *d.Properties.ProvisioningState

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

		if !key.IsSucceededProvisioningState(s) {
			r.debugger.LogFailedDeployment(ctx, d, err)
		}
		if !key.IsFinalProvisioningState(s) {
			reconciliationcanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
			return nil
		}

		params := map[string]interface{}{
			"initialProvisioning": "No",
		}
		deployment, err = r.newDeployment(ctx, cr, params)
		if err != nil {
			return microerror.Mask(err)
		}

		err = r.enrichControllerContext(ctx, cr)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	desiredDeploymentTemplateChk, err := getDeploymentTemplateChecksum(deployment)
	if err != nil {
		return microerror.Mask(err)
	}

	desiredDeploymentParametersChk, err := getDeploymentParametersChecksum(deployment)
	if err != nil {
		return microerror.Mask(err)
	}

	currentDeploymentTemplateChk, err := r.getResourceStatus(cr, DeploymentTemplateChecksum)
	if err != nil {
		return microerror.Mask(err)
	}

	currentDeploymentParametersChk, err := r.getResourceStatus(cr, DeploymentParametersChecksum)
	if err != nil {
		return microerror.Mask(err)
	}

	if currentDeploymentTemplateChk == desiredDeploymentTemplateChk && currentDeploymentParametersChk == desiredDeploymentParametersChk {
		r.logger.LogCtx(ctx, "level", "debug", "message", "template and parameters unchanged")
		// As current and desired state differs, start process from the beginning.
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "template or parameters changed")

	res, err := deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(cr), mainDeploymentName, deployment)
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment failed; deployment: %#v", deployment), "stack", microerror.Stack(microerror.Mask(err)))

		return microerror.Mask(err)
	}

	deploymentExtended, err := deploymentsClient.CreateOrUpdateResponder(res.Response())
	if err != nil {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment failed; deployment: %#v", deploymentExtended), "stack", microerror.Stack(microerror.Mask(err)))

		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured deployment")

	if desiredDeploymentTemplateChk != "" {
		err = r.setResourceStatus(cr, DeploymentTemplateChecksum, desiredDeploymentTemplateChk)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentTemplateChecksum, desiredDeploymentTemplateChk))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentTemplateChecksum))
	}

	if desiredDeploymentParametersChk != "" {
		err = r.setResourceStatus(cr, DeploymentParametersChecksum, desiredDeploymentParametersChk)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("set %s to '%s'", DeploymentParametersChecksum, desiredDeploymentParametersChk))
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("Unable to get a valid Checksum for %s", DeploymentParametersChecksum))
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
	reconciliationcanceledcontext.SetCanceled(ctx)

	return nil
}

// EnsureDeleted is a noop since the deletion of deployments is redirected to
// the deletion of resource groups because they garbage collect them.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) enrichControllerContext(ctx context.Context, customObject providerv1alpha1.AzureConfig) error {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "api_load_balancer_setup", "backendPoolId")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.APILBBackendPoolID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "etcd_load_balancer_setup", "backendPoolId")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.EtcdLBBackendPoolID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "virtual_network_setup", "masterSubnetID")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.MasterSubnetID = v
		}
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "virtual_network_setup", "workerSubnetID")
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			cc.WorkerSubnetID = v
		}
	}

	return nil
}

func (r *Resource) getDeploymentsClient(ctx context.Context) (*azureresource.DeploymentsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.DeploymentsClient, nil
}

func (r *Resource) getDeploymentOutputValue(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentName string, outputName string) (string, error) {
	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return "", microerror.Mask(err)
	}
	d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), deploymentName)
	if err != nil {
		return "", microerror.Mask(err)
	}

	if d.Properties.Outputs == nil {
		r.logger.LogCtx(ctx, "level", "warning", "message", fmt.Sprintf("cannot get output value '%s' of deployment '%s'", outputName, deploymentName))
		r.logger.LogCtx(ctx, "level", "warning", "message", "assuming deployment is in failed state")
		r.logger.LogCtx(ctx, "level", "warning", "message", "canceling controller context enrichment")
		return "", nil
	}

	m, err := key.ToMap(d.Properties.Outputs)
	if err != nil {
		return "", microerror.Mask(err)
	}
	v, ok := m[outputName]
	if !ok {
		return "", microerror.Maskf(missingOutputValueError, outputName)
	}
	m, err = key.ToMap(v)
	if err != nil {
		return "", microerror.Mask(err)
	}
	v, err = key.ToKeyValue(m)
	if err != nil {
		return "", microerror.Mask(err)
	}
	s, err := key.ToString(v)
	if err != nil {
		return "", microerror.Mask(err)
	}

	return s, nil
}

func (r *Resource) getResourceStatus(customObject providerv1alpha1.AzureConfig, t string) (string, error) {
	{
		c, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(customObject.Namespace).Get(customObject.Name, metav1.GetOptions{})
		if err != nil {
			return "", microerror.Mask(err)
		}

		customObject = *c
	}

	for _, r := range customObject.Status.Cluster.Resources {
		if r.Name != Name {
			continue
		}

		for _, c := range r.Conditions {
			if c.Type == t {
				return c.Status, nil
			}
		}
	}

	return "", nil
}

func (r *Resource) setResourceStatus(customObject providerv1alpha1.AzureConfig, t string, s string) error {
	// Get the newest CR version. Otherwise status update may fail because of:
	//
	//	 the object has been modified; please apply your changes to the
	//	 latest version and try again
	//
	{
		c, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(customObject.Namespace).Get(customObject.Name, metav1.GetOptions{})
		if err != nil {
			return microerror.Mask(err)
		}

		customObject = *c
	}

	resourceStatus := providerv1alpha1.StatusClusterResource{
		Conditions: []providerv1alpha1.StatusClusterResourceCondition{
			{
				Status: s,
				Type:   t,
			},
		},
		Name: Name,
	}

	var set bool
	for i, r := range customObject.Status.Cluster.Resources {
		if r.Name != Name {
			continue
		}

		for _, c := range r.Conditions {
			if c.Type == t {
				continue
			}
			resourceStatus.Conditions = append(resourceStatus.Conditions, c)
		}

		customObject.Status.Cluster.Resources[i] = resourceStatus
		set = true
	}

	if !set {
		customObject.Status.Cluster.Resources = append(customObject.Status.Cluster.Resources, resourceStatus)
	}

	{
		n := customObject.GetNamespace()
		_, err := r.g8sClient.ProviderV1alpha1().AzureConfigs(n).UpdateStatus(&customObject)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	return nil
}
