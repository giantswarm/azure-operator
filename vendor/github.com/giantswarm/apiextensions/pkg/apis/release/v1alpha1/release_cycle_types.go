package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	kindReleaseCycle = "ReleaseCycle"

	CyclePhaseUpcoming ReleaseCyclePhase = "upcoming"
	CyclePhaseEnabled  ReleaseCyclePhase = "enabled"
	CyclePhaseDisabled ReleaseCyclePhase = "disabled"
	CyclePhaseEOL      ReleaseCyclePhase = "eol"
)

type ReleaseCyclePhase string

func (r ReleaseCyclePhase) String() string {
	return string(r)
}

func NewReleaseCycleCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(group, kindReleaseCycle)
}

func NewReleaseCycleTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: SchemeGroupVersion.String(),
		Kind:       kindReleaseCycle,
	}
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

type ReleaseCycle struct {
	metav1.TypeMeta   `json:",inline" yaml:",inline"`
	metav1.ObjectMeta `json:"metadata" yaml:"metadata"`
	Spec              ReleaseCycleSpec `json:"spec" yaml:"spec"`
	// +kubebuilder:validation:Optional
	Status ReleaseCycleStatus `json:"status,omitempty" yaml:"status,omitempty"`
}

type ReleaseCycleSpec struct {
	// DisabledDate is the date of the cycle phase being changed to "disabled".
	DisabledDate DeepCopyDate `json:"disabledDate,omitempty" yaml:"disabledDate,omitempty"`
	// EnabledDate is the date of the cycle phase being changed to "enabled".
	EnabledDate DeepCopyDate `json:"enabledDate,omitempty" yaml:"enabledDate,omitempty"`
	// Phase is the release phase. It can be one of: "upcoming", "enabled",
	// "disabled", "eol".
	Phase ReleaseCyclePhase `json:"phase" yaml:"phase"`
}

type ReleaseCycleStatus struct {
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ReleaseCycleList struct {
	metav1.TypeMeta `json:",inline" yaml:",inline"`
	metav1.ListMeta `json:"metadata" yaml:"metadata"`
	Items           []ReleaseCycle `json:"items" yaml:"items"`
}
