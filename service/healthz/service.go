package healthz

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	healthzservice "github.com/giantswarm/microendpoint/service/healthz"
	"github.com/giantswarm/microerror"
	"github.com/giantswarm/micrologger"

	"github.com/giantswarm/azure-operator/client"
)

const (
	// Description describes which functionality this health check implements.
	Description = "Ensure Azure API availability."
	// Name is the identifier of the health check. This can be used for emitting
	// metrics.
	Name = "azure"
	// SuccessMessage is the message returned in case the health check did not
	// fail.
	SuccessMessage = "all good"
	// Timeout is the time being waited until timing out health check, which
	// renders its result unsuccessful.
	Timeout = 5 * time.Second

	// TopResultCount is how many results to return.
	TopResultCount = 1
)

const ()

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

// GetHealthz implements the health check for Azure. It does this by calling
// Resource Groups API and getting the first Resource Group. This checks that
// we can connect to the API and the credentials are correct.
func (s *Service) GetHealthz(ctx context.Context) (healthzservice.Response, error) {
	failed := false
	message := SuccessMessage
	{
		ch := make(chan string, 1)

		go func() {
			azureClients, err := client.NewAzureClientSet(s.azureConfig)
			if err != nil {
				ch <- err.Error()
				return
			}

			_, err = azureClients.GroupsClient.List(ctx, "", to.Int32Ptr(TopResultCount))
			if err != nil {
				ch <- err.Error()
				return
			}

			ch <- ""
		}()

		select {
		case m := <-ch:
			if m != "" {
				failed = true
				message = m
			}
		case <-time.After(Timeout):
			failed = true
			message = fmt.Sprintf("timed out after %s", Timeout)
		}
	}

	response := healthzservice.Response{
		Description: Description,
		Failed:      failed,
		Message:     message,
		Name:        Name,
	}

	return response, nil
}
