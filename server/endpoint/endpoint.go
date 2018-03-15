package endpoint

import (
	versionendpoint "github.com/giantswarm/microendpoint/endpoint/version"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/server/endpoint/healthz"
	"github.com/giantswarm/azure-operator/server/middleware"
	"github.com/giantswarm/azure-operator/service"
)

// Config represents the configuration used to create a endpoint.
type Config struct {
	// Dependencies.
	Logger     micrologger.Logger
	Middleware *middleware.Middleware
	Service    *service.Service
}

// DefaultConfig provides a default configuration to create a new endpoint by
// best effort.
func DefaultConfig() Config {
	return Config{
		// Dependencies.
		Logger:     nil,
		Middleware: nil,
		Service:    nil,
	}
}

// New creates a new configured endpoint.
func New(config Config) (*Endpoint, error) {
	var err error

	var versionEndpoint *versionendpoint.Endpoint
	{
		versionConfig := versionendpoint.DefaultConfig()
		versionConfig.Logger = config.Logger
		versionConfig.Service = config.Service.Version
		versionEndpoint, err = versionendpoint.New(versionConfig)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	var healthzEndpoint *healthz.Endpoint
	{
		c := healthz.Config{
			Logger:     config.Logger,
			Middleware: config.Middleware,
			Service:    config.Service,
		}
		healthzEndpoint, err = healthz.New(c)
		if err != nil {
			return nil, microerror.Mask(err)
		}
	}

	newEndpoint := &Endpoint{
		Healthz: healthzEndpoint,
		Version: versionEndpoint,
	}
	return newEndpoint, nil
}

// Endpoint is the endpoint collection.
type Endpoint struct {
	Healthz *healthz.Endpoint
	Version *versionendpoint.Endpoint
}
