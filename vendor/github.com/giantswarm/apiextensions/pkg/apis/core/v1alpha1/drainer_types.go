package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	DrainerConfigStatusStatusTrue = "True"
)

const (
	DrainerConfigStatusTypeDrained = "Drained"
)

const (
	DrainerConfigStatusTypeTimeout = "Timeout"
)

const (
	kindDrainerConfig = "DrainerConfig"
)

func NewDrainerConfigCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(group, kindDrainerConfig)
}

func NewDrainerTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: version,
		Kind:       kindDrainerConfig,
	}
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

type DrainerConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              DrainerConfigSpec `json:"spec"`
	// +kubebuilder:validation:Optional
	Status DrainerConfigStatus `json:"status"`
}

type DrainerConfigSpec struct {
	Guest         DrainerConfigSpecGuest         `json:"guest" yaml:"guest"`
	VersionBundle DrainerConfigSpecVersionBundle `json:"versionBundle" yaml:"versionBundle"`
}

type DrainerConfigSpecGuest struct {
	Cluster DrainerConfigSpecGuestCluster `json:"cluster" yaml:"cluster"`
	Node    DrainerConfigSpecGuestNode    `json:"node" yaml:"node"`
}

type DrainerConfigSpecGuestCluster struct {
	API DrainerConfigSpecGuestClusterAPI `json:"api" yaml:"api"`
	// ID is the guest cluster ID of which a node should be drained.
	ID string `json:"id" yaml:"id"`
}

type DrainerConfigSpecGuestClusterAPI struct {
	// Endpoint is the guest cluster API endpoint.
	Endpoint string `json:"endpoint" yaml:"endpoint"`
}

type DrainerConfigSpecGuestNode struct {
	// Name is the identifier of the guest cluster's master and worker nodes. In
	// Kubernetes/Kubectl they are represented as node names. The names are manage
	// in an abstracted way because of provider specific differences.
	//
	//     AWS: EC2 instance DNS.
	//     Azure: VM name.
	//     KVM: host cluster pod name.
	//
	Name string `json:"name" yaml:"name"`
}

type DrainerConfigSpecVersionBundle struct {
	Version string `json:"version" yaml:"version"`
}

type DrainerConfigStatus struct {
	Conditions []DrainerConfigStatusCondition `json:"conditions" yaml:"conditions"`
}

// DrainerConfigStatusCondition expresses a condition in which a node may is.
type DrainerConfigStatusCondition struct {
	// LastHeartbeatTime is the last time we got an update on a given condition.
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime" yaml:"lastHeartbeatTime"`
	// LastTransitionTime is the last time the condition transitioned from one
	// status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime" yaml:"lastTransitionTime"`
	// Status may be True, False or Unknown.
	Status string `json:"status" yaml:"status"`
	// Type may be Pending, Ready, Draining, Drained.
	Type string `json:"type" yaml:"type"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type DrainerConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []DrainerConfig `json:"items"`
}
