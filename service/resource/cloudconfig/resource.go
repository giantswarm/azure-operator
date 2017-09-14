package cloudconfig

import (
	"github.com/giantswarm/azuretpr"
	"github.com/giantswarm/certificatetpr"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"
	"github.com/giantswarm/operatorkit/framework"

	"github.com/giantswarm/azure-operator/client"
)

const (
	// Name is the identifier of the resource.
	Name = "cloudconfig"
)

// Config represents the configuration used to create a new cloud config resource.
type Config struct {
	// Dependencies.

	AzureConfig *client.AzureConfig
	CertWatcher certificatetpr.Searcher
	Logger      micrologger.Logger
}

// DefaultConfig provides a default configuration to create a new cloud config
// resource by best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		AzureConfig: nil,
		CertWatcher: nil,
		Logger:      nil,
	}
}

// Resource implements the cloud config resource.
type Resource struct {
	// Dependencies.
	azureConfig *client.AzureConfig
	certWatcher certificatetpr.Searcher
	logger      micrologger.Logger
}

// New creates a new configured cloud config resource.
func New(config Config) (*Resource, error) {
	// Dependencies.
	if config.AzureConfig == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.AzureConfig must not be empty")
	}
	if config.CertWatcher == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.CertWatcher must not be empty")
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "config.Logger must not be empty")
	}

	newService := &Resource{
		// Dependencies.
		azureConfig: config.AzureConfig,
		certWatcher: config.CertWatcher,
		logger: config.Logger.With(
			"resource", Name,
		),
	}

	return newService, nil
}

// TODO GetCurrentState is not yet implemented.
func (r *Resource) GetCurrentState(obj interface{}) (interface{}, error) {
	return []CloudConfigBlob{}, nil
}

// TODO GetDesiredState is not yet implemented.
func (r *Resource) GetDesiredState(obj interface{}) (interface{}, error) {
	return []CloudConfigBlob{}, nil
}

// GetCreateState returns the cloud config blobs that should be created for
// this cluster.
func (r *Resource) GetCreateState(obj, currentState, desiredState interface{}) (interface{}, error) {
	currentCloudConfigs, err := toCloudConfigBlobs(currentState)
	if err != nil {
		return nil, microerror.Mask(err)
	}
	desiredCloudConfigs, err := toCloudConfigBlobs(desiredState)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	var cloudConfigsToCreate []CloudConfigBlob

	for _, desiredCloudConfig := range desiredCloudConfigs {
		if !existsCloudConfigByName(currentCloudConfigs, desiredCloudConfig.Name) {
			cloudConfigsToCreate = append(cloudConfigsToCreate, desiredCloudConfig)
		}
	}

	return cloudConfigsToCreate, nil
}

// GetDeleteState returns an empty cloud configs collection. Cloud configs
// and all other resources are deleted when the Resource Group is deleted.
func (r *Resource) GetDeleteState(obj, currentState, desiredState interface{}) (interface{}, error) {
	return []CloudConfigBlob{}, nil
}

// GetUpdateState is not yet implemented.
func (r *Resource) GetUpdateState(obj, currentState, desiredState interface{}) (interface{}, interface{}, interface{}, error) {
	return []CloudConfigBlob{}, []CloudConfigBlob{}, []CloudConfigBlob{}, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}

// TODO ProcessCreateState is not yet implemented.
func (r *Resource) ProcessCreateState(obj, createState interface{}) error {
	return nil
}

// ProcessDeleteState returns nil because cloud configs are deleted when the
// resource group is deleted.
func (r *Resource) ProcessDeleteState(obj, deleteState interface{}) error {
	return nil
}

// ProcessUpdateState is not yet implemented.
func (r *Resource) ProcessUpdateState(obj, updateState interface{}) error {
	return nil
}

// Underlying returns the underlying resource.
func (r *Resource) Underlying() framework.Resource {
	return r
}

func existsCloudConfigByName(list []CloudConfigBlob, name string) bool {
	for _, c := range list {
		if c.Name == name {
			return true
		}
	}

	return false
}

func toCustomObject(v interface{}) (azuretpr.CustomObject, error) {
	if v == nil {
		return azuretpr.CustomObject{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &azuretpr.CustomObject{}, v)
	}

	customObjectPointer, ok := v.(*azuretpr.CustomObject)
	if !ok {
		return azuretpr.CustomObject{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &azuretpr.CustomObject{}, v)
	}
	customObject := *customObjectPointer

	return customObject, nil
}

func toCloudConfigBlobs(v interface{}) ([]CloudConfigBlob, error) {
	if v == nil {
		return nil, nil
	}

	cloudConfigBlobs, ok := v.([]CloudConfigBlob)
	if !ok {
		return nil, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", []CloudConfigBlob{}, v)
	}

	return cloudConfigBlobs, nil
}
