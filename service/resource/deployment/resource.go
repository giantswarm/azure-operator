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

	createTimeout  = 5 * time.Minute
	deploymentMode = "Incremental"
	masterBranch   = "master"
)

type Config struct {
	// Dependencies.

	AzureConfig *client.AzureConfig
	CertWatcher certificatetpr.Searcher
	CloudConfig *cloudconfig.CloudConfig
	Logger      micrologger.Logger

	// Settings.

	// URIVersion is used when creating template links for ARM templates.
	// Defaults to master for deploying templates hosted on GitHub.
	URIVersion string
}

// DefaultConfig provides a default configuration to create a new resource by
// best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		AzureConfig: nil,
		CertWatcher: nil,
		CloudConfig: nil,
		Logger:      nil,

		// Settings.
		URIVersion: masterBranch,
	}
}

type Resource struct {
	// Dependencies.

	azureConfig *client.AzureConfig
	certWatcher certificatetpr.Searcher
	cloudConfig *cloudconfig.CloudConfig
	logger      micrologger.Logger

	// Settings.

	uriVersion string
}

// New creates a new configured deploy resource.
func New(config Config) (*Resource, error) {
	// Dependencies.
	if config.AzureConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig must not be empty.")
	}
	if config.CertWatcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.CertWatcher must not be empty")
	}
	if config.CloudConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.CloudConfig must not be empty.")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty.")
	}

	newService := &Resource{
		azureConfig: config.AzureConfig,
		certWatcher: config.CertWatcher,
		cloudConfig: config.CloudConfig,
		logger: config.Logger.With(
			"resource", Name,
		),
		uriVersion: config.URIVersion,
	}

	return newService, nil
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
				Name:            *deploymentExtended.Name,
				Parameters:      *deploymentExtended.Properties.Parameters,
				ResourceGroup:   resourceGroupName,
				TemplateURI:     *deploymentExtended.Properties.TemplateLink.URI,
				TemplateVersion: *deploymentExtended.Properties.TemplateLink.ContentVersion,
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

// GetCreateState returns the deployments that should be created for this
// cluster.
func (r *Resource) GetCreateState(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
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

// GetDeleteState returns an empty deployments collection. Deployments and the
// resources they manage are deleted when the Resource Group is deleted.
func (r *Resource) GetDeleteState(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, error) {
	return []Deployment{}, nil
}

// GetUpdateState is not yet implemented.
func (r *Resource) GetUpdateState(ctx context.Context, obj, currentState, desiredState interface{}) (interface{}, interface{}, interface{}, error) {
	return []Deployment{}, []Deployment{}, []Deployment{}, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

// ProcessCreateState creates the deployments via the Azure API.
func (r *Resource) ProcessCreateState(ctx context.Context, obj, createState interface{}) error {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsToCreate, err := toDeployments(createState)
	if err != nil {
		return microerror.Mask(err)
	}

	if len(deploymentsToCreate) != 0 {
		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "creating deployments in the Azure API")

		resourceGroupName := key.ClusterID(customObject)
		deploymentsClient, err := r.getDeploymentsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		for _, deploy := range deploymentsToCreate {
			r.logger.Log("cluster", key.ClusterID(customObject), "debug", fmt.Sprintf("creating deployment %s", deploy.Name))

			deployment := azureresource.Deployment{
				Properties: &azureresource.DeploymentProperties{
					Mode:       azureresource.Complete,
					Parameters: &deploy.Parameters,
					TemplateLink: &azureresource.TemplateLink{
						URI:            to.StringPtr(deploy.TemplateURI),
						ContentVersion: to.StringPtr(deploy.TemplateVersion),
					},
				},
			}

			_, errchan := deploymentsClient.CreateOrUpdate(resourceGroupName, deploy.Name, deployment, nil)
			select {
			case err := <-errchan:
				if err != nil {
					return microerror.Mask(err)
				}
			case <-time.After(createTimeout):
				return microerror.Mask(createTimeoutError)
			}

			r.logger.Log("cluster", key.ClusterID(customObject), "debug", fmt.Sprintf("created deployment %s", deploy.Name))
		}

		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "created the deployments in the Azure API")
	}

	return nil
}

// ProcessDeleteState returns nil because deployments are not deleted.
func (r *Resource) ProcessDeleteState(ctx context.Context, obj, deleteState interface{}) error {
	return nil
}

// ProcessUpdateState is not yet implemented.
func (r *Resource) ProcessUpdateState(ctx context.Context, obj, updateState interface{}) error {
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
