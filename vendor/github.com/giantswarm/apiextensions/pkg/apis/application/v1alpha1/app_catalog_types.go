package v1alpha1

import (
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/giantswarm/apiextensions/pkg/crd"
)

const (
	crDocsAnnotation            = "giantswarm.io/docs"
	kindAppCatalog              = "AppCatalog"
	appCatalogDocumentationLink = "https://pkg.go.dev/github.com/giantswarm/apiextensions/pkg/apis/application/v1alpha1?tab=doc#AppCatalog"
)

func NewAppCatalogCRD() *v1beta1.CustomResourceDefinition {
	return crd.LoadV1Beta1(group, kindAppCatalog)
}

func NewAppCatalogTypeMeta() metav1.TypeMeta {
	return metav1.TypeMeta{
		APIVersion: SchemeGroupVersion.String(),
		Kind:       kindAppCatalog,
	}
}

// NewAppCatalogCR returns an AppCatalog Custom Resource.
func NewAppCatalogCR() *AppCatalog {
	return &AppCatalog{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				crDocsAnnotation: appCatalogDocumentationLink,
			},
		},
		TypeMeta: NewAppCatalogTypeMeta(),
	}
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AppCatalog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              AppCatalogSpec `json:"spec"`
}

type AppCatalogSpec struct {
	// Title is the name of the app catalog for this CR
	// e.g. Catalog of Apps by Giant Swarm
	Title       string `json:"title" yaml:"title"`
	Description string `json:"description" yaml:"description"`
	// Config is the config to be applied when apps belonging to this
	// catalog are deployed.
	Config AppCatalogSpecConfig `json:"config" yaml:"config"`
	// LogoURL contains the links for logo image file for this app catalog
	LogoURL string `json:"logoURL" yaml:"logoURL"`
	// Storage references a map containing values that should be applied to
	// the appcatalog.
	Storage AppCatalogSpecStorage `json:"storage" yaml:"storage"`
}

type AppCatalogSpecConfig struct {
	// ConfigMap references a config map containing catalog values that
	// should be applied to apps in this catalog.
	ConfigMap AppCatalogSpecConfigConfigMap `json:"configMap" yaml:"configMap"`
	// Secret references a secret containing catalog values that should be
	// applied to apps in this catalog.
	Secret AppCatalogSpecConfigSecret `json:"secret" yaml:"secret"`
}

type AppCatalogSpecConfigConfigMap struct {
	// Name is the name of the config map containing catalog values to
	// apply, e.g. app-catalog-values.
	Name string `json:"name" yaml:"name"`
	// Namespace is the namespace of the catalog values config map,
	// e.g. giantswarm.
	Namespace string `json:"namespace" yaml:"namespace"`
}

type AppCatalogSpecConfigSecret struct {
	// Name is the name of the secret containing catalog values to apply,
	// e.g. app-catalog-secret.
	Name string `json:"name" yaml:"name"`
	// Namespace is the namespace of the secret,
	// e.g. giantswarm.
	Namespace string `json:"namespace" yaml:"namespace"`
}

type AppCatalogSpecStorage struct {
	// Type indicates which repository type would be used for this AppCatalog.
	// e.g. helm
	Type string `json:"type" yaml:"type"`
	// URL is the link to where this AppCatalog's repository is located
	// e.g. https://example.com/app-catalog/
	URL string `json:"URL" yaml:"URL"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type AppCatalogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []AppCatalog `json:"items"`
}
