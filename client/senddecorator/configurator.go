package senddecorator

import (
	"github.com/Azure/go-autorest/autorest"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/giantswarm/azure-operator/pkg/backpressure"
)

// ConfigureClient accepts backpressure and prometheus summary vector instances
// and configures given autorest Client instance with all local
// `autorest.SendDecorator` implementations in this package.
//
// Existing SendDecorators are preserved, but moved to end of slice.
func ConfigureClient(g *backpressure.Backpressure, s *prometheus.SummaryVec, c *autorest.Client) {
	c.SendDecorators = append([]autorest.SendDecorator{
		// NOTE: Order matters here since these decorators are executed in
		// order. See: https://godoc.org/github.com/Azure/go-autorest/autorest#Client
		MetricsCollector(s),
		RateLimitCircuitBreaker(g),
	}, c.SendDecorators...)
}
