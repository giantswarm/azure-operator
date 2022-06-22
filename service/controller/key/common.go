package key

import (
	"fmt"
	"strings"
	"time"

	"github.com/giantswarm/microerror"
	releasev1alpha1 "github.com/giantswarm/release-operator/v3/api/v1alpha1"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"

	"github.com/giantswarm/azure-operator/v5/pkg/label"
	"github.com/giantswarm/azure-operator/v5/pkg/normalize"
)

const (
	// ComponentKubernetes is the name of the component specified in a Release
	// CR which determines the version of the Kubernetes to be used for tenant
	// cluster nodes.
	ComponentKubernetes = "kubernetes"

	// ComponentOS is the name of the component specified in a Release CR which
	// determines the version of the OS to be used for tenant cluster nodes and
	// is ultimately transformed into an AMI based on TC region.
	ComponentOS = "containerlinux"

	// TerminateUnhealthyNodeResyncPeriod defines resync period for the terminateunhealthynode controller
	TerminateUnhealthyNodeResyncPeriod = time.Minute * 3
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
	_, ok := getter.GetLabels()[capi.MachineControlPlaneLabelName]
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

func OrganizationNamespace(getter LabelsGetter) string {
	return normalize.AsDNSLabelName(fmt.Sprintf("org-%s", getter.GetLabels()[label.Organization]))
}

func ReleaseVersion(getter LabelsGetter) string {
	return getter.GetLabels()[label.ReleaseVersion]
}

func ReleaseName(releaseVersion string) string {
	if strings.HasPrefix(releaseVersion, "v") {
		return releaseVersion
	}
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

func KubernetesVersion(release releasev1alpha1.Release) (string, error) {
	return ComponentVersion(release, ComponentKubernetes)
}

func OSVersion(release releasev1alpha1.Release) (string, error) {
	return ComponentVersion(release, ComponentOS)
}

func EncryptionConfigSecretName(clusterName string) string {
	return fmt.Sprintf("%s-encryption-provider-config", clusterName)
}
