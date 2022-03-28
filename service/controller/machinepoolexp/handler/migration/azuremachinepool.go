package migration

import (
	"context"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v5/pkg/apis/capiexp/v1alpha3"
	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v5/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capzexpv1alpha3 "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1alpha3"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
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

func (r *Resource) ensureNewAzureMachinePoolCreated(ctx context.Context, namespacedName types.NamespacedName) error {
	// First let's check if new AzureMachinePool has already been created.
	exists, err := r.newAzureMachinePoolExists(ctx, namespacedName)
	if err != nil {
		return microerror.Mask(err)
	}
	if exists {
		return nil
	}

	oldAzureMachinePoolV1alpha3 := oldcapzexpv1alpha3.AzureMachinePool{}
	err = r.client.Get(ctx, namespacedName, &oldAzureMachinePoolV1alpha3)
	if apierrors.IsNotFound(err) {
		// New AzureMachinePool does not exist, and the old one does not exist,
		// so there is just a MachinePool CR.
		// TODO: check if it is possible that AzureMachinePool does not exist.
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

	err = r.client.Create(ctx, &newAzureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}

func (r *Resource) ensureNewAzureMachinePoolReferencesUpdated(ctx context.Context, namespacedName types.NamespacedName) error {
	newAzureMachinePool := capzexp.AzureMachinePool{}
	err := r.client.Get(ctx, namespacedName, &newAzureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	for i, ownerRef := range newAzureMachinePool.ObjectMeta.OwnerReferences {
		if ownerRef.APIVersion == oldcapiexpv1alpha3.GroupVersion.String() {
			newAzureMachinePool.ObjectMeta.OwnerReferences[i].APIVersion = capiexp.GroupVersion.String()
		}
	}

	err = r.client.Update(ctx, &newAzureMachinePool)
	if err != nil {
		return microerror.Mask(err)
	}

	return nil
}
