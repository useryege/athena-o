package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server exposes prometheus metrics for block-sniffer.
type Server struct {
	handler  http.Handler
	registry *prometheus.Registry

	latestBlockHeight    *prometheus.GaugeVec
	blockProcessDuration *prometheus.HistogramVec
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
	s.latestBlockHeight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "athena_block_sniffer_latest_block_height",
			Help: "Latest on-chain block height observed by block sniffer.",
		},
		[]string{"chain_id"},
	)
	s.blockProcessDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "athena_block_sniffer_block_process_duration_seconds",
			Help:    "Duration to process one block in block sniffer hot path.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2},
		},
		[]string{"chain_id"},
	)

	s.registry.MustRegister(
		s.latestBlockHeight,
		s.blockProcessDuration,
	)
}

func (s *Server) SetLatestBlockHeight(chainID string, height uint64) {
	s.latestBlockHeight.WithLabelValues(chainID).Set(float64(height))
}

func (s *Server) ObserveBlockProcessDuration(chainID string, d time.Duration) {
	s.blockProcessDuration.WithLabelValues(chainID).Observe(d.Seconds())
}
