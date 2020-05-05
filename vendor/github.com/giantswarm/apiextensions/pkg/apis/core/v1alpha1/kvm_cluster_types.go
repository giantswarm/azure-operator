package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	kindKVMClusterConfig = "KVMClusterConfig"
)

func NewKVMClusterConfigCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(group, kindKVMClusterConfig)
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KVMClusterConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              KVMClusterConfigSpec `json:"spec"`
}

type KVMClusterConfigSpec struct {
	Guest         KVMClusterConfigSpecGuest         `json:"guest" yaml:"guest"`
	VersionBundle KVMClusterConfigSpecVersionBundle `json:"versionBundle" yaml:"versionBundle"`
}

type KVMClusterConfigSpecGuest struct {
	ClusterGuestConfig `json:",inline" yaml:",inline"`
	Masters            []KVMClusterConfigSpecGuestMaster `json:"masters,omitempty" yaml:"masters,omitempty"`
	Workers            []KVMClusterConfigSpecGuestWorker `json:"workers,omitempty" yaml:"workers,omitempty"`
}

type KVMClusterConfigSpecGuestMaster struct {
	KVMClusterConfigSpecGuestNode `json:",inline" yaml:",inline"`
}

type KVMClusterConfigSpecGuestWorker struct {
	KVMClusterConfigSpecGuestNode `json:",inline" yaml:",inline"`
	Labels                        map[string]string `json:"labels" yaml:"labels"`
}

// TODO: change MemorySizeGB and StorageSizeGB to resource.Quantity
type KVMClusterConfigSpecGuestNode struct {
	ID       string `json:"id" yaml:"id"`
	CPUCores int    `json:"cpuCores,omitempty" yaml:"cpuCores,omitempty"`
	// +kubebuilder:validation:Type=number
	MemorySizeGB string `json:"memorySizeGB,omitempty" yaml:"memorySizeGB,omitempty"`
	// +kubebuilder:validation:Type=number
	StorageSizeGB string `json:"storageSizeGB,omitempty" yaml:"storageSizeGB,omitempty"`
}

type KVMClusterConfigSpecVersionBundle struct {
	Version string `json:"version" yaml:"version"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type KVMClusterConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []KVMClusterConfig `json:"items"`
}
