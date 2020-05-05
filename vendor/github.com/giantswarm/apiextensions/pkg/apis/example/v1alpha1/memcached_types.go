package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	kindMemcachedConfig = "MemcachedConfig"
)

func NewMemcachedConfigCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(group, kindMemcachedConfig)
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MemcachedConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              MemcachedConfigSpec `json:"spec"`
}

type MemcachedConfigSpec struct {
	// Replicas is the number of instances of Memcache.
	Replicas int `json:"replicas" yaml:"replicas"`
	// e.g. 3
	// Memory is how much RAM to use for item storage.
	// e.g. 4G
	Memory string `json:"memory" yaml:"memory"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MemcachedConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []MemcachedConfig `json:"items"`
}
