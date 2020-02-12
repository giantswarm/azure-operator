package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"

	"github.com/giantswarm/azure-operator/service/controller/v13/encrypter"
)

type Interface interface {
	NewMasterCloudConfig(customObject *providerv1alpha1.AzureConfig, certs certs.Cluster, encrypter encrypter.Interface) (string, error)
	NewWorkerCloudConfig(customObject *providerv1alpha1.AzureConfig, certs certs.Cluster, encrypter encrypter.Interface) (string, error)
}
