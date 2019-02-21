package blobobject

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/giantswarm/azure-operator/service/controller/v5/encrypter"
	"github.com/giantswarm/certs"
)

type CloudConfigMock struct {
	encrypter EncrypterMock
	template  string
}

func (c *CloudConfigMock) NewMasterCloudConfig(customObject providerv1alpha1.AzureConfig, certs certs.Cluster, encrypter encrypter.Encrypter) (string, error) {
	return c.template, nil
}

func (c *CloudConfigMock) NewWorkerCloudConfig(customObject providerv1alpha1.AzureConfig, certs certs.Cluster, encrypter encrypter.Encrypter) (string, error) {
	return c.template, nil
}
