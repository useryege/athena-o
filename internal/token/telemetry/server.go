package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	address string
	tracker *Tracker
	ping    func(context.Context) error

	mu       sync.Mutex
	listener net.Listener
	server   *http.Server
	done     chan struct{}
}

func NewServer(address string, tracker *Tracker, ping func(context.Context) error) *Server {
	return &Server{address: address, tracker: tracker, ping: ping}
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.server != nil {
		return nil
	}
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("listen token health server on %s: %w", s.address, err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.health)
	mux.HandleFunc("/readyz", s.ready)
	mux.Handle("/metrics", promhttp.HandlerFor(s.tracker.Registry(), promhttp.HandlerOpts{}))
	s.listener = listener
	s.server = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	s.done = make(chan struct{})
	go func(server *http.Server, done chan struct{}) {
		defer close(done)
		_ = server.Serve(listener)
	}(s.server, s.done)
	return nil
}

func (s *Server) Stop(ctx context.Context) error {
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

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	if !s.tracker.IsRunning() {
		http.Error(w, "not running", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}

func (s *Server) ready(w http.ResponseWriter, request *http.Request) {
	statuses, ready := s.tracker.ScopeStatuses(time.Now().UTC())
	var databaseError string
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
		Ready         bool          `json:"ready"`
		DatabaseError string        `json:"databaseError,omitempty"`
		Components    []ScopeStatus `json:"components"`
	}{Ready: ready, DatabaseError: databaseError, Components: statuses})
}
