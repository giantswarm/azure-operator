package machinepooldependents

import (
	"context"

	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/controller/context/finalizerskeptcontext"
	"k8s.io/apimachinery/pkg/api/errors"
	expcapzv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	expcapiv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v4/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "machinepooldependents"
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

// Resource ensures that there are no orphaned dependent AzureMachinePool CRs.
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

// EnsureCreated is a no-op.
func (r *Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	return nil
}

// EnsureDeleted ensures that all dependent CRs are deleted before finalizer
// from MachinePool is removed.
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	cr, err := key.ToMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	exists, err := r.infrastructureCRExists(ctx, cr)
	if err != nil {
		return microerror.Mask(err)
	}

	if exists {
		finalizerskeptcontext.SetKept(ctx)
		return nil
	}

	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

func (r *Resource) infrastructureCRExists(ctx context.Context, cr expcapiv1alpha3.MachinePool) (bool, error) {
	azureMachinePool := new(expcapzv1alpha3.AzureMachinePool)
	err := r.ctrlClient.Get(ctx, client.ObjectKey{Namespace: cr.Namespace, Name: cr.Spec.Template.Spec.InfrastructureRef.Name}, azureMachinePool)
	if errors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}
