package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	kindFlannelConfig = "FlannelConfig"
)

func NewFlannelConfigCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(group, kindFlannelConfig)
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FlannelConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              FlannelConfigSpec `json:"spec"`
}

type FlannelConfigSpec struct {
	Bridge        FlannelConfigSpecBridge        `json:"bridge" yaml:"bridge"`
	Cluster       FlannelConfigSpecCluster       `json:"cluster" yaml:"cluster"`
	Flannel       FlannelConfigSpecFlannel       `json:"flannel" yaml:"flannel"`
	Health        FlannelConfigSpecHealth        `json:"health" yaml:"health"`
	VersionBundle FlannelConfigSpecVersionBundle `json:"versionBundle" yaml:"versionBundle"`
}

type FlannelConfigSpecBridge struct {
	Docker FlannelConfigSpecBridgeDocker `json:"docker" yaml:"docker"`
	Spec   FlannelConfigSpecBridgeSpec   `json:"spec" yaml:"spec"`
}

type FlannelConfigSpecBridgeDocker struct {
	Image string `json:"image" yaml:"image"`
}

type FlannelConfigSpecBridgeSpec struct {
	Interface      string                         `json:"interface" yaml:"interface"`
	PrivateNetwork string                         `json:"privateNetwork" yaml:"privateNetwork"`
	DNS            FlannelConfigSpecBridgeSpecDNS `json:"dns" yaml:"dns"`
	NTP            FlannelConfigSpecBridgeSpecNTP `json:"ntp" yaml:"ntp"`
}

type FlannelConfigSpecBridgeSpecDNS struct {
	Servers []string `json:"servers" yaml:"servers"`
}

type FlannelConfigSpecBridgeSpecNTP struct {
	Servers []string `json:"servers" yaml:"servers"`
}

type FlannelConfigSpecCluster struct {
	ID        string `json:"id" yaml:"id"`
	Customer  string `json:"customer" yaml:"customer"`
	Namespace string `json:"namespace" yaml:"namespace"`
}

type FlannelConfigSpecFlannel struct {
	Spec FlannelConfigSpecFlannelSpec `json:"spec" yaml:"spec"`
}

type FlannelConfigSpecFlannelSpec struct {
	Network   string `json:"network" yaml:"network"`
	SubnetLen int    `json:"subnetLen" yaml:"subnetLen"`
	RunDir    string `json:"runDir" yaml:"runDir"`
	VNI       int    `json:"vni" yaml:"vni"`
}

type FlannelConfigSpecHealth struct {
	Docker FlannelConfigSpecHealthDocker `json:"docker" yaml:"docker"`
}

type FlannelConfigSpecHealthDocker struct {
	Image string `json:"image" yaml:"image"`
}

type FlannelConfigSpecVersionBundle struct {
	Version string `json:"version" yaml:"version"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type FlannelConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []FlannelConfig `json:"items"`
}
