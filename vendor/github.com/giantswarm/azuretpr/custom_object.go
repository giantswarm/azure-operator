package azuretpr

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CustomObject represents the Azure custom object. It holds the specification
// of the resources the Azure operator is managing.
type CustomObject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              Spec `json:"spec" yaml:"spec"`
}
