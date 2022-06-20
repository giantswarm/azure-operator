package migration

import (
	"context"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capiexp/v1alpha3"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capiexpv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"

	"github.com/giantswarm/azure-operator/v5/pkg/machinepoolmigration"
)

const (
	operatorkitMachinePoolExpFinalizer = "operatorkit.giantswarm.io/azure-operator-machine-pool-exp-controller"
)

func (r *Resource) newMachinePoolExists(ctx context.Context, namespacedName types.NamespacedName) (*capiexp.MachinePool, error) {
	newMachinePool := capiexp.MachinePool{}
	err := r.client.Get(ctx, namespacedName, &newMachinePool)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		return nil, microerror.Mask(err)
	}

	return &newMachinePool, nil
}

func (r *Resource) ensureNewMachinePoolCreated(ctx context.Context, oldMachinePoolV1alpha3 oldcapiexpv1alpha3.MachinePool) (*capiexp.MachinePool, error) {
	r.logger.Debugf(ctx, "Ensuring new MachinePool %s/%s has been created", oldMachinePoolV1alpha3.Namespace, oldMachinePoolV1alpha3.Name)
	namespacedName := types.NamespacedName{
		Namespace: oldMachinePoolV1alpha3.Namespace,
		Name:      oldMachinePoolV1alpha3.Name,
	}

	// First let's check if new MachinePool has already been created.
	mp, err := r.newMachinePoolExists(ctx, namespacedName)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	if mp != nil {
		r.logger.Debugf(ctx, "New MachinePool %s/%s already exists", oldMachinePoolV1alpha3.Namespace, oldMachinePoolV1alpha3.Name)
		return mp, nil
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
		return nil, microerror.Mask(err)
	}

	// Finally, we convert new MachinePool from v1alpha3 to v1beta1. We call
	// conversion manually in order not to depend on conversion webhook.
	newMachinePool := capiexp.MachinePool{}
	err = newMachinePoolV1Alpha3.ConvertTo(&newMachinePool)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	// Adjust ownerReferences.
	for i, ownerRef := range newMachinePool.ObjectMeta.OwnerReferences {
		if ownerRef.Kind == "Cluster" && ownerRef.APIVersion != machinepoolmigration.DesiredCAPIGroupVersion {
			newMachinePool.ObjectMeta.OwnerReferences[i].APIVersion = machinepoolmigration.DesiredCAPIGroupVersion
		}
	}

	err = r.client.Create(ctx, &newMachinePool)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Ensured new MachinePool %s/%s has been created", oldMachinePoolV1alpha3.Namespace, oldMachinePoolV1alpha3.Name)

	return &newMachinePool, nil
}

func (r *Resource) ensureNewMachinePoolReferencesUpdated(ctx context.Context, namespacedName types.NamespacedName) error {
	r.logger.Debugf(ctx, "Ensuring new MachinePool %s/%s references have been updated", namespacedName.Namespace, namespacedName.Name)
	newMachinePool := capiexp.MachinePool{}
	err := r.client.Get(ctx, namespacedName, &newMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	update := false
	if newMachinePool.Spec.Template.Spec.InfrastructureRef.Kind == "AzureMachinePool" &&
		newMachinePool.Spec.Template.Spec.InfrastructureRef.APIVersion != machinepoolmigration.DesiredCAPZGroupVersion {
		newMachinePool.Spec.Template.Spec.InfrastructureRef.APIVersion = machinepoolmigration.DesiredCAPZGroupVersion
		update = true
	}

	if update {
		err = r.client.Update(ctx, &newMachinePool)
		if err != nil {
			return microerror.Mask(err)
		}
		r.logger.Debugf(ctx, "Ensured new MachinePool %s/%s references have been updated", namespacedName.Namespace, namespacedName.Name)
	} else {
		r.logger.Debugf(ctx, "New MachinePool %s/%s references have been already updated", namespacedName.Namespace, namespacedName.Name)
	}

	return nil
}
