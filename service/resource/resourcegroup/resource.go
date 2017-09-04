package resourcegroup

import (
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
	Name = "resourcegroup"

	clusterIDTag  = "ClusterID"
	customerIDTag = "CustomerID"
	deleteTimeout = 5 * time.Minute
	managedBy     = "azure-operator"
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

// New creates a new configured resource group resource.
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

// GetCurrentState gets the resource group for this cluster from the Azure API.
func (r *Resource) GetCurrentState(obj interface{}) (interface{}, error) {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var group Group
	{
		groupsClient, err := r.getGroupsClient()
		if err != nil {
			return nil, microerror.Mask(err)
		}

		resourceGroup, err := groupsClient.Get(key.ClusterID(customObject))
		if err != nil {
			if client.ResponseWasNotFound(resourceGroup.Response) {
				// Fall through.
				return Group{}, nil
			}

			return nil, microerror.Mask(err)
		}
		group = Group{
			Name:     *resourceGroup.Name,
			Location: *resourceGroup.Location,
			Tags:     to.StringMap(*resourceGroup.Tags),
		}
	}

	return group, nil
}

// GetDesiredState returns the desired resource group for this cluster.
func (r *Resource) GetDesiredState(obj interface{}) (interface{}, error) {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	tags := map[string]string{
		clusterIDTag:  key.ClusterID(customObject),
		customerIDTag: key.ClusterCustomer(customObject),
	}
	resourceGroup := Group{
		Name:     key.ClusterID(customObject),
		Location: key.Location(customObject),
		Tags:     tags,
	}

	return resourceGroup, nil
}

// GetCreateState returns the resource group for this cluster if it should be
// created.
func (r *Resource) GetCreateState(obj, currentState, desiredState interface{}) (interface{}, error) {
	currentResourceGroup, err := toGroup(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredResourceGroup, err := toGroup(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var resourceGroupToCreate Group
	if currentResourceGroup.Name == "" {
		resourceGroupToCreate = desiredResourceGroup
	}

	return resourceGroupToCreate, nil
}

// GetDeleteState returns the resource group for this cluster if it should be
// deleted.
func (r *Resource) GetDeleteState(obj, currentState, desiredState interface{}) (interface{}, error) {
	currentResourceGroup, err := toGroup(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredResourceGroup, err := toGroup(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var resourceGroupToDelete Group
	if currentResourceGroup.Name != "" {
		resourceGroupToDelete = desiredResourceGroup
	}

	return resourceGroupToDelete, nil
}

// GetUpdateState returns an empty group for the create, delete and update
// states because resource groups are not updated.
func (r *Resource) GetUpdateState(obj, currentState, desiredState interface{}) (interface{}, interface{}, interface{}, error) {
	return Group{}, Group{}, Group{}, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

// ProcessCreateState creates the resource group via the Azure API.
func (r *Resource) ProcessCreateState(obj, createState interface{}) error {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	resourceGroupToCreate, err := toGroup(createState)
	if err != nil {
		return microerror.Mask(err)
	}

	if resourceGroupToCreate.Name != "" {
		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "creating the resource group in the Azure API")

		groupClient, err := r.getGroupsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		resourceGroup := azureresource.Group{
			Name:      to.StringPtr(resourceGroupToCreate.Name),
			Location:  to.StringPtr(resourceGroupToCreate.Location),
			ManagedBy: to.StringPtr(managedBy),
			Tags:      to.StringMapPtr(resourceGroupToCreate.Tags),
		}
		_, err = groupClient.CreateOrUpdate(resourceGroupToCreate.Name, resourceGroup)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "created the resource group in the Azure API")
	}

	return nil
}

// ProcessDeleteState deletes the resource group via the Azure API.
func (r *Resource) ProcessDeleteState(obj, deleteState interface{}) error {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	resourceGroupToDelete, err := toGroup(deleteState)
	if err != nil {
		return microerror.Mask(err)
	}

	if resourceGroupToDelete.Name != "" {
		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "deleting the resource group in the Azure API")

		groupsClient, err := r.getGroupsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		// Delete the resource group which also deletes all resources it
		// contains. We wait for the error channel while the deletion happens.
		_, errchan := groupsClient.Delete(resourceGroupToDelete.Name, nil)
		select {
		case err := <-errchan:
			if err != nil {
				return microerror.Mask(err)
			}
		case <-time.After(deleteTimeout):
			return microerror.Mask(deleteTimeoutError)
		}

		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "deleted the resource group in the Azure API")
	}

	return nil
}

// ProcessUpdateState returns nil because resource groups are not updated.
func (r *Resource) ProcessUpdateState(obj, updateState interface{}) error {
	return nil
}

// Underlying returns the underlying resource.
func (r *Resource) Underlying() framework.Resource {
	return r
}

func (r *Resource) getGroupsClient() (*azureresource.GroupsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.GroupsClient, nil
}

func toCustomObject(v interface{}) (azuretpr.CustomObject, error) {
	customObjectPointer, ok := v.(*azuretpr.CustomObject)
	if !ok {
		return azuretpr.CustomObject{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &azuretpr.CustomObject{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func toGroup(v interface{}) (Group, error) {
	if v == nil {
		return Group{}, nil
	}

	resourceGroup, ok := v.(Group)
	if !ok {
		return Group{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", Group{}, v)
	}

	return resourceGroup, nil
}
