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
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
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
	azureMachinePool, err := key.ToAzureMachinePool(obj)
	if err != nil {
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Ensuring new AzureMachinePool %s/%s has been migrated", azureMachinePool.Namespace, azureMachinePool.Name)

	if !areReferencesUpdated(azureMachinePool) {
		// Migration from old to new MachinePool is not completed. Cancel
		// remaining reconciliation.
		r.logger.Debugf(ctx, "AzureMachinePool %s/%s CR references have not been updated, assuming migration has not been completed, canceling reconciliation", azureMachinePool.Namespace, azureMachinePool.Name)
		reconciliationcanceledcontext.SetCanceled(ctx)
		return nil
	}

	// Now the new AzureMachinePool has been created with Spec copied from old
	// AzureMachinePool, so here we want to:
	// 1. Clone Status from old AzureMachinePool
	// 2. Delete old AzureMachinePool

	// Get old AzureMachinePool
	namespacedName := types.NamespacedName{
		Namespace: azureMachinePool.Namespace,
		Name:      azureMachinePool.Name,
	}
	oldAzureMachinePool := &oldcapzexpv1alpha3.AzureMachinePool{}
	err = r.ctrlClient.Get(ctx, namespacedName, oldAzureMachinePool)
	if apierrors.IsNotFound(err) {
		// Old AzureMachinePool not found, so nothing to do here.
		r.logger.Debugf(ctx, "Old AzureMachinePool not found, assuming AzureMachinePool %s/%s has been migrated", azureMachinePool.Namespace, azureMachinePool.Name)
		return nil
	} else if err != nil {
		// Migration from old to new AzureMachinePool is not completed
		// because we still didn't update the status in the new AzureMachinePool,
		// so we cancel remaining reconciliation.
		r.logger.Debugf(ctx, "Failed to fetch old AzureMachinePool %s/%s, assuming migration has not been completed, canceling reconciliation", azureMachinePool.Namespace, azureMachinePool.Name)
		reconciliationcanceledcontext.SetCanceled(ctx)
		return microerror.Mask(err)
	}

	// 1. Update status in new AzureMachinePool
	if isStatusEmpty(azureMachinePool) {
		r.logger.Debugf(ctx, "Updating new AzureMachinePool %s/%s status", azureMachinePool.Namespace, azureMachinePool.Name)
		err = cloneObject(&oldAzureMachinePool.Status, &azureMachinePool.Status)
		if err != nil {
			// Migration from old to new AzureMachinePool is not completed
			// because we still didn't update the status in the new AzureMachinePool,
			// so we cancel remaining reconciliation.
			r.logger.Debugf(ctx, "Failed to copy AzureMachinePool %s/%s status from old AzureMachinePool, assuming migration has not been completed, canceling reconciliation", azureMachinePool.Namespace, azureMachinePool.Name)
			reconciliationcanceledcontext.SetCanceled(ctx)
			return microerror.Mask(err)
		}

		// Update new AzureMachinePool
		err = r.ctrlClient.Status().Update(ctx, &azureMachinePool)
		if err != nil {
			// Migration from old to new AzureMachinePool is not completed
			// because we still didn't update the status in the new AzureMachinePool,
			// so we cancel remaining reconciliation.
			r.logger.Debugf(ctx, "Failed to update AzureMachinePool %s/%s status, assuming migration has not been completed, canceling reconciliation", azureMachinePool.Namespace, azureMachinePool.Name)
			reconciliationcanceledcontext.SetCanceled(ctx)
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Updated new AzureMachinePool %s/%s status", azureMachinePool.Namespace, azureMachinePool.Name)
	}

	// 2. Finally, delete the old AzureMachinePool
	err = r.deleteOldAzureMachinePool(ctx, oldAzureMachinePool)
	if err != nil {
		// AzureMachinePool status is updated, so we don't cancel the
		// reconciliation, other handlers can continue working. It will try to
		// delete the old AzureMachinePool again in the next reconciliation loop.
		return microerror.Mask(err)
	}

	r.logger.Debugf(ctx, "Ensured AzureMachinePool %s/%s has been migrated", azureMachinePool.Namespace, azureMachinePool.Name)
	return nil
}

func areReferencesUpdated(azureMachinePool capzexp.AzureMachinePool) bool {
	// check MachinePool owner reference
	for _, ref := range azureMachinePool.ObjectMeta.OwnerReferences {
		if ref.Kind == "MachinePool" && ref.APIVersion == oldcapiexpv1alpha3.GroupVersion.String() {
			return false
		}
	}

	return true
}

func isStatusEmpty(azureMachinePool capzexp.AzureMachinePool) bool {
	return reflect.DeepEqual(azureMachinePool.Status, capzexp.AzureMachinePoolStatus{})
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
func (r *Resource) EnsureDeleted(_ context.Context, _ interface{}) error {
	return nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
