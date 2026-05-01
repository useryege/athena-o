package projectsync

import (
	"context"
	"sync"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/klog/v2"
)

type Server struct {
	started bool
	stopped bool
	cancel  context.CancelFunc
	doneCh  chan struct{}
	mu      sync.Mutex
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) RunWithContext(ctx context.Context) {
	// Recover from panic and log with the configured logger.
	defer utilruntime.HandleCrashWithContext(ctx)
	logger := klog.FromContext(ctx)

	// TODO: add projectsync main loop.
	logger.Info("projectsync server started")
	<-ctx.Done()
	logger.Info("projectsync server stopped")
}

func (s *Server) HasStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.started
}

func (s *Server) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}

	runCtx, cancel := context.WithCancel(context.Background())
	doneCh := make(chan struct{})
	s.cancel = cancel
	s.doneCh = doneCh
	s.started = true
	s.stopped = false
	s.mu.Unlock()

	go func() {
		defer close(doneCh)
		defer func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.started = false
			s.stopped = true
			s.cancel = nil
		}()
		s.RunWithContext(runCtx)
	}()

	return nil
}

func (s *Server) Stop() error {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return nil
	}

	cancel := s.cancel
	doneCh := s.doneCh
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if doneCh != nil {
		<-doneCh
	}

	return nil
}
