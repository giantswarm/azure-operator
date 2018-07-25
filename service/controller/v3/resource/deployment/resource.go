package deployment

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v3/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v3/debugger"
	"github.com/giantswarm/azure-operator/service/controller/v3/key"
)

const (
	// Name is the identifier of the resource.
	Name = "deploymentv3"
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
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	var deployment azureresource.Deployment

	d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), mainDeploymentName)
	if IsNotFound(err) {
		params := map[string]interface{}{
			"initialProvisioning": "Yes",
		}
		deployment, err = r.newDeployment(ctx, customObject, params)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", *d.Properties.ProvisioningState))

		if !key.IsFinalProvisioningState(*d.Properties.ProvisioningState) {
			r.debugger.LogFailedDeployment(ctx, d)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource")

			return nil
		}

		params := map[string]interface{}{
			"initialProvisioning": "No",
		}
		deployment, err = r.newDeployment(ctx, customObject, params)
		if err != nil {
			return microerror.Mask(err)
		}

		err = r.enrichControllerContext(ctx, customObject)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	_, err = deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(customObject), mainDeploymentName, deployment)
	if err != nil {
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
		if err != nil {
			return microerror.Mask(err)
		}
		cc.APILBBackendPoolID = v
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "etcd_load_balancer_setup", "backendPoolId")
		if err != nil {
			return microerror.Mask(err)
		}
		cc.EtcdLBBackendPoolID = v
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "virtual_network_setup", "masterSubnetID")
		if err != nil {
			return microerror.Mask(err)
		}
		cc.MasterSubnetID = v
	}

	{
		v, err := r.getDeploymentOutputValue(ctx, customObject, "virtual_network_setup", "workerSubnetID")
		if err != nil {
			return microerror.Mask(err)
		}
		cc.WorkerSubnetID = v
	}

	return nil
}

func (r *Resource) getDeploymentsClient(ctx context.Context) (*azureresource.DeploymentsClient, error) {
	sc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.DeploymentsClient, nil
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
