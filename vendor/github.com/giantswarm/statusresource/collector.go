package statusresource

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/api/meta"
)

var (
	clusterStatusDescription *prometheus.Desc = prometheus.NewDesc(
		prometheus.BuildFQName("statusresource", "cluster", "status"),
		"Cluster status condition as provided by the CR status.",
		[]string{
			"cluster_id",
			"status",
		},
		nil,
	)
)

func (r *Resource) Collect(ch chan<- prometheus.Metric) {
	r.logger.Log("level", "debug", "message", "start collecting metrics")

	watcher, err := r.restClient.Get().Watch()
	if err != nil {
		r.logger.Log("level", "error", "message", "watching CRs failed", "stack", fmt.Sprintf("%#v", err))
		return
	}
	defer watcher.Stop()

	for {
		select {
		case event, ok := <-watcher.ResultChan():
			if !ok {
				continue
			}

			fmt.Printf("\n")
			fmt.Printf("%#v\n", event.Object)
			fmt.Printf("\n")

			m, err := meta.Accessor(event.Object)
			if err != nil {
				r.logger.Log("level", "error", "message", "getting meta accessor failed", "stack", fmt.Sprintf("%#v", err))
				break
			}
			s, err := r.clusterStatusFunc(event.Object)
			if err != nil {
				r.logger.Log("level", "error", "message", "getting cluster status failed", "stack", fmt.Sprintf("%#v", err))
				break
			}

			ch <- prometheus.MustNewConstMetric(
				clusterStatusDescription,
				prometheus.GaugeValue,
				float64(boolToInt(s.HasCreatingCondition())),
				m.GetName(),
				"Creating",
			)
			ch <- prometheus.MustNewConstMetric(
				clusterStatusDescription,
				prometheus.GaugeValue,
				float64(boolToInt(s.HasCreatedCondition())),
				m.GetName(),
				"Created",
			)
			ch <- prometheus.MustNewConstMetric(
				clusterStatusDescription,
				prometheus.GaugeValue,
				float64(boolToInt(s.HasUpdatingCondition())),
				m.GetName(),
				"Updating",
			)
			ch <- prometheus.MustNewConstMetric(
				clusterStatusDescription,
				prometheus.GaugeValue,
				float64(boolToInt(s.HasUpdatedCondition())),
				m.GetName(),
				"Updated",
			)
			ch <- prometheus.MustNewConstMetric(
				clusterStatusDescription,
				prometheus.GaugeValue,
				float64(boolToInt(s.HasDeletingCondition())),
				m.GetName(),
				"Deleting",
			)
		case <-time.After(time.Second):
			r.logger.Log("level", "debug", "message", "finished collecting metrics")
			return
		}
	}
}

func (r *Resource) Describe(ch chan<- *prometheus.Desc) {
	ch <- clusterStatusDescription
}

func boolToInt(b bool) int {
	if b {
		return 1
	}

	return 0
}
