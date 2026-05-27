package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/useryege/athena/util/profile"
)

type Server struct {
	handler  http.Handler
	registry *prometheus.Registry
}

func NewMetricsServer() *Server {
	registry := prometheus.NewRegistry()
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(prometheus.Gatherers{
		registry,
		prometheus.DefaultGatherer,
	}, promhttp.HandlerOpts{}))
	profile.RegisterProfiler(mux)

	return &Server{
		handler:  mux,
		registry: registry,
	}
}

func (s *Server) GetHandler() http.Handler {
	return s.handler
}
