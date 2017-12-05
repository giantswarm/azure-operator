package deployment

import (
	"context"
	"fmt"
	"time"

	azureresource "github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/certificatetpr"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/cloudconfig"
	"github.com/giantswarm/azure-operator/service/key"
)

const (
	// Name is the identifier of the resource.
	Name = "deployment"

	createTimeout = 30 * time.Minute
)

type Config struct {
	// Dependencies.

	CertWatcher certificatetpr.Searcher
	CloudConfig *cloudconfig.CloudConfig
	Logger      micrologger.Logger

	// Settings.

	AzureConfig client.AzureConfig
	// TemplateVersion is the ARM template version. Currently is the name
	// of the git branch in which the version is stored.
	TemplateVersion string
}

// DefaultConfig provides a default configuration to create a new resource by
// best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.

		CertWatcher: nil,
		Logger:      nil,

		// Settings.

		AzureConfig:     client.DefaultAzureConfig(),
		TemplateVersion: "",
	}
}

type Resource struct {
	// Dependencies.

	certWatcher certificatetpr.Searcher
	cloudConfig *cloudconfig.CloudConfig
	logger      micrologger.Logger

	// Settings.

	azureConfig     client.AzureConfig
	templateVersion string
}

// New creates a new configured deploy resource.
func New(config Config) (*Resource, error) {
	// Dependencies.
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig.%s", err)
	}
	if config.CertWatcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.CertWatcher must not be empty")
	}
	if config.CloudConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.CloudConfig must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}
	// Settings.
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "config.TemplateURIVersion must not be empty")
	}

	r := &Resource{
		// Dependencies.
		certWatcher: config.CertWatcher,
		cloudConfig: config.CloudConfig,
		logger:      config.Logger.With("resource", Name),

		// Settings.
		azureConfig:     config.AzureConfig,
		templateVersion: config.TemplateVersion,
	}

	return r, nil
}

// GetCurrentState gets the current deployments for this cluster via the
// Azure API.
func (r *Resource) GetCurrentState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	resourceGroupName := key.ClusterID(customObject)
	deploymentClient, err := r.getDeploymentsClient()
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var deployments []Deployment
	{
		for _, deploymentName := range getDeploymentNames() {
			deploymentExtended, err := deploymentClient.Get(resourceGroupName, deploymentName)
			if err != nil {
				if client.ResponseWasNotFound(deploymentExtended.Response) {
					// Fall through.
					continue
				}

				return nil, microerror.Mask(err)
			}

			deployment := Deployment{
				Name:                   *deploymentExtended.Name,
				Parameters:             *deploymentExtended.Properties.Parameters,
				ResourceGroup:          resourceGroupName,
				TemplateURI:            *deploymentExtended.Properties.TemplateLink.URI,
				TemplateContentVersion: *deploymentExtended.Properties.TemplateLink.ContentVersion,
			}
			deployments = append(deployments, deployment)
		}
	}

	return deployments, nil
}

// GetDesiredState returns the desired deployments for this cluster.
func (r *Resource) GetDesiredState(ctx context.Context, obj interface{}) (interface{}, error) {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	mainDeployment, err := r.newMainDeployment(customObject)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	deployments := []Deployment{
		mainDeployment,
	}
	return deployments, nil
}

// NewUpdatePatch returns the deployments that should be created for this
// cluster.
func (r *Resource) NewUpdatePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	patch := framework.NewPatch()

	deploymentsToCreate, err := r.newCreateChange(ctx, obj, currentState, desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	patch.SetCreateChange(deploymentsToCreate)
	return patch, nil
}

// NewDeletePatch returns an empty patch. Deployments are deleted together with
// the resource group.
func (r *Resource) NewDeletePatch(ctx context.Context, obj, currentState, desiredState interface{}) (*framework.Patch, error) {
	p := framework.NewPatch()
	return p, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

// ApplyCreateChange creates the deployments via the Azure API.
func (r *Resource) ApplyCreateChange(ctx context.Context, obj, createState interface{}) error {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsToCreate, err := toDeployments(createState)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "debug", "creating Azure deployments")

	if len(deploymentsToCreate) != 0 {
		resourceGroupName := key.ClusterID(customObject)
		deploymentsClient, err := r.getDeploymentsClient()
		if err != nil {
			return microerror.Maskf(err, "creating Azure deployments")
		}

		for _, deploy := range deploymentsToCreate {
			r.logger.LogCtx(ctx, "debug", fmt.Sprintf("creating Azure deployments: creating %#q", deploy.Name))

			deployment := azureresource.Deployment{
				Properties: &azureresource.DeploymentProperties{
					Mode:       azureresource.Complete,
					Parameters: &deploy.Parameters,
					TemplateLink: &azureresource.TemplateLink{
						URI:            to.StringPtr(deploy.TemplateURI),
						ContentVersion: to.StringPtr(deploy.TemplateContentVersion),
					},
				},
			}

			_, errchan := deploymentsClient.CreateOrUpdate(resourceGroupName, deploy.Name, deployment, nil)
			select {
			case err := <-errchan:
				if err != nil {
					return microerror.Maskf(err, "creating Azure deployments: creating %#q", deploy.Name)
				}
			case <-time.After(createTimeout):
				return microerror.Maskf(timeoutError, "creating Azure deployments: creating %#q", deploy.Name)
			}

			r.logger.LogCtx(ctx, "debug", fmt.Sprintf("creating Azure deployments: creating %#q: created", deploy.Name))
		}

		r.logger.LogCtx(ctx, "debug", "creating Azure deployments: created")
	} else {
		r.logger.LogCtx(ctx, "debug", "creating Azure deployments: already created")
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

// Underlying returns the underlying resource.
func (r *Resource) Underlying() framework.Resource {
	return r
}

func (r *Resource) getDeploymentsClient() (*azureresource.DeploymentsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.DeploymentsClient, nil
}

func (r *Resource) newCreateChange(ctx context.Context, obj, currentState, desiredState interface{}) ([]Deployment, error) {
	currentDeployments, err := toDeployments(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredDeployments, err := toDeployments(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var deploymentsToCreate []Deployment

	for _, desiredDeployment := range desiredDeployments {
		if !existsDeploymentByName(currentDeployments, desiredDeployment.Name) {
			deploymentsToCreate = append(deploymentsToCreate, desiredDeployment)
		}
	}

	return deploymentsToCreate, nil
}

func existsDeploymentByName(list []Deployment, name string) bool {
	for _, d := range list {
		if d.Name == name {
			return true
		}
	}

	return false
}

func getDeploymentByName(list []Deployment, name string) (Deployment, error) {
	for _, d := range list {
		if d.Name == name {
			return d, nil
		}
	}

	return Deployment{}, microerror.Maskf(notFoundError, name)
}

func toCustomObject(v interface{}) (azuretpr.CustomObject, error) {
	if v == nil {
		return azuretpr.CustomObject{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &azuretpr.CustomObject{}, v)
	}

	customObjectPointer, ok := v.(*azuretpr.CustomObject)
	if !ok {
		return azuretpr.CustomObject{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &azuretpr.CustomObject{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func toDeployments(v interface{}) ([]Deployment, error) {
	if v == nil {
		return []Deployment{}, nil
	}

	deployments, ok := v.([]Deployment)
	if !ok {
		return []Deployment{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", []Deployment{}, v)
	}

	return deployments, nil
}
