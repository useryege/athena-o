package bscswap

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const readinessStaleAfter = 2 * time.Minute

type Metrics struct {
	registry           *prometheus.Registry
	finalizedBlock     prometheus.Gauge
	indexedBlock       prometheus.Gauge
	lagBlocks          prometheus.Gauge
	processedBlocks    prometheus.Counter
	matchingLogs       prometheus.Counter
	storedTransactions prometheus.Counter
	scanFailures       prometheus.Counter
	queryDuration      prometheus.Histogram

	running     atomic.Bool
	caughtUp    atomic.Bool
	lastSuccess atomic.Int64
}

func NewMetrics() *Metrics {
	metrics := &Metrics{
		registry: prometheus.NewRegistry(),
		finalizedBlock: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "athena_bsc_swap_finalized_block", Help: "Latest finalized BSC block observed by the swap scanner.",
		}),
		indexedBlock: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "athena_bsc_swap_indexed_block", Help: "Highest finalized BSC block committed by the swap scanner.",
		}),
		lagBlocks: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "athena_bsc_swap_lag_blocks", Help: "Finalized blocks not yet committed by the swap scanner.",
		}),
		processedBlocks: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "athena_bsc_swap_processed_blocks_total", Help: "Finalized BSC blocks committed by the swap scanner.",
		}),
		matchingLogs: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "athena_bsc_swap_matching_logs_total", Help: "BSC logs matching the configured V2 Swap topic.",
		}),
		storedTransactions: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "athena_bsc_swap_stored_transactions_total", Help: "Transactions containing at least one matching V2 Swap topic log.",
		}),
		scanFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "athena_bsc_swap_scan_failures_total", Help: "Failed BSC swap scanner iterations.",
		}),
		queryDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name: "athena_bsc_swap_query_duration_seconds", Help: "ListSwapTransactions request duration.",
		}),
	}
	metrics.registry.MustRegister(
		metrics.finalizedBlock, metrics.indexedBlock, metrics.lagBlocks,
		metrics.processedBlocks, metrics.matchingLogs, metrics.storedTransactions,
		metrics.scanFailures, metrics.queryDuration,
	)
	return metrics
}

func (m *Metrics) Registry() *prometheus.Registry { return m.registry }

func (m *Metrics) SetRunning(running bool) { m.running.Store(running) }

func (m *Metrics) RecordProgress(finalized, indexed uint64) {
	m.finalizedBlock.Set(float64(finalized))
	m.indexedBlock.Set(float64(indexed))
	lag := uint64(0)
	if finalized > indexed {
		lag = finalized - indexed
	}
	m.lagBlocks.Set(float64(lag))
	m.caughtUp.Store(indexed >= finalized)
}

func (m *Metrics) RecordScanSuccess(blocks, logs, transactions uint64) {
	m.processedBlocks.Add(float64(blocks))
	m.matchingLogs.Add(float64(logs))
	m.storedTransactions.Add(float64(transactions))
	m.lastSuccess.Store(time.Now().UTC().Unix())
}

func (m *Metrics) RecordScanFailure() { m.scanFailures.Inc() }

func (m *Metrics) ObserveQuery(start time.Time) { m.queryDuration.Observe(time.Since(start).Seconds()) }

func (m *Metrics) Running() bool { return m != nil && m.running.Load() }

func (m *Metrics) Ready(now time.Time) bool {
	if m == nil || !m.running.Load() || !m.caughtUp.Load() {
		return false
	}
	lastSuccess := m.lastSuccess.Load()
	return lastSuccess > 0 && now.Sub(time.Unix(lastSuccess, 0)) <= readinessStaleAfter
}

type TelemetryServer struct {
	address string
	metrics *Metrics
	ping    func(context.Context) error

	mu       sync.Mutex
	server   *http.Server
	listener net.Listener
	done     chan struct{}
}

func NewTelemetryServer(address string, metrics *Metrics, ping func(context.Context) error) *TelemetryServer {
	return &TelemetryServer{address: address, metrics: metrics, ping: ping}
}

func (s *TelemetryServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return nil
	}
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("listen BSC swap telemetry on %s: %w", s.address, err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/readyz", s.ready)
	mux.Handle("/metrics", promhttp.HandlerFor(s.metrics.Registry(), promhttp.HandlerOpts{}))
	s.listener = listener
	s.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	s.done = make(chan struct{})
	go func(server *http.Server, done chan struct{}) {
		defer close(done)
		_ = server.Serve(listener)
	}(s.server, s.done)
	return nil
}

func (s *TelemetryServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	server, done := s.server, s.done
	s.server = nil
	s.listener = nil
	s.done = nil
	s.mu.Unlock()
	if server == nil {
		return nil
	}
	err := server.Shutdown(ctx)
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			if err == nil {
				err = ctx.Err()
			}
		}
	}
	return err
}

func (s *TelemetryServer) health(w http.ResponseWriter, _ *http.Request) {
	if !s.metrics.Running() {
		http.Error(w, "not running", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *TelemetryServer) ready(w http.ResponseWriter, request *http.Request) {
	ready := s.metrics.Ready(time.Now().UTC())
	databaseError := ""
	if s.ping != nil {
		ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
		err := s.ping(ctx)
		cancel()
		if err != nil {
			ready = false
			databaseError = err.Error()
		}
	}
	w.Header().Set("Content-Type", "application/json")
	if !ready {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	_ = json.NewEncoder(w).Encode(struct {
		Ready         bool   `json:"ready"`
		DatabaseError string `json:"databaseError,omitempty"`
	}{Ready: ready, DatabaseError: databaseError})
}
