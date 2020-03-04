package deployment

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

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
	mainDeploymentName = "cluster-main-template"
)

type Config struct {
	Debugger *debugger.Debugger
	Logger   micrologger.Logger

	Azure setting.Azure
	// TemplateVersion is the ARM template version. Currently is the name
	// of the git branch in which the version is stored.
	TemplateVersion string
}

type Resource struct {
	debugger *debugger.Debugger
	logger   micrologger.Logger

	azure           setting.Azure
	templateVersion string
}

func New(config Config) (*Resource, error) {
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
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
		debugger: config.Debugger,
		logger:   config.Logger,

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
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")
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
