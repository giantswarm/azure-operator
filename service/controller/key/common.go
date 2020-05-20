package key

import (
	"fmt"

	releasev1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/release/v1alpha1"
	"github.com/giantswarm/microerror"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"

	"github.com/giantswarm/azure-operator/v4/pkg/label"
)

const (
	// ComponentOS is the name of the component specified in a Release CR which
	// determines the version of the OS to be used for tenant cluster nodes and
	// is ultimately transformed into an AMI based on TC region.
	ComponentOS = "containerlinux"
)

func ClusterCloudProviderTag(getter LabelsGetter) string {
	return fmt.Sprintf("kubernetes.io/cluster/%s", ClusterID(getter))
}

func ClusterID(getter LabelsGetter) string {
	return getter.GetLabels()[label.Cluster]
}

func HealthCheckTarget(port int) string {
	return fmt.Sprintf("TCP:%d", port)
}

func InternalELBNameAPI(getter LabelsGetter) string {
	return fmt.Sprintf("%s-api-internal", ClusterID(getter))
}

func IsControlPlaneMachine(getter LabelsGetter) bool {
	_, ok := getter.GetLabels()[capiv1alpha3.MachineControlPlaneLabelName]
	return ok
}

func IsDeleted(getter DeletionTimestampGetter) bool {
	return getter.GetDeletionTimestamp() != nil
}

func OperatorVersion(getter LabelsGetter) string {
	return getter.GetLabels()[label.OperatorVersion]
}

func OrganizationID(getter LabelsGetter) string {
	return getter.GetLabels()[label.Organization]
}

func ReleaseVersion(getter LabelsGetter) string {
	return getter.GetLabels()[label.ReleaseVersion]
}

func ReleaseName(releaseVersion string) string {
	return fmt.Sprintf("v%s", releaseVersion)
}

func ComponentVersion(release releasev1alpha1.Release, componentName string) (string, error) {
	for _, component := range release.Spec.Components {
		if component.Name == componentName {
			return component.Version, nil
		}
	}
	return "", microerror.Maskf(notFoundError, "version for component %#v not found on release %#v", componentName, release.Name)
}

func OSVersion(release releasev1alpha1.Release) (string, error) {
	return ComponentVersion(release, ComponentOS)
}