package v1alpha2

import (
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	kindAWSMachineDeployment = "AWSMachineDeployment"
)

func NewAWSMachineDeploymentCRD() *v1.CustomResourceDefinition {
	return crd.LoadV1(group, kindAWSMachineDeployment)
}

func NewAWSMachineDeploymentTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: SchemeGroupVersion.String(),
		Kind:       kindAWSMachineDeployment,
	}
}

// NewAWSMachineDeploymentCR returns an AWSMachineDeployment Custom Resource.
func NewAWSMachineDeploymentCR() *AWSMachineDeployment {
	return &AWSMachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				crDocsAnnotation: awsClusterDocumentationLink,
			},
		},
		TypeMeta: NewAWSMachineDeploymentTypeMeta(),
	}
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

// AWSMachineDeployment is the infrastructure provider referenced in Kubernetes Cluster API MachineDeployment resources.
// It contains provider-specific specification and status for a node pool.
// In use on AWS since Giant Swarm release v10.x.x and reconciled by aws-operator.
type AWSMachineDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Contains the specification.
	Spec AWSMachineDeploymentSpec `json:"spec" yaml:"spec"`
	// +kubebuilder:validation:Optional
	// Holds status information.
	Status AWSMachineDeploymentStatus `json:"status" yaml:"status"`
}

type AWSMachineDeploymentSpec struct {
	// Specifies details of node pool and the worker nodes it should contain.
	NodePool AWSMachineDeploymentSpecNodePool `json:"nodePool" yaml:"nodePool"`
	// Contains AWS specific details.
	Provider AWSMachineDeploymentSpecProvider `json:"provider" yaml:"provider"`
}

type AWSMachineDeploymentSpecNodePool struct {
	// User-friendly name or description of the purpose of the node pool.
	Description string `json:"description" yaml:"description"`
	// Specification of the worker node machine.
	Machine AWSMachineDeploymentSpecNodePoolMachine `json:"machine" yaml:"machine"`
	// Scaling settings for the node pool, configuring the cluster-autosaler
	// determining the number of nodes to have in this node pool.
	Scaling AWSMachineDeploymentSpecNodePoolScaling `json:"scaling" yaml:"scaling"`
}

type AWSMachineDeploymentSpecNodePoolMachine struct {
	// Size of the volume reserved for Docker images and overlay file systems of
	// Docker containers. Unit: 1 GB = 1,000,000,000 Bytes.
	DockerVolumeSizeGB int `json:"dockerVolumeSizeGB" yaml:"dockerVolumeSizeGB"`
	// Size of the volume reserved for the kubelet, which can be used by Pods via
	// volumes of type EmptyDir. Unit: 1 GB = 1,000,000,000 Bytes.
	KubeletVolumeSizeGB int `json:"kubeletVolumeSizeGB" yaml:"kubeletVolumeSizeGB"`
}

type AWSMachineDeploymentSpecNodePoolScaling struct {
	// Maximum number of worker nodes in this node pool.
	Max int `json:"max" yaml:"max"`
	// Minimum number of worker nodes in this node pool.
	Min int `json:"min" yaml:"min"`
}

type AWSMachineDeploymentSpecProvider struct {
	// Name(s) of the availability zone(s) to use for worker nodes. Using multiple
	// availability zones results in higher resilience but can also result in higher
	// cost due to network traffic between availability zones.
	AvailabilityZones []string `json:"availabilityZones" yaml:"availabilityZones"`
	// +kubebuilder:validation:Optional
	// Settings defining the distribution of on-demand and spot instances in the node pool.
	InstanceDistribution AWSMachineDeploymentSpecInstanceDistribution `json:"instanceDistribution" yaml:"instanceDistribution"`
	// Specification of worker nodes.
	Worker AWSMachineDeploymentSpecProviderWorker `json:"worker" yaml:"worker"`
}

type AWSMachineDeploymentSpecInstanceDistribution struct {
	// +kubebuilder:default=0
	// +kubebuilder:validation:Minimum=0
	// Base capacity of on-demand instances to use for worker nodes in this pool. When this larger
	// than 0, this value defines a number of worker nodes that will be created using on-demand
	// EC2 instances, regardless of the value configured as `onDemandPercentageAboveBaseCapacity`.
	OnDemandBaseCapacity int `json:"onDemandBaseCapacity" yaml:"onDemandBaseCapacity"`
	// +kubebuilder:default=100
	// +kubebuilder:validation:Maximum=100
	// +kubebuilder:validation:Minimum=0
	// Percentage of on-demand EC2 instances to use for worker nodes, instead of spot instances,
	// for instances exceeding `onDemandBaseCapacity`. For example, to have half of the worker nodes
	// use spot instances and half use on-demand, set this value to 50.
	OnDemandPercentageAboveBaseCapacity int `json:"onDemandPercentageAboveBaseCapacity" yaml:"onDemandPercentageAboveBaseCapacity"`
}

type AWSMachineDeploymentSpecProviderWorker struct {
	// AWS EC2 instance type name to use for the worker nodes in this node pool.
	InstanceType string `json:"instanceType" yaml:"instanceType"`
	// +kubebuilder:default=false
	// If true, certain instance types with specs similar to instanceType will be used.
	UseAlikeInstanceTypes bool `json:"useAlikeInstanceTypes" yaml:"useAlikeInstanceTypes"`
}

type AWSMachineDeploymentStatus struct {
	// +kubebuilder:validation:Optional
	// Status specific to AWS.
	Provider AWSMachineDeploymentStatusProvider `json:"provider" yaml:"provider"`
}

type AWSMachineDeploymentStatusProvider struct {
	// +kubebuilder:validation:Optional
	// Status of worker nodes.
	Worker AWSMachineDeploymentStatusProviderWorker `json:"worker" yaml:"worker"`
}

type AWSMachineDeploymentStatusProviderWorker struct {
	// +kubebuilder:validation:Optional
	// AWS EC2 instance types used for the worker nodes in this node pool.
	InstanceTypes []string `json:"instanceTypes" yaml:"instanceTypes"`
	// +kubebuilder:validation:Optional
	// Number of EC2 spot instances used in this node pool.
	SpotInstances int `json:"spotInstances" yaml:"spotInstances"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AWSMachineDeploymentList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []AWSMachineDeployment `json:"items" yaml:"items"`
}
