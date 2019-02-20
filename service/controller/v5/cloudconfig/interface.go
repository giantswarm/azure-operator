package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/certs"
)

type Interface interface {
	GetEncryptionKey() string
	GetInitialVector() string
	NewMasterCloudConfig(customObject providerv1alpha1.AzureConfig, certs certs.Cluster) (string, error)
	NewWorkerCloudConfig(customObject providerv1alpha1.AzureConfig, certs certs.Cluster) (string, error)
}
