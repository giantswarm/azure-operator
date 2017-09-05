package deployment

import (
	azureresource "github.com/Azure/azure-sdk-for-go/arm/resources/resources"
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
)

type Config struct {
	// Dependencies.

	AzureConfig *client.AzureConfig
	Logger      micrologger.Logger
}

// DefaultConfig provides a default configuration to create a new resource by
// best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		AzureConfig: nil,
		Logger:      nil,
	}
}

type Resource struct {
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
	}

	return newService, nil
}

// GetCurrentState gets the current deployments for this cluster from the
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
				Name:          *deploymentExtended.Name,
				Parameters:    *deploymentExtended.Properties.Parameters,
				ResourceGroup: resourceGroupName,
				Template:      *deploymentExtended.Properties.Template,
			}
			deployments = append(deployments, deployment)
		}
	}

	return deployments, nil
}

// GetDesiredState is not yet implemented.
func (r *Resource) GetDesiredState(obj interface{}) (interface{}, error) {
	return []Deployment{}, nil
}

// GetCreateState is not yet implemented.
func (r *Resource) GetCreateState(obj, currentState, desiredState interface{}) (interface{}, error) {
	return []Deployment{}, nil
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

// ProcessCreateState is not yet implemented.
func (r *Resource) ProcessCreateState(obj, createState interface{}) error {
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
