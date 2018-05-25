package resourcegroup

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

const (
	// Name is the identifier of the resource.
	Name = "resourcegroupv2"

	managedBy = "azure-operator"
)

type Config struct {
	Logger micrologger.Logger

	AzureConfig      client.AzureConfig
	Azure            setting.Azure
	InstallationName string
}

// Resource manages Azure resource groups.
type Resource struct {
	logger micrologger.Logger

	azure            setting.Azure
	azureConfig      client.AzureConfig
	installationName string
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
	if config.InstallationName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstallationName must not be empty", config)
	}

	r := &Resource{
		installationName: config.InstallationName,

		azure:       config.Azure,
		azureConfig: config.AzureConfig,
		logger:      config.Logger,
	}

	return r, nil
}

// EnsureCreated ensures the resource group is created via the Azure API.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	groupsClient, err := r.getGroupsClient()
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring resource group is created")

	resourceGroup := azureresource.Group{
		Name:      to.StringPtr(key.ClusterID(customObject)),
		Location:  to.StringPtr(r.azure.Location),
		ManagedBy: to.StringPtr(managedBy),
		Tags:      key.ClusterTags(customObject, r.installationName),
	}
	_, err = groupsClient.CreateOrUpdate(ctx, *resourceGroup.Name, resourceGroup)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured resource group is created")

	return nil
}

// EnsureDeleted ensures the resource group is deleted via the Azure API.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	groupsClient, err := r.getGroupsClient()
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring resource group is deleted")

	f, err := groupsClient.Delete(ctx, key.ClusterID(customObject))
	if err != nil {
		return microerror.Mask(err)
	}
	err = f.WaitForCompletion(ctx, groupsClient.Client)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured resource group is deleted")

	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getGroupsClient() (*azureresource.GroupsClient, error) {
	azureClients, err := client.NewAzureClientSet(r.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return azureClients.GroupsClient, nil
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
