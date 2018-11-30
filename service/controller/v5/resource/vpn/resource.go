package vpn

import (
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/service/controller/setting"
	"github.com/giantswarm/azure-operator/service/controller/v5/debugger"
)

const (
	// Name is the identifier of the resource.
	Name = "vpnv5"
)

// Config contains information required by Resource.
type Config struct {
	Debugger *debugger.Debugger
	Logger   micrologger.Logger

	Azure setting.Azure
}

// Resource ensures Microsoft Virtual Network Gateways are running.
type Resource struct {
	debugger *debugger.Debugger
	logger   micrologger.Logger

	azure setting.Azure
}

// New validates Config and creates a new Resource with it.
func New(config Config) (*Resource, error) {
	if config.Debugger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Debugger must not be empty", config)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}
	if err := config.Azure.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Azure.%s", config, err)
	}

	r := &Resource{
		debugger: config.Debugger,
		logger:   config.Logger,

		azure: config.Azure,
	}

	return r, nil
}

// Name returns the resource name.
func (r *Resource) Name() string {
	return Name
}
