package machinepoolmigration

import (
	capz "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	capzexp "sigs.k8s.io/cluster-api-provider-azure/exp/api/v1beta1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	capiexp "sigs.k8s.io/cluster-api/exp/api/v1beta1"
)

var (
	DesiredCAPIGroupVersion = capi.GroupVersion.String()
	DesiredCAPZGroupVersion = capz.GroupVersion.String()
)

func AreMachinePoolReferencesUpdated(machinePool capiexp.MachinePool) bool {
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

func AreAzureMachinePoolReferencesUpdated(azureMachinePool capzexp.AzureMachinePool) bool {
	// check MachinePool owner reference
	for _, ref := range azureMachinePool.ObjectMeta.OwnerReferences {
		if ref.Kind == "MachinePool" && ref.APIVersion != DesiredCAPIGroupVersion {
			return false
		}
	}

	return true
}
