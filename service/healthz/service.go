package healthz

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
)

const (
	// topResultCount is how many results to return.
	topResultCount = 1
)

// Config represents the configuration used to create a healthz service.
type Config struct {
	AzureConfig client.AzureConfig
	Logger      micrologger.Logger
}

// Service implements the healthz service interface.
type Service struct {
	azureConfig client.AzureConfig
	logger      micrologger.Logger
}

// New creates a new configured healthz service.
func New(config Config) (*Service, error) {
	if err := config.AzureConfig.Validate(); err != nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.AzureConfig.%s", config, err)
	}
	if config.Logger == nil {
		return nil, microerror.Maskf(invalidConfigError, "%T.Logger must not be empty", config)
	}

	s := &Service{
		azureConfig: config.AzureConfig,
		logger:      config.Logger,
	}

	return s, nil
}

// Check implements the health check which lists the first Resource Group
// in the subscriptiontion to check we can authenticate.
func (s *Service) Check(ctx context.Context, request Request) (*Response, error) {
	azureClients, err := client.NewAzureClientSet(s.azureConfig)
	if err != nil {
		return nil, microerror.Mask(err)
	}

	_, err = azureClients.GroupsClient.List("", to.Int32Ptr(topResultCount))
	if err != nil {
		return nil, microerror.Mask(err)
	}

	return DefaultResponse(), nil
}
