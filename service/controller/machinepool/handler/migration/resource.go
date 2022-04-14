package migration

import (
	"context"
	"encoding/json"
	"reflect"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capiexp/v1alpha3"
	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
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

	r.logger.Debugf(ctx, "Ensuring new MachinePool %s/%s has been migrated", machinePool.Namespace, machinePool.Name)

	if !areReferencesUpdated(machinePool) {
		// Migration from old to new MachinePool is not completed. Cancel
		// remaining reconciliation.
		r.logger.Debugf(ctx, "MachinePool %s/%s CR references have not been updated, assuming migration has not been completed, canceling reconciliation", machinePool.Namespace, machinePool.Name)
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
		r.logger.Debugf(ctx, "Old MachinePool not found, assuming MachinePool %s/%s has been migrated", machinePool.Namespace, machinePool.Name)

		oldAzureMachinePoolExists, err := r.checkIfOldAzureMachinePoolExists(ctx, namespacedName)
		if err != nil {
			r.logger.Debugf(ctx, "Failed to fetch old AzureMachinePool %s/%s, assuming migration has not been completed, canceling reconciliation", namespacedName.Namespace, namespacedName.Name)
			reconciliationcanceledcontext.SetCanceled(ctx)
			return microerror.Mask(err)
		}
		migratedAzureMachinePoolExists, err := r.checkIfMigratedAzureMachinePoolExists(ctx, namespacedName)
		if err != nil {
			r.logger.Debugf(ctx, "Failed to fetch migrated AzureMachinePool %s/%s, assuming migration has not been completed, canceling reconciliation", namespacedName.Namespace, namespacedName.Name)
			reconciliationcanceledcontext.SetCanceled(ctx)
			return microerror.Mask(err)
		}

		if oldAzureMachinePoolExists && !migratedAzureMachinePoolExists {
			// We want to cancel the MachinePool reconciliation if AzureMachinePool
			// has not been migrated, because otherwise the machinepoolupgrade will
			// think that AzureMachinePool CR does not exist (as it's still not
			// migrated) and the upgrade will fail.
			r.logger.Debugf(ctx, "Found old AzureMachinePool %s/%s, but migrated AzureMachinePool not found, assuming migration has not been completed, canceling reconciliation", namespacedName.Namespace, namespacedName.Name)
			reconciliationcanceledcontext.SetCanceled(ctx)
		}
		return nil
	} else if err != nil {
		// Migration from old to new MachinePool is not completed because we
		// still didn't update the status in the new MachinePool, so we cancel
		// remaining reconciliation.
		r.logger.Debugf(ctx, "Failed to fetch old MachinePool %s/%s, assuming migration has not been completed, canceling reconciliation", machinePool.Namespace, machinePool.Name)
		reconciliationcanceledcontext.SetCanceled(ctx)
		return microerror.Mask(err)
	}

	// 1. Update status in new MachinePool
	if isStatusEmpty(machinePool) {
		r.logger.Debugf(ctx, "Updating new MachinePool %s/%s status", machinePool.Namespace, machinePool.Name)
		err = cloneObject(&oldMachinePool.Status, &machinePool.Status)
		if err != nil {
			// Migration from old to new MachinePool is not completed because
			// we still didn't update the status in the new MachinePool, so we
			// cancel remaining reconciliation.
			r.logger.Debugf(ctx, "Failed to copy MachinePool %s/%s status from old MachinePool, assuming migration has not been completed, canceling reconciliation", machinePool.Namespace, machinePool.Name)
			reconciliationcanceledcontext.SetCanceled(ctx)
			return microerror.Mask(err)
		}

		// Update new MachinePool
		err = r.ctrlClient.Status().Update(ctx, &machinePool)
		if err != nil {
			// Migration from old to new MachinePool is not completed because
			// we still didn't update the status in the new MachinePool, so we
			// cancel remaining reconciliation.
			r.logger.Debugf(ctx, "Failed to update MachinePool %s/%s status, assuming migration has not been completed, canceling reconciliation", machinePool.Namespace, machinePool.Name)
			reconciliationcanceledcontext.SetCanceled(ctx)
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Updated new MachinePool %s/%s status", machinePool.Namespace, machinePool.Name)
	}

	// 2. Finally, delete the old MachinePool
	err = r.deleteOldMachinePool(ctx, oldMachinePool)
	if err != nil {
		// MachinePool status is updated, so we don't cancel the reconciliation,
		// other handlers can continue working. It will try to delete the old
		// MachinePool again in the next reconciliation loop.
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Ensured MachinePool %s/%s has been migrated", machinePool.Namespace, machinePool.Name)

	oldAzureMachinePoolExists, err := r.checkIfOldAzureMachinePoolExists(ctx, namespacedName)
	if err != nil {
		r.logger.Debugf(ctx, "Failed to fetch old AzureMachinePool %s/%s, assuming migration has not been completed, canceling reconciliation", namespacedName.Namespace, namespacedName.Name)
		reconciliationcanceledcontext.SetCanceled(ctx)
		return microerror.Mask(err)
	}
	migratedAzureMachinePoolExists, err := r.checkIfMigratedAzureMachinePoolExists(ctx, namespacedName)
	if err != nil {
		r.logger.Debugf(ctx, "Failed to fetch migrated AzureMachinePool %s/%s, assuming migration has not been completed, canceling reconciliation", namespacedName.Namespace, namespacedName.Name)
		reconciliationcanceledcontext.SetCanceled(ctx)
		return microerror.Mask(err)
	}

	if oldAzureMachinePoolExists && !migratedAzureMachinePoolExists {
		// We want to cancel the MachinePool reconciliation if AzureMachinePool
		// has not been migrated, because otherwise the machinepoolupgrade will
		// think that AzureMachinePool CR does not exist (as it's still not
		// migrated) and the upgrade will fail.
		r.logger.Debugf(ctx, "Found old AzureMachinePool %s/%s, but migrated AzureMachinePool not found, assuming migration has not been completed, canceling reconciliation", namespacedName.Namespace, namespacedName.Name)
		reconciliationcanceledcontext.SetCanceled(ctx)
	}

	return nil
}

func (r *Resource) checkIfOldAzureMachinePoolExists(ctx context.Context, namespacedName types.NamespacedName) (bool, error) {
	r.logger.Debugf(ctx, "Checking if old exp AzureMachinePool %s/%s still exists", namespacedName.Namespace, namespacedName.Name)
	oldAzureMachinePool := &oldcapzexpv1alpha3.AzureMachinePool{}
	err := r.ctrlClient.Get(ctx, namespacedName, oldAzureMachinePool)
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "Old exp AzureMachinePool %s/%s not found", namespacedName.Namespace, namespacedName.Name)
		return false, nil
	} else if err != nil {
		r.logger.Debugf(ctx, "Failed to fetch old exp AzureMachinePool %s/%s", namespacedName.Namespace, namespacedName.Name)
		return false, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Old exp AzureMachinePool %s/%s found", namespacedName.Namespace, namespacedName.Name)
	return true, nil
}

func (r *Resource) checkIfMigratedAzureMachinePoolExists(ctx context.Context, namespacedName types.NamespacedName) (bool, error) {
	r.logger.Debugf(ctx, "Checking if migrated AzureMachinePool %s/%s has been created", namespacedName.Namespace, namespacedName.Name)
	azureMachinePool := &capzexp.AzureMachinePool{}
	err := r.ctrlClient.Get(ctx, namespacedName, azureMachinePool)
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "Migrated AzureMachinePool %s/%s not found", namespacedName.Namespace, namespacedName.Name)
		return false, nil
	} else if err != nil {
		r.logger.Debugf(ctx, "Failed to fetch migrated AzureMachinePool %s/%s", namespacedName.Namespace, namespacedName.Name)
		return false, microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Migrated AzureMachinePool %s/%s found", namespacedName.Namespace, namespacedName.Name)
	return true, nil
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
