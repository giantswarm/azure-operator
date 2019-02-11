package blobobject

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

type CloudConfigMock struct {
	template string
}

func (c *CloudConfigMock) NewMasterCloudConfig(customObject providerv1alpha1.AzureConfig) (string, error) {
	return c.template, nil
}

func (c *CloudConfigMock) NewWorkerCloudConfig(customObject providerv1alpha1.AzureConfig) (string, error) {
	return c.template, nil
}
