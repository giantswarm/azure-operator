package resourcegroup

import (
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

	var resourceGroup azureresource.Group
	{
		groupsClient, err := r.getGroupsClient()
		if err != nil {
			return nil, microerror.Mask(err)
		}

		resourceGroup, err = groupsClient.Get(key.ClusterID(customObject))
		if err != nil {
			if client.ResponseWasNotFound(resourceGroup.Response) {
				// Fall through.
				return nil, nil
			}

			return nil, microerror.Mask(err)
		}
	}

	return &resourceGroup, nil
}

// GetDesiredState returns the desired resource group for this cluster.
func (r *Resource) GetDesiredState(obj interface{}) (interface{}, error) {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Log("cluster", key.ClusterID(customObject), "debug", "computing the new resource group")

	tags := map[string]string{
		clusterIDTag:  key.ClusterID(customObject),
		customerIDTag: key.ClusterCustomer(customObject),
	}
	resourceGroup := &azureresource.Group{
		Name:      to.StringPtr(key.ClusterID(customObject)),
		Location:  to.StringPtr(key.Location(customObject)),
		ManagedBy: to.StringPtr(managedBy),
		Tags:      to.StringMapPtr(tags),
	}

	r.logger.Log("cluster", key.ClusterID(customObject), "debug", "computed the new resource group")

	return resourceGroup, nil
}

// GetCreateState returns the resource group for this cluster if it should be
// created.
func (r *Resource) GetCreateState(obj, currentState, desiredState interface{}) (interface{}, error) {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	currentResourceGroup, err := toResourceGroup(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredResourceGroup, err := toResourceGroup(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Log("cluster", key.ClusterID(customObject), "debug", "checking if the resource group has to be created")

	var resourceGroupToCreate *azureresource.Group
	if currentResourceGroup == nil {
		resourceGroupToCreate = desiredResourceGroup
	}

	r.logger.Log("cluster", key.ClusterID(customObject), "debug", "checked if the resource group has to be created")

	return resourceGroupToCreate, nil
}

// GetDeleteState returns the resource group for this cluster if it should be
// deleted.
func (r *Resource) GetDeleteState(obj, currentState, desiredState interface{}) (interface{}, error) {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	currentResourceGroup, err := toResourceGroup(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredResourceGroup, err := toResourceGroup(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	r.logger.Log("cluster", key.ClusterID(customObject), "debug", "checking if the resource group has to be deleted")

	var resourceGroupToDelete *azureresource.Group
	if currentResourceGroup != nil {
		resourceGroupToDelete = desiredResourceGroup
	}

	r.logger.Log("cluster", key.ClusterID(customObject), "debug", "checked if the resource group has to be deleted")

	return resourceGroupToDelete, nil
}

// GetUpdateState returns nil for the create, delete and update states because
// resource groups are not updated.
func (r *Resource) GetUpdateState(obj, currentState, desiredState interface{}) (interface{}, interface{}, interface{}, error) {
	return nil, nil, nil, nil
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

	resourceGroupToCreate, err := toResourceGroup(createState)
	if err != nil {
		return microerror.Mask(err)
	}

	if resourceGroupToCreate != nil {
		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "creating the resource group in the Azure API")

		groupClient, err := r.getGroupsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		_, err = groupClient.CreateOrUpdate(*resourceGroupToCreate.Name, *resourceGroupToCreate)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "created the resource group in the Azure API")

	} else {
		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "the resource group already exists in the Azure API")
	}

	return nil
}

// ProcessDeleteState deletes the resource group via the Azure API.
func (r *Resource) ProcessDeleteState(obj, deleteState interface{}) error {
	customObject, err := toCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}
	resourceGroupToDelete, err := toResourceGroup(deleteState)
	if err != nil {
		return microerror.Mask(err)
	}

	if resourceGroupToDelete != nil {
		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "deleting the resource group in the Azure API")

		groupsClient, err := r.getGroupsClient()
		if err != nil {
			return microerror.Mask(err)
		}

		// Delete the resource group which also deletes all resources it
		// contains. We wait for the error channel while the deletion happens.
		_, errchan := groupsClient.Delete(*resourceGroupToDelete.Name, nil)
		err = <-errchan
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "deleted the resource group in the Azure API")

	} else {
		r.logger.Log("cluster", key.ClusterID(customObject), "debug", "the resource group does not exist in the Azure API")
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

func toResourceGroup(v interface{}) (*azureresource.Group, error) {
	if v == nil {
		return nil, nil
	}

	resourceGroup, ok := v.(*azureresource.Group)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &azureresource.Group{}, v)
	}

	return resourceGroup, nil
}
