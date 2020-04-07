package senddecorator

import (
	"github.com/Azure/go-autorest/autorest"

	"github.com/giantswarm/azure-operator/pkg/gatekeeper"
)

// ConfigureClient accepts gatekeeper instance and configures given autorest
// Client instance with all local `autorest.SendDecorator` implementations in this
// package.
//
// Existing SendDecorators are preserved, but moved to end of slice.
func ConfigureClient(g *gatekeeper.Gatekeeper, c *autorest.Client) {
	c.SendDecorators = append([]autorest.SendDecorator{
		// NOTE: Order matters here since these decorators are executed in
		// order. See: https://godoc.org/github.com/Azure/go-autorest/autorest#Client
		RateLimitCircuitBreaker(g),
	}, c.SendDecorators...)
}
