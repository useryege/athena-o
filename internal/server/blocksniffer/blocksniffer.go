package blocksniffer

import (
	"context"
	"net/url"
	"strings"
	"sync"

	blocksnifferpkg "github.com/useryege/athena/pkg/apiclient/blocksniffer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	mu          sync.RWMutex
	nodeGrpcURL string
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) SetNodeGrpcURL(_ context.Context, req *blocksnifferpkg.SetNodeGrpcURLRequest) (*blocksnifferpkg.SetNodeGrpcURLResponse, error) {
	raw := strings.TrimSpace(req.GetNodeGrpcUrl())
	if raw == "" {
		return nil, status.Error(codes.InvalidArgument, "node_grpc_url is required")
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid node_grpc_url")
	}

	s.mu.Lock()
	s.nodeGrpcURL = raw
	s.mu.Unlock()

	return &blocksnifferpkg.SetNodeGrpcURLResponse{
		NodeGrpcUrl: raw,
	}, nil
}

func (s *Server) GetNodeGrpcURL(_ context.Context, _ *blocksnifferpkg.GetNodeGrpcURLRequest) (*blocksnifferpkg.GetNodeGrpcURLResponse, error) {
	s.mu.RLock()
	current := s.nodeGrpcURL
	s.mu.RUnlock()

	return &blocksnifferpkg.GetNodeGrpcURLResponse{
		NodeGrpcUrl: current,
	}, nil
}
