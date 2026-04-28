package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server exposes prometheus metrics for node-scanner.
type Server struct {
	handler  http.Handler
	registry *prometheus.Registry

	lastScanHeight      *prometheus.GaugeVec
	scanProcessDuration *prometheus.HistogramVec
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
	s.lastScanHeight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "athena_node_scanner_last_scan_height",
			Help: "Latest block height scanned by node scanner.",
		},
		[]string{"chain_id"},
	)
	s.scanProcessDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "athena_node_scanner_scan_process_duration_seconds",
			Help:    "Duration to process one scan in node scanner hot path.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		},
		[]string{"chain_id"},
	)

	s.registry.MustRegister(
		s.lastScanHeight,
		s.scanProcessDuration,
	)
}

func (s *Server) SetLastScanHeight(chainID string, height uint64) {
	s.lastScanHeight.WithLabelValues(chainID).Set(float64(height))
}

func (s *Server) ObserveScanProcessDuration(chainID string, d time.Duration) {
	s.scanProcessDuration.WithLabelValues(chainID).Observe(d.Seconds())
}
