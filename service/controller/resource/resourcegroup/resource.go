package resourcegroup

import (
	"context"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/finalizerskeptcontext"
	"github.com/giantswarm/operatorkit/v2/pkg/controller/context/reconciliationcanceledcontext"

	"github.com/giantswarm/azure-operator/v4/pkg/project"
	"github.com/giantswarm/azure-operator/v4/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
	"github.com/giantswarm/azure-operator/v4/service/controller/setting"
)

const (
	// Name is the identifier of the resource.
	Name = "resourcegroup"
)

type Config struct {
	Logger micrologger.Logger

	Azure            setting.Azure
	InstallationName string
}

// Resource manages Azure resource groups.
type Resource struct {
	logger micrologger.Logger

	azure            setting.Azure
	installationName string
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if config.InstallationName == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.InstallationName must not be empty", config)
	}

	r := &Resource{
		installationName: config.InstallationName,

		azure:  config.Azure,
		logger: config.Logger,
	}

	return r, nil
}

// EnsureCreated ensures the resource group is created via the Azure API.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	groupsClient, err := r.getGroupsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring resource group is created")

	resourceGroup := azureresource.Group{
		Name:      to.StringPtr(key.ClusterID(&cr)),
		Location:  to.StringPtr(r.azure.Location),
		ManagedBy: to.StringPtr(project.Name()),
		Tags:      key.ClusterTags(cr, r.installationName),
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
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	groupsClient, err := r.getGroupsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring resource group deletion")

	_, err = groupsClient.Get(ctx, key.ClusterID(&cr))
	if IsNotFound(err) {
		// fall through
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		res, err := groupsClient.Delete(ctx, key.ClusterID(&cr))
		if IsNotFound(err) {
			// fall through
		} else if err != nil {
			return microerror.Mask(err)
		} else {
			_, err = groupsClient.DeleteResponder(res.Response())
			if err != nil {
				return microerror.Mask(err)
			}

			r.logger.LogCtx(ctx, "level", "debug", "message", "resource group deletion in progress")
			finalizerskeptcontext.SetKept(ctx)
			reconciliationcanceledcontext.SetCanceled(ctx)
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")

			return nil
		}
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured resource group deletion")

	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) getGroupsClient(ctx context.Context) (*azureresource.GroupsClient, error) {
	cc, err := controllercontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return cc.AzureClientSet.GroupsClient, nil
}
