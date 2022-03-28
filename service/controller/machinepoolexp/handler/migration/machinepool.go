package migration

import (
	"context"

	oldcapiexpv1alpha3 "github.com/giantswarm/apiextensions/v5/pkg/apis/capiexp/v1alpha3"
	oldcapzexpv1alpha3 "github.com/giantswarm/apiextensions/v5/pkg/apis/capzexp/v1alpha3"
	"github.com/giantswarm/microerror"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexpv1alpha3 "sigs.k8s.io/cluster-api/exp/api/v1alpha3"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
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
		return nil
	}

	// Let's create a new non-exp v1alpha3 MachinePool where we will clone
	// Metadata from the old exp v1alpha3 MachinePool.
	newMachinePoolV1Alpha3 := capiexpv1alpha3.MachinePool{
		ObjectMeta: cloneObjectMeta(oldMachinePoolV1alpha3.ObjectMeta),
	}

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

	return nil
}

func (r *Resource) ensureNewMachinePoolReferencesUpdated(ctx context.Context, namespacedName types.NamespacedName) error {
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

	return nil
}
