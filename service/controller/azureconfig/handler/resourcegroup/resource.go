package resourcegroup

import (
	"context"
	"fmt"
	"net/http"

	azureresource "github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/apiextensions/v5/pkg/apis/provider/v1alpha1"
	azureconditions "github.com/giantswarm/apiextensions/v5/pkg/conditions/azure"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/finalizerskeptcontext"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/reconciliationcanceledcontext"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/helpers"
	"github.com/giantswarm/azure-operator/v5/pkg/project"
	"github.com/giantswarm/azure-operator/v5/service/controller/controllercontext"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
	"github.com/giantswarm/azure-operator/v5/service/controller/setting"
)

const (
	// Name is the identifier of the resource.
	Name = "resourcegroup"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger

	Azure            setting.Azure
	InstallationName string
}

// Resource manages Azure resource groups.
type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger

	azure            setting.Azure
	installationName string
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
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

		azure:      config.Azure,
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
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

	conditionsUpdateError := r.checkAndUpdateResourceGroupReadyCondition(ctx, cr, groupsClient)
	if conditionsUpdateError != nil {
		r.logger.LogCtx(ctx, "level", "warning", "message", "error while updating AzureCluster ResourceGroupReady condition", "error", conditionsUpdateError.Error())
	}

	r.logger.Debugf(ctx, "ensuring resource group is created")

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

	r.logger.Debugf(ctx, "ensured resource group is created")

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

	r.logger.Debugf(ctx, "ensuring resource group deletion")

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

			r.logger.Debugf(ctx, "resource group deletion in progress")
			finalizerskeptcontext.SetKept(ctx)
			reconciliationcanceledcontext.SetCanceled(ctx)
			r.logger.Debugf(ctx, "canceling reconciliation")

			return nil
		}
	}

	r.logger.Debugf(ctx, "ensured resource group deletion")

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

func (r *Resource) checkAndUpdateResourceGroupReadyCondition(ctx context.Context, azureConfig v1alpha1.AzureConfig, groupsClient *azureresource.GroupsClient) error {
	logger := r.logger.With("level", "debug", "type", "AzureCluster", "message", "setting Status.Condition", "conditionType", azureconditions.ResourceGroupReadyCondition)

	organizationNamespace := key.OrganizationNamespace(&azureConfig)
	azureCluster, err := helpers.GetAzureClusterByName(ctx, r.ctrlClient, organizationNamespace, azureConfig.Name)
	if err != nil {
		return microerror.Mask(err)
	}

	isResourceGroupCreated := conditions.IsTrue(azureCluster, azureconditions.ResourceGroupReadyCondition)

	// Resource group is already created
	if isResourceGroupCreated {
		return nil
	}

	group, err := groupsClient.Get(ctx, azureConfig.Name)
	const genericErrorMessage = "Failed to get resource group from Azure API"
	var conditionReason string
	var conditionSeverity capi.ConditionSeverity

	if IsNotFound(err) {
		// resource group is not found, which means that the cluster is being created
		err = nil

		// let's set AzureCluster condition "ResourceGroupReady" to False, with reason
		// ResourceGroupNotFound, to signal that the resource group is not created yet
		conditionReason = "ResourceGroupNotFound"
		conditionSeverity = capi.ConditionSeverityWarning
		conditions.MarkFalse(
			azureCluster,
			azureconditions.ResourceGroupReadyCondition,
			conditionReason,
			conditionSeverity,
			"Resource group is not found")
		logger.LogCtx(ctx, "conditionStatus", false, "conditionReason", conditionReason, "conditionSeverity", conditionSeverity)
	} else if err != nil {
		conditionReason = "AzureAPIResourceGroupGetError"
		conditionSeverity = capi.ConditionSeverityWarning
		conditions.MarkFalse(
			azureCluster,
			azureconditions.ResourceGroupReadyCondition,
			conditionReason,
			conditionSeverity,
			fmt.Sprintf("%s, %s", genericErrorMessage, group.Status))
		logger.LogCtx(ctx, "conditionStatus", false, "conditionReason", conditionReason, "conditionSeverity", conditionSeverity)
	} else if group.StatusCode == http.StatusOK {
		conditions.MarkTrue(azureCluster, azureconditions.ResourceGroupReadyCondition)
		logger.LogCtx(ctx, "conditionStatus", true)
	} else {
		conditionReason = "AzureAPIResourceGroupGetError"
		conditionSeverity = capi.ConditionSeverityWarning
		conditions.MarkFalse(
			azureCluster,
			azureconditions.ResourceGroupReadyCondition,
			conditionReason,
			conditionSeverity,
			fmt.Sprintf("%s, %s", genericErrorMessage, group.Status))
		logger.LogCtx(ctx, "conditionStatus", false, "conditionReason", conditionReason, "conditionSeverity", conditionSeverity)
	}

	ctrlClientError := r.ctrlClient.Status().Update(ctx, azureCluster)
	// Prioritize initial Azure API error over controller client update error
	if err != nil {
		return microerror.Mask(err)
	} else if ctrlClientError != nil {
		return microerror.Mask(ctrlClientError)
	}

	r.logger.LogCtx(ctx, "level", "debug", "type", "AzureCluster", "message", "set Status.Condition", "conditionType", azureconditions.ResourceGroupReadyCondition)
	return nil
}
