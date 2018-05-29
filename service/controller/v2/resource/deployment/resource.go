package deployment

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v2/cloudconfig"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

const (
	// Name is the identifier of the resource.
	Name = "deploymentv2"
)

type Config struct {
	CloudConfig *cloudconfig.CloudConfig
	Logger      micrologger.Logger

	Azure       setting.Azure
	AzureConfig client.AzureConfig
	// TemplateVersion is the ARM template version. Currently is the name
	// of the git branch in which the version is stored.
	TemplateVersion string
}

type Resource struct {
	cloudConfig *cloudconfig.CloudConfig
	logger      micrologger.Logger

	azure           setting.Azure
	azureConfig     client.AzureConfig
	templateVersion string
}

func New(config Config) (*Resource, error) {
	if config.CloudConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CloudConfig must not be empty", config)
	}
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
		return nil, microerror.Maskf(invalidConfigError, "%T.TemplateURIVersion must not be empty", config)
	}

	r := &Resource{
		cloudConfig: config.CloudConfig,
		logger:      config.Logger,

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

	resourceGroupName := key.ClusterID(customObject)
	mainDeployment, err := r.newMainDeployment(customObject)
	if err != nil {
		return microerror.Mask(err)
	}
	newDeployment := azureresource.Deployment{
		Properties: &azureresource.DeploymentProperties{
			Mode:       azureresource.Complete,
			Parameters: &mainDeployment.Parameters,
			TemplateLink: &azureresource.TemplateLink{
				URI:            to.StringPtr(mainDeployment.TemplateURI),
				ContentVersion: to.StringPtr(mainDeployment.TemplateContentVersion),
			},
		},
	}

	d, err := deploymentsClient.Get(ctx, resourceGroupName, mainDeployment.Name)
	if IsNotFound(err) {
		_, err := deploymentsClient.CreateOrUpdate(ctx, resourceGroupName, mainDeployment.Name, newDeployment)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment in progress")
		reconciliationcanceledcontext.SetCanceled(ctx)
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation for custom object")

		return nil
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		s := *d.Properties.ProvisioningState
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

		if !key.IsFinalProvisioningState(s) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

			return nil
		}
	}

	_, err = deploymentsClient.CreateOrUpdate(ctx, resourceGroupName, mainDeployment.Name, newDeployment)
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

func (r *Resource) getDeploymentsClient() (*azureresource.DeploymentsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.DeploymentsClient, nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) ([]deployment, error) {
	currentDeployments, err := toDeployments(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredDeployments, err := toDeployments(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var deploymentsToCreate []deployment

	for _, desiredDeployment := range desiredDeployments {
		if !existsDeploymentByName(currentDeployments, desiredDeployment.Name) {
			deploymentsToCreate = append(deploymentsToCreate, desiredDeployment)
		}
	}

	return deploymentsToCreate, nil
}

func existsDeploymentByName(list []deployment, name string) bool {
	for _, d := range list {
		if d.Name == name {
			return true
		}
	}

	return false
}

func getDeploymentByName(list []deployment, name string) (deployment, error) {
	for _, d := range list {
		if d.Name == name {
			return d, nil
		}
	}

	return deployment{}, microerror.Maskf(notFoundError, name)
}

func toDeployments(v interface{}) ([]deployment, error) {
	if v == nil {
		return []deployment{}, nil
	}

	deployments, ok := v.([]deployment)
	if !ok {
		return []deployment{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", []deployment{}, v)
	}

	return deployments, nil
}
