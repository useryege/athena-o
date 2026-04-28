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
	mu           sync.RWMutex
	evmNodeWsURL string
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) SetEvmNodeWsURL(_ context.Context, req *blocksnifferpkg.SetEvmNodeWsURLRequest) (*blocksnifferpkg.SetEvmNodeWsURLResponse, error) {
	raw := strings.TrimSpace(req.GetEvmNodeWsURL())
	if raw == "" {
		return nil, status.Error(codes.InvalidArgument, "evm_node_ws_url is required")
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid evm_node_ws_url")
	}

	s.mu.Lock()
	s.evmNodeWsURL = raw
	s.mu.Unlock()

	return &blocksnifferpkg.SetEvmNodeWsURLResponse{
		EvmNodeWsURL: raw,
	}, nil
}

func (s *Server) GetEvmNodeWsURL(_ context.Context, _ *blocksnifferpkg.GetEvmNodeWsURLRequest) (*blocksnifferpkg.GetEvmNodeWsURLResponse, error) {
	s.mu.RLock()
	current := s.evmNodeWsURL
	s.mu.RUnlock()

	return &blocksnifferpkg.GetEvmNodeWsURLResponse{
		EvmNodeWsURL: current,
	}, nil
}
