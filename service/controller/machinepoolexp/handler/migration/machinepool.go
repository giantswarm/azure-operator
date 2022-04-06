package migration

import (
	"context"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capiexp/v1alpha3"
	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexpv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
)

const (
	operatorkitMachinePoolExpFinalizer = "operatorkit.giantswarm.io/azure-operator-machine-pool-exp-controller"
)

func (r *Resource) newMachinePoolExists(ctx context.Context, namespacedName types.NamespacedName) (bool, error) {
	newMachinePool := capiexp.MachinePool{}
	err := r.client.Get(ctx, namespacedName, &newMachinePool)
	if apierrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func (r *Resource) ensureNewMachinePoolCreated(ctx context.Context, oldMachinePoolV1alpha3 oldcapiexpv1alpha3.MachinePool) error {
	r.logger.Debugf(ctx, "Ensuring new MachinePool %s/%s has been created", oldMachinePoolV1alpha3.Namespace, oldMachinePoolV1alpha3.Name)
	namespacedName := types.NamespacedName{
		Namespace: oldMachinePoolV1alpha3.Namespace,
		Name:      oldMachinePoolV1alpha3.Name,
	}

	// First let's check if new MachinePool has already been created.
	exists, err := r.newMachinePoolExists(ctx, namespacedName)
	if err != nil {
		return microerror.Mask(err)
	}
	if exists {
		r.logger.Debugf(ctx, "New MachinePool %s/%s already exists", oldMachinePoolV1alpha3.Namespace, oldMachinePoolV1alpha3.Name)
		return nil
	}

	// Let's create a new non-exp v1alpha3 MachinePool where we will clone
	// Metadata from the old exp v1alpha3 MachinePool.
	newMachinePoolV1Alpha3 := capiexpv1alpha3.MachinePool{
		ObjectMeta: cloneObjectMeta(oldMachinePoolV1alpha3.ObjectMeta),
	}

	// Remove old exp finalizer from new MachinePool, otherwise new MachinePool
	// will get stuck while deleting.
	newMachinePoolV1Alpha3.ObjectMeta = removeFinalizer(newMachinePoolV1Alpha3.ObjectMeta, operatorkitMachinePoolExpFinalizer)

	// Now let's convert old exp v1alpha3 MachinePool.Spec to new non-exp
	// v1alpha3 MachinePool.Spec.
	err = convertSpec(&oldMachinePoolV1alpha3.Spec, &newMachinePoolV1Alpha3.Spec)
	if err != nil {
		return microerror.Mask(err)
	}

	// Finally, we convert new MachinePool from v1alpha3 to v1beta1. We call
	// conversion manually in order not to depend on conversion webhook.
	newMachinePool := capiexp.MachinePool{}
	err = newMachinePoolV1Alpha3.ConvertTo(&newMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	err = r.client.Create(ctx, &newMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Ensured new MachinePool %s/%s has been created", oldMachinePoolV1alpha3.Namespace, oldMachinePoolV1alpha3.Name)

	return nil
}

func (r *Resource) ensureNewMachinePoolReferencesUpdated(ctx context.Context, namespacedName types.NamespacedName) error {
	r.logger.Debugf(ctx, "Ensuring new MachinePool %s/%s references have been updated", namespacedName.Namespace, namespacedName.Name)
	newMachinePool := capiexp.MachinePool{}
	err := r.client.Get(ctx, namespacedName, &newMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	if newMachinePool.Spec.Template.Spec.InfrastructureRef.APIVersion == oldcapzexpv1alpha3.GroupVersion.String() {
		newMachinePool.Spec.Template.Spec.InfrastructureRef.APIVersion = capzexp.GroupVersion.String()
	}

	for i, ownerRef := range newMachinePool.ObjectMeta.OwnerReferences {
		if ownerRef.APIVersion == capiv1alpha3.GroupVersion.String() {
			newMachinePool.ObjectMeta.OwnerReferences[i].APIVersion = capi.GroupVersion.String()
		}
	}

	err = r.client.Update(ctx, &newMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Ensured new MachinePool %s/%s references have been updated", namespacedName.Namespace, namespacedName.Name)

	return nil
}
