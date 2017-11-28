package cloudconfig

import (
	"encoding/base64"

	"github.com/giantswarm/azuretpr"
	k8scloudconfig "github.com/giantswarm/k8scloudconfig/v_0_1_0"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
)

// Config represents the configuration used to create a cloudconfig service.
type Config struct {
	// Dependencies.

	Logger micrologger.Logger
}

// DefaultConfig provides a default configuration to create a new cloudconfig service
// by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		Logger: nil,
	}
}

// CloudConfig implements the cloudconfig service interface.
type CloudConfig struct {
	// Dependencies.
	logger micrologger.Logger
}

// New creates a new configured cloudconfig service.
func New(config Config) (*CloudConfig, error) {
	// Dependencies.
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "logger must not be empty")
	}

	newService := &CloudConfig{
		// Dependencies.
		logger: config.Logger,
	}

	return newService, nil
}

// NewMasterCloudConfig generates a new master cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewMasterCloudConfig(customObject azuretpr.CustomObject) (string, error) {
	params := k8scloudconfig.Params{
		Cluster: customObject.Spec.Cluster,
		Extension: &MasterExtension{
			CloudConfigExtension{
				CustomObject: customObject,
			},
		},
	}

	return newCloudConfig(k8scloudconfig.MasterTemplate, params)
}

// NewWorkerCloudConfig generates a new worker cloudconfig and returns it as a
// base64 encoded string.
func (c CloudConfig) NewWorkerCloudConfig(customObject azuretpr.CustomObject) (string, error) {
	params := k8scloudconfig.Params{
		Cluster: customObject.Spec.Cluster,
		Extension: &WorkerExtension{
			CloudConfigExtension{
				CustomObject: customObject,
			},
		},
	}

	return newCloudConfig(k8scloudconfig.WorkerTemplate, params)
}

func newCloudConfig(template string, params k8scloudconfig.Params) (string, error) {
	cloudConfig, err := k8scloudconfig.NewCloudConfig(template, params)
	if err != nil {
		return "", microerror.Mask(err)
	}
	err = cloudConfig.ExecuteTemplate()
	if err != nil {
		return "", microerror.Mask(err)
	}

	return base64.StdEncoding.EncodeToString([]byte(cloudConfig.String())), nil
}
