package instance

import (
	"strings"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

const (
	// Types
	Stage                        = "Stage"
	DeploymentTemplateChecksum   = "TemplateChecksum"
	DeploymentParametersChecksum = "ParametersChecksum"

	// States
	ClusterUpgradeRequirementCheck = "ClusterUpgradeRequirementCheck"
	CordonOldWorkers               = "CordonOldWorkers"
	DeploymentUninitialized        = ""
	DeploymentInitialized          = "DeploymentInitialized"
	DeploymentCompleted            = "DeploymentCompleted"
	DrainOldWorkerNodes            = "DrainOldWorkerNodes"
	ProvisioningSuccessful         = "ProvisioningSuccessful"
	ScaleUpWorkerVMSS              = "ScaleUpWorkerVMSS"
	ScaleDownWorkerVMSS            = "ScaleDownWorkerVMSS"
	TerminateOldWorkerInstances    = "TerminateOldWorkerInstances"
	WaitForWorkersToBecomeReady    = "WaitForWorkersToBecomeReady"
)

func computeForDeleteResourceStatus(customObject providerv1alpha1.AzureConfig, n string, t string, s string) providerv1alpha1.AzureConfig {
	var allResources []providerv1alpha1.StatusClusterResource

	for _, r := range customObject.Status.Cluster.Resources {
		resourceStatus := providerv1alpha1.StatusClusterResource{
			Conditions: nil,
			Name:       n,
		}

		// At this point we ensure resource statuses of other resources are
		// preserved as they are. When we want to remove a status of resource A, but
		// find a status of resource B, we keep B, because we were not asked to
		// remove it.
		if !unversionedNameMatches(r.Name, n) {
			allResources = append(allResources, r)
			continue
		}

		// At this point we have a status of a resource we were asked to filter for
		// to some extend.
		for _, c := range r.Conditions {
			if c.Type == t && c.Status == s {
				continue
			}

			resourceStatus.Conditions = append(resourceStatus.Conditions, c)
		}

		// At this point we add the filtered resource status to the list of all
		// resource statuses we want to keep. In case the filter mechanism above
		// decided to filter all conditions, we do not add it to the list of
		// resource statuses, because we do not want to track resource statuses that
		// are essentially empty.
		if len(resourceStatus.Conditions) > 0 {
			allResources = append(allResources, resourceStatus)
		}
	}

	customObject.Status.Cluster.Resources = allResources

	return customObject
}

// unversionedName provides a comparable name without the exact version number
// suffix. When the resource name should ever change, the mechanism of managing
// the resource status has to be adapted accordingly. Otherwise unversionedName
// will return an empty string and the mechanism will break in unexpected ways.
func unversionedName(name string) string {
	if !strings.HasPrefix(name, "instance") {
		return ""
	}

	return "instance"
}

func unversionedNameMatches(a string, b string) bool {
	return unversionedName(a) == unversionedName(b)
}
