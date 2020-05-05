package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	kindChartConfig = "ChartConfig"
)

func NewChartConfigCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(group, kindChartConfig)
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:subresource:status

type ChartConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ChartConfigSpec `json:"spec"`
	// +kubebuilder:validation:Optional
	Status ChartConfigStatus `json:"status"`
}

type ChartConfigSpec struct {
	Chart         ChartConfigSpecChart         `json:"chart" yaml:"chart"`
	VersionBundle ChartConfigSpecVersionBundle `json:"versionBundle" yaml:"versionBundle"`
}

type ChartConfigSpecChart struct {
	// Channel is the name of the Appr channel to reconcile against,
	// e.g. 1-0-stable.
	Channel string `json:"channel" yaml:"channel"`
	// ConfigMap references a config map containing values that should be
	// applied to the chart.
	ConfigMap ChartConfigSpecConfigMap `json:"configMap" yaml:"configMap"`
	// UserConfigMap references a config map containing custom values.
	// These custom values are specified by the user to override default values.
	UserConfigMap ChartConfigSpecConfigMap `json:"userConfigMap" yaml:"userConfigMap"`
	// Name is the name of the Helm chart to deploy,
	// e.g. kubernetes-node-exporter.
	Name string `json:"name" yaml:"name"`
	// Namespace is the namespace where the Helm chart is to be deployed,
	// e.g. giantswarm.
	Namespace string `json:"namespace" yaml:"namespace"`
	// Release is the name of the Helm release when the chart is deployed,
	// e.g. node-exporter.
	Release string `json:"release" yaml:"release"`
	// Secret references a secret containing secret values that should be
	// applied to the chart.
	Secret ChartConfigSpecSecret `json:"secret" yaml:"secret"`
}

type ChartConfigSpecConfigMap struct {
	// Name is the name of the config map containing chart values to apply,
	// e.g. node-exporter-chart-values.
	Name string `json:"name" yaml:"name"`
	// Namespace is the namespace of the values config map,
	// e.g. kube-system.
	Namespace string `json:"namespace" yaml:"namespace"`
	// ResourceVersion is the Kubernetes resource version of the configmap.
	// Used to detect if the configmap has changed, e.g. 12345.
	ResourceVersion string `json:"resourceVersion" yaml:"resourceVersion"`
}

type ChartConfigSpecSecret struct {
	// Name is the name of the secret containing chart values to apply,
	// e.g. node-exporter-chart-secret.
	Name string `json:"name" yaml:"name"`
	// Namespace is the namespace of the secret,
	// e.g. kube-system.
	Namespace string `json:"namespace" yaml:"namespace"`
	// ResourceVersion is the Kubernetes resource version of the secret.
	// Used to detect if the secret has changed, e.g. 12345.
	ResourceVersion string `json:"resourceVersion" yaml:"resourceVersion"`
}

type ChartConfigStatus struct {
	// ReleaseStatus is the status of the Helm release when the chart is
	// installed, e.g. DEPLOYED.
	ReleaseStatus string `json:"releaseStatus" yaml:"releaseStatus"`
	// Reason is the description of the last status of helm release when the chart is
	// not installed successfully, e.g. deploy resource already exists.
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

type ChartConfigSpecVersionBundle struct {
	Version string `json:"version" yaml:"version"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ChartConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ChartConfig `json:"items"`
}
