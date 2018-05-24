package deployment

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller"
	"github.com/giantswarm/operatorkit/controller/context/resourcecanceledcontext"

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

// GetCurrentState gets the current deployments for this cluster via the
// Azure API.
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Deleting the resource group will take care about cleaning
	// deployments.
	if key.IsDeleted(customObject) {
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling deployment deletion: deleted with the resource group")
		resourcecanceledcontext.SetCanceled(ctx)
		return nil, nil
	}

	resourceGroupName := key.ClusterID(customObject)
	deploymentClient, err := r.getDeploymentsClient()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var deployments []deployment
	{
		for _, deploymentName := range getDeploymentNames() {
			deploymentExtended, err := deploymentClient.Get(ctx, resourceGroupName, deploymentName)
			if err != nil {
				if client.ResponseWasNotFound(deploymentExtended.Response) {
					// Fall through.
					continue
				}

				return nil, microerror.Mask(err)
			}

			d := deployment{
				Name: *deploymentExtended.Name,
			}
			deployments = append(deployments, d)
		}
	}

	return deployments, nil
}

// GetDesiredState returns the desired deployments for this cluster.
func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	mainDeployment, err := r.newMainDeployment(customObject)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	deployments := []deployment{
		mainDeployment,
	}
	return deployments, nil
}

// NewUpdatePatch returns the deployments that should be created for this
// cluster.
func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	patch := controller.NewPatch()

	deploymentsToCreate, err := r.newCreateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch.SetCreateChange(deploymentsToCreate)
	return patch, nil
}

// NewDeletePatch returns an empty patch. Deployments are deleted together with
// the resource group.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*controller.Patch, error) {
	p := controller.NewPatch()
	return p, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

// ApplyCreateChange creates the deployments via the Azure API.
func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createState interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsToCreate, err := toDeployments(createState)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "creating Azure deployments")

	if len(deploymentsToCreate) != 0 {
		resourceGroupName := key.ClusterID(customObject)
		deploymentsClient, err := r.getDeploymentsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		for _, deploy := range deploymentsToCreate {
			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating Azure deployments: creating %#q", deploy.Name))

			d := azureresource.Deployment{
				Properties: &azureresource.DeploymentProperties{
					Mode:       azureresource.Complete,
					Parameters: &deploy.Parameters,
					TemplateLink: &azureresource.TemplateLink{
						URI:            to.StringPtr(deploy.TemplateURI),
						ContentVersion: to.StringPtr(deploy.TemplateContentVersion),
					},
				},
			}

			f, err := deploymentsClient.CreateOrUpdate(ctx, resourceGroupName, deploy.Name, d)
			if err != nil {
				return microerror.Mask(err)
			}
			err = f.WaitForCompletion(ctx, deploymentsClient.Client)
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("creating Azure deployments: creating %#q: created", deploy.Name))
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "creating Azure deployments: created")
	} else {
		r.logger.LogCtx(ctx, "level", "debug", "message", "creating Azure deployments: already created")
	}

	return nil
}

// ApplyDeleteChange is a noop. Deployments are deleted with the resource
// group.
func (r *Resource) ApplyDeleteChange(ctx context.Context, obj, deleteState interface{}) error {
	return nil
}

// ApplyUpdateChange is not yet implemented.
func (r *Resource) ApplyUpdateChange(ctx context.Context, obj, updateState interface{}) error {
	return nil
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
