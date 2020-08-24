package clusterid

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/reconciliationcanceledcontext"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "clusterid"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// Resource manages Azure resource groups.
type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		ctrlClient: config.CtrlClient,
		logger:     config.Logger,
	}

	return r, nil
}

// EnsureCreated ensures that reconciled AzureConfig CR has cluster ID label.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	cr, err := key.ToCustomResource(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensuring cluster id label is set")

	{
		// Refresh the CR object.
		nsName := types.NamespacedName{
			Name:      cr.Name,
			Namespace: cr.Namespace,
		}
		err := r.ctrlClient.Get(ctx, nsName, &cr)
		if err != nil {
			return microerror.Mask(err)
		}
	}

	v, exists := cr.Labels[label.Cluster]
	if !exists || v != cr.Spec.Cluster.ID {
		cr.Labels[label.Cluster] = cr.Spec.Cluster.ID

		r.logger.LogCtx(ctx, "level", "debug", "message", "updating CR labels with cluster id")

		err := r.ctrlClient.Update(ctx, &cr)
		if err != nil {
			return microerror.Mask(err)
		}

		r.logger.LogCtx(ctx, "level", "debug", "message", "updated CR labels with cluster id")
		r.logger.LogCtx(ctx, "level", "debug", "message", "canceling reconciliation")
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}

	r.logger.LogCtx(ctx, "level", "debug", "message", "ensured cluster id label is set")

	return nil
}

// EnsureDeleted is no-op.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	r.logger.LogCtx(ctx, "level", "debug", "message", "delete event on clusterid handler")
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
