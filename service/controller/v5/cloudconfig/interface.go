package cloudconfig

import (
	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
)

type Interface interface {
	NewMasterCloudConfig(customObject providerv1alpha1.AzureConfig) (string, error)
	NewWorkerCloudConfig(customObject providerv1alpha1.AzureConfig) (string, error)
}
