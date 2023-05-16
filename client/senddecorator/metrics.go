package senddecorator

import (
	"net/http"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/giantswarm/azure-operator/v8/service/collector"
)

const metricsNamespace = "azure_operator_azure_api"

func MetricsDecorator(name, subscriptionID string, metricsCollector collector.AzureAPIMetrics) autorest.SendDecorator {
	lowerName := strings.ToLower(name)

	totalCallsOpts := prometheus.Opts{Namespace: metricsNamespace, Name: "total_calls", Help: "Total number of API calls"}
	ratelimitedCallsOpts := prometheus.Opts{Namespace: metricsNamespace, Name: "ratelimited_calls", Help: "Total number of API calls ratelimited"}
	errorRespOpts := prometheus.Opts{Namespace: metricsNamespace, Name: "error_resp", Help: "Total number of API error responses"}
	callLatencyOpts := prometheus.Opts{Namespace: metricsNamespace, Name: "req_latency", Help: "API request latency"}

	labels := prometheus.Labels{
		"api_service":     lowerName,
		"subscription_id": subscriptionID,
	}

	var labelNames []string
	for k := range labels {
		labelNames = append(labelNames, k)
	}

	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			start := time.Now()

			// Pass the request to next SendDecorator.
			resp, err := s.Do(r)

			elapsed := time.Since(start)

			metricsCollector.GetCounterVec(totalCallsOpts, labelNames).With(labels).Inc()
			metricsCollector.GetHistogramVec(callLatencyOpts, labelNames).With(labels).Observe(elapsed.Seconds())

			if resp != nil && resp.StatusCode >= 400 {
				metricsCollector.GetCounterVec(errorRespOpts, labelNames).With(labels).Inc()

				if resp.StatusCode == 429 {
					metricsCollector.GetCounterVec(ratelimitedCallsOpts, labelNames).With(labels).Inc()
				}
			}

			return resp, err
		})
	}
}
