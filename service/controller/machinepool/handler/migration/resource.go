package migration

import (
	"context"
	"encoding/json"
	"reflect"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v5/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v5/service/controller/key"
)

const (
	// Name is the identifier of the resource.
	Name = "migration"
)

var (
	DesiredCAPIGroupVersion = capi.GroupVersion.String()
	DesiredCAPZGroupVersion = capz.GroupVersion.String()
)

type Config struct {
	CtrlClient client.Client
	Logger     micrologger.Logger
}

type Resource struct {
	ctrlClient client.Client
	logger     micrologger.Logger
}

func New(c Config) (*Resource, error) {
	if c.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", c)
	}
	if c.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", c)
	}

	r := &Resource{
		ctrlClient: c.CtrlClient,
		logger:     c.Logger,
	}

	return r, nil
}

func (r Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	machinePool, err := key.ToMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	if !areReferencesUpdated(machinePool) {
		// Migration from old to new MachinePool is not completed. Cancel
		// remaining reconciliation.
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}

	// Now the new MachinePool has been created with Spec copied from old
	// MachinePool, so here we want to:
	// 1. Clone Status from old MachinePool
	// 2. Delete old MachinePool

	// Get old MachinePool
	namespacedName := types.NamespacedName{
		Namespace: machinePool.Namespace,
		Name:      machinePool.Name,
	}
	oldMachinePool := &oldcapiexpv1alpha3.MachinePool{}
	err = r.ctrlClient.Get(ctx, namespacedName, oldMachinePool)
	if apierrors.IsNotFound(err) {
		// Old MachinePool not found, so nothing to do here.
		return nil
	} else if err != nil {
		// TODO: cancel remaining reconciliation
		return microerror.Mask(err)
	}

	// 1. Update status in new MachinePool
	if isStatusEmpty(machinePool) {
		err = cloneObject(&oldMachinePool.Status, &machinePool.Status)
		if err != nil {
			// TODO: cancel remaining reconciliation
			return microerror.Mask(err)
		}

		// Update new MachinePool
		err = r.ctrlClient.Status().Update(ctx, &machinePool)
		if err != nil {
			// TODO: cancel remaining reconciliation
			return microerror.Mask(err)
		}
	}

	// 2. Finally, delete the old MachinePool
	err = r.deleteOldMachinePool(ctx, oldMachinePool)
	if err != nil {
		// We don't cancel the reconciliation, so it will again try to delete
		// the old MachinePool.
		return microerror.Mask(err)
	}

	return nil
}

func areReferencesUpdated(machinePool capiexp.MachinePool) bool {
	// check Cluster owner reference
	for _, ref := range machinePool.ObjectMeta.OwnerReferences {
		if ref.Kind == "Cluster" && ref.APIVersion != DesiredCAPIGroupVersion {
			return false
		}
	}

	// check InfrastructureRef (AzureMachinePool) API version
	if machinePool.Spec.Template.Spec.InfrastructureRef.Kind == "AzureMachinePool" &&
		machinePool.Spec.Template.Spec.InfrastructureRef.APIVersion != DesiredCAPZGroupVersion {
		return false
	}

	return true
}

func isStatusEmpty(machinePool capiexp.MachinePool) bool {
	return reflect.DeepEqual(machinePool.Status, capiexp.MachinePoolStatus{})
}

func cloneObject(oldObject interface{}, newObject interface{}) error {
	oldJson, err := json.Marshal(oldObject)
	if err != nil {
		return err
	}

	err = json.Unmarshal(oldJson, newObject)
	if err != nil {
		return err
	}

	return nil
}

// EnsureDeleted noop
func (r *Resource) EnsureDeleted(ctx context.Context, obj interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
