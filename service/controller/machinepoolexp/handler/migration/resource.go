package migration

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "migration"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

type Resource struct {
	client client.Client
	logger micrologger.Logger
}

func New(config Config) (*Resource, error) {
	if config.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	r := &Resource{
		client: config.CtrlClient,
		logger: config.Logger,
	}

	return r, nil
}

// EnsureCreated creates non-experimental MachinePool CR if it doesn't already exist.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	// We do this even before upgrade? We must wait for new CRs in the upgrade handler!!!
	oldMachinePool, err := key.ToOldExpMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	namespacedName := types.NamespacedName{
		Namespace: oldMachinePool.Namespace,
		Name:      oldMachinePool.Name,
	}
	err = r.ensureNewMachinePoolCreated(ctx, oldMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureNewAzureMachinePoolCreated(ctx, namespacedName)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureNewMachinePoolReferencesUpdated(ctx, namespacedName)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.ensureNewAzureMachinePoolReferencesUpdated(ctx, namespacedName)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

// EnsureDeleted just logs that the old exp MachinePool has been deleted.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	oldMachinePool, err := key.ToOldExpMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Deleting old exp MachinePool CR %s/%s", oldMachinePool.Namespace, oldMachinePool.Name)
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
