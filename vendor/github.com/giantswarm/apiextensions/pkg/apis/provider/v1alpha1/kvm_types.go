package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	kindKVMConfig = "KVMConfig"
)

func NewKVMConfigCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(group, kindKVMConfig)
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

type KVMConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              KVMConfigSpec `json:"spec"`
	// +kubebuilder:validation:Optional
	Status KVMConfigStatus `json:"status" yaml:"status"`
}

type KVMConfigSpec struct {
	Cluster       Cluster                    `json:"cluster" yaml:"cluster"`
	KVM           KVMConfigSpecKVM           `json:"kvm" yaml:"kvm"`
	VersionBundle KVMConfigSpecVersionBundle `json:"versionBundle" yaml:"versionBundle"`
}

type KVMConfigSpecKVM struct {
	EndpointUpdater KVMConfigSpecKVMEndpointUpdater `json:"endpointUpdater" yaml:"endpointUpdater"`
	K8sKVM          KVMConfigSpecKVMK8sKVM          `json:"k8sKVM" yaml:"k8sKVM"`
	Masters         []KVMConfigSpecKVMNode          `json:"masters" yaml:"masters"`
	Network         KVMConfigSpecKVMNetwork         `json:"network" yaml:"network"`
	// NOTE THIS IS DEPRECATED
	NodeController KVMConfigSpecKVMNodeController `json:"nodeController" yaml:"nodeController"`
	PortMappings   []KVMConfigSpecKVMPortMappings `json:"portMappings" yaml:"portMappings"`
	Workers        []KVMConfigSpecKVMNode         `json:"workers" yaml:"workers"`
}

type KVMConfigSpecKVMEndpointUpdater struct {
	Docker KVMConfigSpecKVMEndpointUpdaterDocker `json:"docker" yaml:"docker"`
}

type KVMConfigSpecKVMEndpointUpdaterDocker struct {
	Image string `json:"image" yaml:"image"`
}

type KVMConfigSpecKVMK8sKVM struct {
	Docker      KVMConfigSpecKVMK8sKVMDocker `json:"docker" yaml:"docker"`
	StorageType string                       `json:"storageType" yaml:"storageType"`
}

type KVMConfigSpecKVMK8sKVMDocker struct {
	Image string `json:"image" yaml:"image"`
}

type KVMConfigSpecKVMNode struct {
	CPUs int `json:"cpus" yaml:"cpus"`
	// +kubebuilder:validation:Type=number
	Disk               string `json:"disk" yaml:"disk"`
	Memory             string `json:"memory" yaml:"memory"`
	DockerVolumeSizeGB int    `json:"dockerVolumeSizeGB" yaml:"dockerVolumeSizeGB"`
}

type KVMConfigSpecKVMNetwork struct {
	Flannel KVMConfigSpecKVMNetworkFlannel `json:"flannel" yaml:"flannel"`
}

type KVMConfigSpecKVMNetworkFlannel struct {
	VNI int `json:"vni" yaml:"vni"`
}

// NOTE THIS IS DEPRECATED
type KVMConfigSpecKVMNodeController struct {
	Docker KVMConfigSpecKVMNodeControllerDocker `json:"docker" yaml:"docker"`
}

// NOTE THIS IS DEPRECATED
type KVMConfigSpecKVMNodeControllerDocker struct {
	Image string `json:"image" yaml:"image"`
}

type KVMConfigSpecKVMPortMappings struct {
	Name       string `json:"name" yaml:"name"`
	NodePort   int    `json:"nodePort" yaml:"nodePort"`
	TargetPort int    `json:"targetPort" yaml:"targetPort"`
}

type KVMConfigSpecVersionBundle struct {
	Version string `json:"version" yaml:"version"`
}

type KVMConfigStatus struct {
	Cluster StatusCluster      `json:"cluster" yaml:"cluster"`
	KVM     KVMConfigStatusKVM `json:"kvm" yaml:"kvm"`
}

type KVMConfigStatusKVM struct {
	// NodeIndexes is a map from nodeID -> nodeIndex. This is used to create deterministic iSCSI initiator names.
	NodeIndexes map[string]int `json:"nodeIndexes" yaml:"nodeIndexes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KVMConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []KVMConfig `json:"items"`
}
