package migration

import (
	"context"
	"encoding/json"
	"reflect"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/v7/pkg/controller/context/reconciliationcanceledcontext"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/azure-operator/v6/pkg/machinepoolmigration"
	"github.com/giantswarm/azure-operator/v6/service/controller/key"
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
	ctrlClient client.Client
	logger     micrologger.Logger
	checker    *machinepoolmigration.Checker
}

func New(c Config) (*Resource, error) {
	var err error
	if c.CtrlClient == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.CtrlClient must not be empty", c)
	}
	if c.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", c)
	}

	var checker *machinepoolmigration.Checker
	{
		config := machinepoolmigration.CheckerConfig{
			CtrlClient: c.CtrlClient,
			Logger:     c.Logger,
		}
		checker, err = machinepoolmigration.NewChecker(config)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	r := &Resource{
		ctrlClient: c.CtrlClient,
		logger:     c.Logger,
		checker:    checker,
	}

	return r, nil
}

func (r Resource) EnsureCreated(ctx context.Context, obj interface{}) error {
	machinePool, err := key.ToMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Ensuring new MachinePool %s/%s has been migrated", machinePool.Namespace, machinePool.Name)

	// Let's first check if MachinePool references are updated after migration.
	if !machinepoolmigration.AreMachinePoolReferencesUpdated(machinePool) {
		// Migration from old to new MachinePool is not completed. Cancel
		// remaining reconciliation.
		r.logger.Debugf(ctx, "MachinePool %s/%s CR references have not been updated, assuming migration has not been completed, canceling reconciliation", machinePool.Namespace, machinePool.Name)
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}

	// Now we check if old experimental MachinePool still exists.
	namespacedName := types.NamespacedName{
		Namespace: machinePool.Namespace,
		Name:      machinePool.Name,
	}
	oldMachinePool := &oldcapiexpv1alpha3.MachinePool{}
	err = r.ctrlClient.Get(ctx, namespacedName, oldMachinePool)
	if apierrors.IsNotFound(err) {
		r.logger.Debugf(ctx, "Old MachinePool not found, assuming MachinePool %s/%s has been migrated, but we also check AzureMachinePool", machinePool.Namespace, machinePool.Name)

		newAzureMachinePoolPendingCreation, err := r.checker.NewAzureMachinePoolPendingCreation(ctx, namespacedName)
		if err != nil {
			r.logger.Debugf(ctx, "Failed to check AzureMachinePool %s/%s migration, assuming migration has not been completed, canceling reconciliation", namespacedName.Namespace, namespacedName.Name)
			reconciliationcanceledcontext.SetCanceled(ctx)
			return microerror.Mask(err)
		}
		if newAzureMachinePoolPendingCreation {
			// New AzureMachinePool is not created, so we cancel remaining reconciliation,
			// otherwise machinepoolupgrade will fail if new AzureMachinePool is missing.
			r.logger.Debugf(ctx, "Still waiting for new AzureMachinePool %s/%s, assuming migration has not been completed, canceling reconciliation", namespacedName.Namespace, namespacedName.Name)
			reconciliationcanceledcontext.SetCanceled(ctx)
			return nil
		}

		r.logger.Debugf(ctx, "New AzureMachinePool %s/%s has been migrated, continuing MachinePool reconciliation", namespacedName.Namespace, namespacedName.Name)
		return nil
	} else if err != nil {
		// Migration from old to new MachinePool is not completed because we
		// still didn't update the status in the new MachinePool, so we cancel
		// remaining reconciliation.
		r.logger.Debugf(ctx, "Failed to fetch old MachinePool %s/%s, assuming migration has not been completed, canceling reconciliation", machinePool.Namespace, machinePool.Name)
		reconciliationcanceledcontext.SetCanceled(ctx)
		return microerror.Mask(err)
	}

	// Now the new MachinePool has been created with Spec copied from old
	// MachinePool, so here we want to:
	// 1. Clone Status from old MachinePool
	// 2. Delete old MachinePool

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

	// Before deleting the old MachinePool, we first check if new AzureMachinePool
	// has been fully migrated (new AzureMachinePool created, old AzureMachinePool
	// deleted).
	oldAzureMachinePoolExists, err := r.checker.CheckIfOldAzureMachinePoolExists(ctx, namespacedName)
	if err != nil {
		r.logger.Debugf(ctx, "Failed to check AzureMachinePool %s/%s migration, assuming migration has not been completed, canceling reconciliation", namespacedName.Namespace, namespacedName.Name)
		reconciliationcanceledcontext.SetCanceled(ctx)
		return microerror.Mask(err)
	}

	// 2. Finally, delete the old MachinePool when AzureMachinePool has been fully migrated
	if !oldAzureMachinePoolExists {
		err = r.deleteOldMachinePool(ctx, oldMachinePool)
		if err != nil {
			// MachinePool status is updated, so we don't cancel the reconciliation,
			// other handlers can continue working. It will try to delete the old
			// MachinePool again in the next reconciliation loop.
			return microerror.Mask(err)
		}
	}

	r.logger.Debugf(ctx, "Ensured MachinePool %s/%s has been migrated", machinePool.Namespace, machinePool.Name)
	return nil
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
