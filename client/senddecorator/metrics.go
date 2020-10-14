package senddecorator

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/giantswarm/azure-operator/v5/service/collector"
)

const metricsNamespace = "azure_api"

func MetricsDecorator(name, subscriptionID string, metricsCollector collector.AzureAPIMetrics) autorest.SendDecorator {
	lowerName := strings.ToLower(name)

	// XXX: Following is slightly ugly as code, but better for UX. I buy the tradeoff. - @tuommaki
	titleName := strings.ReplaceAll(lowerName, "_", " ")
	titleName = strings.Title(titleName)
	titleName = strings.ReplaceAll(titleName, "Dns", "DNS")
	titleName = strings.ReplaceAll(titleName, "Gw", "GW")
	titleName = strings.ReplaceAll(titleName, "Ip", "IP")
	titleName = strings.ReplaceAll(titleName, "Nat", "NAT")
	titleName = strings.ReplaceAll(titleName, "Skus", "SKUs")
	titleName = strings.ReplaceAll(titleName, "Vms", "VMs")

	totalCallsOpts := prometheus.Opts{Namespace: metricsNamespace, Name: fmt.Sprintf("%s_total_calls", lowerName), Help: fmt.Sprintf("Total number of %s API calls", titleName), ConstLabels: prometheus.Labels{"subscription_id": subscriptionID}}
	errorRespOpts := prometheus.Opts{Namespace: metricsNamespace, Name: fmt.Sprintf("%s_error_resp", lowerName), Help: fmt.Sprintf("Total number of %s API error responses", titleName), ConstLabels: prometheus.Labels{"subscription_id": subscriptionID}}
	callLatencyOpts := prometheus.Opts{Namespace: metricsNamespace, Name: fmt.Sprintf("%s_req_latency", lowerName), Help: fmt.Sprintf("%s API request latency", titleName), ConstLabels: prometheus.Labels{"subscription_id": subscriptionID}}

	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			start := time.Now()

			// Pass the request to next SendDecorator.
			resp, err := s.Do(r)

			elapsed := time.Since(start)

			metricsCollector.GetCounter(totalCallsOpts).Inc()
			metricsCollector.GetHistogram(callLatencyOpts).Observe(elapsed.Seconds())

			if resp != nil && resp.StatusCode >= 400 {
				metricsCollector.GetCounter(errorRespOpts).Inc()
			}

			return resp, err
		})
	}
}

func RateLimitedMetricsDecorator(name, subscriptionID string, metricsCollector collector.AzureAPIMetrics) autorest.SendDecorator {
	lowerName := strings.ToLower(name) + "_rate_limited"

	// XXX: Following is slightly ugly as code, but better for UX. I buy the tradeoff. - @tuommaki
	titleName := strings.ReplaceAll(lowerName, "_", " ")
	titleName = strings.Title(titleName)
	titleName = strings.ReplaceAll(titleName, "Dns", "DNS")
	titleName = strings.ReplaceAll(titleName, "Gw", "GW")
	titleName = strings.ReplaceAll(titleName, "Ip", "IP")
	titleName = strings.ReplaceAll(titleName, "Nat", "NAT")
	titleName = strings.ReplaceAll(titleName, "Skus", "SKUs")
	titleName = strings.ReplaceAll(titleName, "Vms", "VMs")

	rateLimitedOpts := prometheus.Opts{Namespace: metricsNamespace, Name: lowerName, Help: fmt.Sprintf("Total number of %s API calls", titleName), ConstLabels: prometheus.Labels{"subscription_id": subscriptionID}}

	return func(s autorest.Sender) autorest.Sender {
		return autorest.SenderFunc(func(r *http.Request) (*http.Response, error) {
			// Pass the request to next SendDecorator.
			resp, err := s.Do(r)

			if resp != nil && resp.StatusCode == 429 {
				metricsCollector.GetCounter(rateLimitedOpts).Inc()
			}

			return resp, err
		})
	}
}
