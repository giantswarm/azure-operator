package migration

import (
	"context"

	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v6/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	"sigs.k8s.io/cluster-api/exp/api/v1beta1"

	"github.com/giantswarm/azure-operator/v7/pkg/machinepoolmigration"
)

func (r *Resource) newAzureMachinePoolExists(ctx context.Context, namespacedName types.NamespacedName) (bool, error) {
	newAzureMachinePool := capzexp.AzureMachinePool{}
	err := r.client.Get(ctx, namespacedName, &newAzureMachinePool)
	if apierrors.IsNotFound(err) {
		return false, nil
	} else if err != nil {
		return false, microerror.Mask(err)
	}

	return true, nil
}

func (r *Resource) ensureNewAzureMachinePoolCreated(ctx context.Context, namespacedName types.NamespacedName, owner *v1beta1.MachinePool) error {
	r.logger.Debugf(ctx, "Ensuring new AzureMachinePool %s/%s has been created", namespacedName.Namespace, namespacedName.Name)
	// First let's check if new AzureMachinePool has already been created.
	exists, err := r.newAzureMachinePoolExists(ctx, namespacedName)
	if err != nil {
		return microerror.Mask(err)
	}
	if exists {
		r.logger.Debugf(ctx, "New AzureMachinePool %s/%s already exists", namespacedName.Namespace, namespacedName.Name)
		return nil
	}

	oldAzureMachinePoolV1alpha3 := oldcapzexpv1alpha3.AzureMachinePool{}
	err = r.client.Get(ctx, namespacedName, &oldAzureMachinePoolV1alpha3)
	if apierrors.IsNotFound(err) {
		// New AzureMachinePool does not exist, and the old one does not exist,
		// so there is just a MachinePool CR.
		r.logger.Debugf(ctx, "Old AzureMachinePool %s/%s not found, nothing to migrate", namespacedName.Namespace, namespacedName.Name)
		return nil
	} else if err != nil {
		return microerror.Mask(err)
	}

	// Let's create a new non-exp v1alpha3 AzureMachinePool where we will clone
	// Metadata from the old exp v1alpha3 AzureMachinePool.
	newAzureMachinePoolV1Alpha3 := capzexpv1alpha3.AzureMachinePool{
		ObjectMeta: cloneObjectMeta(oldAzureMachinePoolV1alpha3.ObjectMeta),
	}

	// Now let's convert old exp v1alpha3 AzureMachinePool.Spec to new non-exp
	// v1alpha3 AzureMachinePool.Spec.
	err = convertSpec(&oldAzureMachinePoolV1alpha3.Spec, &newAzureMachinePoolV1Alpha3.Spec)
	if err != nil {
		return microerror.Mask(err)
	}

	// Finally, we convert new AzureMachinePool from v1alpha3 to v1beta1. We call
	// conversion manually in order not to depend on conversion webhook.
	newAzureMachinePool := capzexp.AzureMachinePool{}
	err = newAzureMachinePoolV1Alpha3.ConvertTo(&newAzureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	for i, ownerRef := range newAzureMachinePool.ObjectMeta.OwnerReferences {
		if ownerRef.Kind == "MachinePool" && ownerRef.APIVersion != machinepoolmigration.DesiredCAPIGroupVersion {
			newAzureMachinePool.ObjectMeta.OwnerReferences[i].APIVersion = machinepoolmigration.DesiredCAPIGroupVersion
			newAzureMachinePool.ObjectMeta.OwnerReferences[i].UID = owner.UID
		}
	}

	err = r.client.Create(ctx, &newAzureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}
	r.logger.Debugf(ctx, "Ensured new AzureMachinePool %s/%s has been created", namespacedName.Namespace, namespacedName.Name)

	return nil
}
