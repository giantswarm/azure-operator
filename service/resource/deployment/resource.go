package deployment

import (
	"fmt"
	"time"

	azureresource "github.com/Azure/azure-sdk-for-go/arm/resources/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/giantswarm/azure-operator/client"
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
	// URIVersion is used when creating template links for ARM templates.
	// Defaults to master for deploying templates hosted on GitHub.
	URIVersion string

	// Dependencies.

	AzureConfig *client.AzureConfig
	Logger      micrologger.Logger
}

// DefaultConfig provides a default configuration to create a new resource by
// best effort.
func DefaultConfig() Config {
	return Config{
		URIVersion: masterBranch,

		// Dependencies.
		AzureConfig: nil,
		Logger:      nil,
	}
}

type Resource struct {
	uriVersion string

	// Dependencies.

	azureConfig *client.AzureConfig
	logger      micrologger.Logger
}

// New creates a new configured deploy resource.
func New(config Config) (*Resource, error) {
	// Dependencies.
	if config.AzureConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig must not be empty.")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty.")
	}

	newService := &Resource{
		azureConfig: config.AzureConfig,
		logger: config.Logger.With(
			"resource", Name,
		),
		uriVersion: config.URIVersion,
	}

	return newService, nil
}

// GetCurrentState gets the current deployments for this cluster via the
// Azure API.
func (r *Resource) GetCurrentState(obj interface{}) (interface{}, error) {
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
func (r *Resource) GetDesiredState(obj interface{}) (interface{}, error) {
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
func (r *Resource) GetCreateState(obj, currentState, desiredState interface{}) (interface{}, error) {
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
func (r *Resource) GetDeleteState(obj, currentState, desiredState interface{}) (interface{}, error) {
	return []Deployment{}, nil
}

// GetUpdateState is not yet implemented.
func (r *Resource) GetUpdateState(obj, currentState, desiredState interface{}) (interface{}, interface{}, interface{}, error) {
	return []Deployment{}, []Deployment{}, []Deployment{}, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

// ProcessCreateState creates the deployments via the Azure API.
func (r *Resource) ProcessCreateState(obj, createState interface{}) error {
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

			params := make(map[string]interface{}, len(deploy.Parameters))
			for key, val := range deploy.Parameters {
				params[key] = struct {
					Value interface{}
				}{
					Value: val,
				}
			}
			deployment := azureresource.Deployment{
				Properties: &azureresource.DeploymentProperties{
					Mode:       azureresource.Complete,
					Parameters: &params,
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
func (r *Resource) ProcessDeleteState(obj, deleteState interface{}) error {
	return nil
}

// ProcessUpdateState is not yet implemented.
func (r *Resource) ProcessUpdateState(obj, updateState interface{}) error {
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
