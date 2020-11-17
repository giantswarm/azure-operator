package machinepoolupgrade

import (
	"context"
	"fmt"

	"github.com/giantswarm/apiextensions/v3/pkg/label"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/pkg/tenantcluster"
	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "machinepoolupgrade"
)

type Config struct {
	CtrlClient          client.Client
	Logger              micrologger.Logger
	TenantClientFactory tenantcluster.Factory
}

// Resource ensures that corresponding AzureMachinePool has same release label as reconciled MachinePool.
type Resource struct {
	ctrlClient          client.Client
	logger              micrologger.Logger
	tenantClientFactory tenantcluster.Factory
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if config.TenantClientFactory == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.TenantClientFactory must not be empty", config)
	}

	r := &Resource{
		ctrlClient:          config.CtrlClient,
		logger:              config.Logger,
		tenantClientFactory: config.TenantClientFactory,
	}

	return r, nil
}

// EnsureCreated ensures corresponding AzureMachinePool has same release label as reconciled MachinePool CR.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring release labels are set on respective AzureMachinePool")

	azureMachinePool := expcapzv1alpha3.AzureMachinePool{}
	err = r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: cr.Spec.Template.Spec.InfrastructureRef.Name}, &azureMachinePool)
	if apierrors.IsNotFound(err) {
		r.logger.LogCtx(ctx, "level", "debug", "message", fmt.Sprintf("AzureMachinePool %s/%s was not found for MachinePool %#q, skipping setting owner reference", cr.Namespace, cr.Spec.Template.Spec.InfrastructureRef.Name, cr.Name))
		r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	if azureMachinePool.Labels == nil {
		azureMachinePool.Labels = make(map[string]string)
	}

	if azureMachinePool.Labels[label.AzureOperatorVersion] != cr.Labels[label.AzureOperatorVersion] ||
		azureMachinePool.Labels[label.ReleaseVersion] != cr.Labels[label.ReleaseVersion] {

		azureMachinePool.Labels[label.AzureOperatorVersion] = cr.Labels[label.AzureOperatorVersion]
		azureMachinePool.Labels[label.ReleaseVersion] = cr.Labels[label.ReleaseVersion]

		err = r.ctrlClient.Update(ctx, &azureMachinePool)
		if apierrors.IsConflict(err) {
			r.logger.LogCtx(ctx, "level", "debug", "message", "conflict trying to save object in k8s API concurrently", "stack", microerror.JSON(microerror.Mask(err)))
			r.logger.LogCtx(ctx, "level", "debug", "message", "cancelling resource")
			return nil
		} else if err != nil {
			return microerror.Mask(err)
		}
	}
	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured release labels are set on respective AzureMachinePool")

	err = r.ensureLastDeployedReleaseVersion(ctx, &cr)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// EnsureDeleted is no-op.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
