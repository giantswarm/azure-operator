package senddecorator

import (
	"net/http"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/prometheus/client_golang/prometheus"
)

// MetricsCollector returns decorator that collects API call metrics.
func MetricsCollector(m *prometheus.SummaryVec) autorest.SendDecorator {
	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {

			start := time.Now()

			// Pass the request to next SendDecorator.
			resp, err := s.Do(r)

			elapsed := time.Since(start)

			// Record latency as milliseconds.
			m.WithLabelValues(string(resp.StatusCode)).Observe(elapsed.Seconds() * 1000)

			return resp, err
		})
	}
}
