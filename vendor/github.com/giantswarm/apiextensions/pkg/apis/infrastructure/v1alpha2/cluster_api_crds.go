package v1alpha2

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1alpha2 "sigs.k8s.io/cluster-api/api/v1alpha2"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	kindCluster                        = "Cluster"
	kindMachineDeployment              = "MachineDeployment"
	clusterDocumentationLink           = "https://pkg.go.dev/sigs.k8s.io/cluster-api/api/v1alpha2?tab=doc#Cluster"
	machineDeploymentDocumentationLink = "https://pkg.go.dev/sigs.k8s.io/cluster-api/api/v1alpha2?tab=doc#MachineDeployment"
)

func NewClusterCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(clusterAPIGroup, kindCluster)
}

func NewClusterTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: SchemeGroupVersion.String(),
		Kind:       kindCluster,
	}
}

// NewClusterCR returns a Cluster Custom Resource.
func NewClusterCR() *apiv1alpha2.Cluster {
	return &apiv1alpha2.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				crDocsAnnotation: clusterDocumentationLink,
			},
		},
		TypeMeta: NewClusterTypeMeta(),
	}
}

func NewMachineDeploymentCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(clusterAPIGroup, kindMachineDeployment)
}

// NewMachineDeploymentTypeMeta returns the type block for a MachineDeployment CR.
func NewMachineDeploymentTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: SchemeGroupVersion.String(),
		Kind:       kindMachineDeployment,
	}
}

// NewMachineDeploymentCR returns a MachineDeployment Custom Resource.
func NewMachineDeploymentCR() *apiv1alpha2.MachineDeployment {
	return &apiv1alpha2.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				crDocsAnnotation: machineDeploymentDocumentationLink,
			},
		},
		TypeMeta: NewMachineDeploymentTypeMeta(),
	}
}
