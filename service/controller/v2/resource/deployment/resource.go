package deployment

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v2/controllercontext"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

const (
	// Name is the identifier of the resource.
	Name = "deploymentv2"
)

const (
	mainDeploymentName = "cluster-main-template"
)

type Config struct {
	Logger micrologger.Logger

	Azure       setting.Azure
	AzureConfig client.AzureClientSetConfig
	// TemplateVersion is the ARM template version. Currently is the name
	// of the git branch in which the version is stored.
	TemplateVersion string
}

type Resource struct {
	logger micrologger.Logger

	azure           setting.Azure
	azureConfig     client.AzureClientSetConfig
	templateVersion string
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureConfig.%s", config, err)
	}
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TemplateVersion must not be empty", config)
	}

	r := &Resource{
		logger: config.Logger,

		azure:           config.Azure,
		azureConfig:     config.AzureConfig,
		templateVersion: config.TemplateVersion,
	}

	return r, nil
}

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsClient, err := r.getDeploymentsClient()
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
		s := *d.Properties.ProvisioningState
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

		if !key.IsFinalProvisioningState(s) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

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

func (r *Resource) getDeploymentsClient() (*azureresource.DeploymentsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.DeploymentsClient, nil
}

func (r *Resource) getDeploymentOutputValue(ctx context.Context, customObject providerv1alpha1.AzureConfig, deploymentName string, outputName string) (string, error) {
	deploymentsClient, err := r.getDeploymentsClient()
	if err != nil {
		return "", microerror.Mask(err)
	}
	d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), deploymentName)
	if err != nil {
		return "", microerror.Mask(err)
	}

	m, ok := d.Properties.Outputs.(map[string]interface{})
	if !ok {
		return "", microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, d.Properties.Outputs)
	}
	fmt.Printf("0: %#v\n", m)
	v, ok := m[outputName]
	if !ok {
		return "", microerror.Maskf(missingOutputValueError, outputName)
	}
	fmt.Printf("1: %#v\n", v)
	m, ok = v.(map[string]interface{})
	if !ok {
		return "", microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", map[string]interface{}{}, d.Properties.Outputs)
	}
	fmt.Printf("2: %#v\n", m)
	v, ok := m["Value"]
	if !ok {
		return "", microerror.Maskf(missingOutputValueError, outputName)
	}
	fmt.Printf("3: %#v\n", v)

	return s, nil
}
