package senddecorator

import (
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/giantswarm/microerror"

	"github.com/giantswarm/azure-operator/pkg/gatekeeper"
	"github.com/giantswarm/azure-operator/pkg/httputil"
)

const (
	// Default wait time in case server returns HTTP 429 Too Many Requests but
	// doesn't provide Retry-After header.
	defaultWaitAfterTooManyRequests = 6 * time.Minute
)

// RateLimitCircuitBreaker utilizes simple gatekeeper implementation to hold
// off from making any additional requests when server responds HTTP 429 Too
// Many Requests.
func RateLimitCircuitBreaker(g *gatekeeper.Gatekeeper) autorest.SendDecorator {
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			// Check if we can proceed with request. If not, short-circuit here.
			if !g.CanProceed() {
				return nil, microerror.Maskf(tooManyRequestsError, "retry after %q", g.RetryAfter())
			}

			// Pass the request to next SendDecorator.
			resp, err := s.Do(r)

			// Check if rate-limiting has kicked in and Gatekeeper needs to be
			// updated correspondingly.
			if resp.StatusCode == http.StatusTooManyRequests {
				retryAfter, err := httputil.ParseRetryAfter(resp)
				if err != nil {
					// In case parsing fails, it's ok to fall back on default delay.
					retryAfter = time.Now().UTC().Add(defaultWaitAfterTooManyRequests)
				}

				g.NotBefore(retryAfter)
				return nil, microerror.Maskf(tooManyRequestsError, "retry after %q", g.RetryAfter)
			}

			return resp, err
		})
	}
}
