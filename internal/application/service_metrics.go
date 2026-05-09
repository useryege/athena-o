package application

import "github.com/prometheus/client_golang/prometheus"

var projectSnapshotLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
	Namespace: "athena",
	Name:      "project_snapshot_latency_ms",
	Help:      "Latency of project snapshot reads in milliseconds.",
	Buckets:   []float64{1, 2.5, 5, 10, 25, 50, 100, 250, 500, 1000},
})

func init() {
	prometheus.MustRegister(projectSnapshotLatency)
}
