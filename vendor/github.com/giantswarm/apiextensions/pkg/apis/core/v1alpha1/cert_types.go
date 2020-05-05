package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	crDocsAnnotation            = "giantswarm.io/docs"
	kindCertConfig              = "CertConfig"
	certConfigDocumentationLink = "https://pkg.go.dev/github.com/giantswarm/apiextensions/pkg/apis/core/v1alpha1?tab=doc#CertConfig"
)

func NewCertConfigCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(group, kindCertConfig)
}

// NewCertConfigTypeMeta returns the type part for the metadata section of a
// CertConfig custom resource.
func NewCertConfigTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: SchemeGroupVersion.String(),
		Kind:       kindCertConfig,
	}
}

// NewCertConfigCR returns an AWSCluster Custom Resource.
func NewCertConfigCR() *CertConfig {
	return &CertConfig{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				crDocsAnnotation: certConfigDocumentationLink,
			},
		},
		TypeMeta: NewCertConfigTypeMeta(),
	}
}

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CertConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              CertConfigSpec `json:"spec"`
}

type CertConfigSpec struct {
	Cert          CertConfigSpecCert          `json:"cert" yaml:"cert"`
	VersionBundle CertConfigSpecVersionBundle `json:"versionBundle" yaml:"versionBundle"`
}

type CertConfigSpecCert struct {
	AllowBareDomains    bool     `json:"allowBareDomains" yaml:"allowBareDomains"`
	AltNames            []string `json:"altNames" yaml:"altNames"`
	ClusterComponent    string   `json:"clusterComponent" yaml:"clusterComponent"`
	ClusterID           string   `json:"clusterID" yaml:"clusterID"`
	CommonName          string   `json:"commonName" yaml:"commonName"`
	DisableRegeneration bool     `json:"disableRegeneration" yaml:"disableRegeneration"`
	IPSANs              []string `json:"ipSans" yaml:"ipSans"`
	Organizations       []string `json:"organizations" yaml:"organizations"`
	TTL                 string   `json:"ttl" yaml:"ttl"`
}

type CertConfigSpecVersionBundle struct {
	Version string `json:"version" yaml:"version"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type CertConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []CertConfig `json:"items"`
}
