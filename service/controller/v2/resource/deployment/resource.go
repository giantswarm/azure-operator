package deployment

import (
	"context"
	"fmt"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2018-02-01/resources"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
	"github.com/giantswarm/azure-operator/service/controller/setting"
	servicecontext "github.com/giantswarm/azure-operator/service/controller/v2/context"
	"github.com/giantswarm/azure-operator/service/controller/v2/key"
)

const (
	// Name is the identifier of the resource.
	Name = "deploymentv2"
)

const (
	mainDeploymentName = "cluster-main-template"
)

type Config struct {
	Logger micrologger.Logger

	Azure           setting.Azure
	HostAzureConfig client.AzureClientSetConfig
	// TemplateVersion is the ARM template version. Currently is the name
	// of the git branch in which the version is stored.
	TemplateVersion string
}

type Resource struct {
	logger micrologger.Logger

	azure           setting.Azure
	hostAzureConfig client.AzureClientSetConfig
	templateVersion string
}

func New(config Config) (*Resource, error) {
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}
	if err := config.HostAzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "config.HostAzureConfig.%s", err)
	}
	if config.TemplateVersion == "" {
		return nil, microerror.Maskf(invalidConfigError, "%T.TemplateURIVersion must not be empty", config)
	}

	r := &Resource{
		logger: config.Logger,

		azure:           config.Azure,
		hostAzureConfig: config.HostAzureConfig,
		templateVersion: config.TemplateVersion,
	}

	return r, nil
}

func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	customObject, err := key.ToCustomObject(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	deploymentsClient, err := r.getDeploymentsClient(ctx)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring deployment")

	var deployment azureresource.Deployment

	d, err := deploymentsClient.Get(ctx, key.ClusterID(customObject), mainDeploymentName)
	if IsNotFound(err) {
		params := map[string]interface{}{
			"initialProvisioning": "Yes",
		}
		deployment, err = r.newDeployment(ctx, customObject, params)
		if err != nil {
			return microerror.Mask(err)
		}
	} else if err != nil {
		return microerror.Mask(err)
	} else {
		s := *d.Properties.ProvisioningState
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("deployment is in state '%s'", s))

		if !key.IsFinalProvisioningState(s) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "canceling resource for custom object")

			return nil
		}

		params := map[string]interface{}{
			"initialProvisioning": "No",
		}
		deployment, err = r.newDeployment(ctx, customObject, params)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	_, err = deploymentsClient.CreateOrUpdate(ctx, key.ClusterID(customObject), mainDeploymentName, deployment)
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

func (r *Resource) getDeploymentsClient(ctx context.Context) (*azureresource.DeploymentsClient, error) {
	sc, err := servicecontext.FromContext(ctx)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return sc.AzureClientSet.DeploymentsClient, nil
}
