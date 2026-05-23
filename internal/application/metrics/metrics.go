package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server exposes prometheus metrics for project-controller.
type Server struct {
	handler  http.Handler
	registry *prometheus.Registry

	// projectCount       *prometheus.GaugeVec
	// reconcileDurationSeconds *prometheus.HistogramVec
}

// NewMetricsServer creates a metrics server skeleton.
func NewMetricsServer() *Server {
	registry := prometheus.NewRegistry()

	srv := &Server{
		handler: promhttp.HandlerFor(prometheus.Gatherers{
			registry,
			prometheus.DefaultGatherer,
		}, promhttp.HandlerOpts{}),
		registry: registry,
	}

	srv.registerBusinessMetrics()

	return srv
}

// GetHandler returns the prometheus HTTP handler.
func (s *Server) GetHandler() http.Handler {
	return s.handler
}

func (s *Server) registerBusinessMetrics() {
	// s.projectCount = prometheus.NewGaugeVec(
	// 	prometheus.GaugeOpts{
	// 		Name: "athena_project_controller_active_project_count",
	// 		Help: "Current number of projects tracked by project controller.",
	// 	},
	// 	[]string{"chain_id"},
	// )

	// s.reconcileDurationSeconds = prometheus.NewHistogramVec(
	// 	prometheus.HistogramOpts{
	// 		Name:    "athena_project_controller_reconcile_duration_seconds",
	// 		Help:    "Duration for one reconcile cycle in project controller hot path.",
	// 		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
	// 	},
	// 	[]string{"chain_id"},
	// )

	// s.registry.MustRegister(
	// 	s.projectCount,
	// 	s.reconcileDurationSeconds,
	// )
}

// func (s *Server) SetActiveProjectCount(chainID string, count uint64) {
// 	s.projectCount.WithLabelValues(chainID).Set(float64(count))
// }

// func (s *Server) ObserveReconcileDuration(chainID string, d time.Duration) {
// 	s.reconcileDurationSeconds.WithLabelValues(chainID).Observe(d.Seconds())
// }
