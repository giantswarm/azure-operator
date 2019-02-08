package blobobject

import (
	"crypto/aes"
	"encoding/hex"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

type CloudConfigMock struct {
	encrypter EncrypterMock
	template  string
}

type EncrypterMock struct {
	key []byte
}

func (c *CloudConfigMock) GetEncryptionKey() string {
	return hex.EncodeToString(c.encrypter.key[aes.BlockSize:])
}

func (c *CloudConfigMock) GetInitialVector() string {
	return hex.EncodeToString(c.encrypter.key[:aes.BlockSize])
}

func (c *CloudConfigMock) NewMasterCloudConfig(customObject providerv1alpha1.AzureConfig) (string, error) {
	return c.template, nil
}

func (c *CloudConfigMock) NewWorkerCloudConfig(customObject providerv1alpha1.AzureConfig) (string, error) {
	return c.template, nil
}
