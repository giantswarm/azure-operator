package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v5/encrypter"
	"github.com/giantswarm/certs"
)

type Interface interface {
	NewMasterCloudConfig(customObject providerv1alpha1.AzureConfig, certs certs.Cluster, encrypter encrypter.Encrypter) (string, error)
	NewWorkerCloudConfig(customObject providerv1alpha1.AzureConfig, certs certs.Cluster, encrypter encrypter.Encrypter) (string, error)
}
