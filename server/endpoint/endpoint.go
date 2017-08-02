package endpoint

import (
	"github.com/giantswarm/micrologger"

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
	newEndpoint := &Endpoint{}
	return newEndpoint, nil
}

// Endpoint is the endpoint collection.
type Endpoint struct {
	// TODO Add endpoints
	// Healthz *healthz.Endpoint
	// Version *version.Endpoint
}
