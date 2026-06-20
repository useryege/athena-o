package polymarket

import (
	"context"
	"strings"

	"github.com/useryege/athena/internal/polymarket/apiclient"
	polymarketstore "github.com/useryege/athena/internal/polymarket/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Service) GetPolymarketFIFAEventConfig(ctx context.Context, _ *apiclient.GetPolymarketFIFAEventConfigRequest) (*apiclient.GetPolymarketFIFAEventConfigResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket store is required")
	}
	config, err := s.store.GetPolymarketFIFAEventConfig(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get polymarket FIFA event config: %v", err)
	}
	if config == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket FIFA event config is not initialized")
	}
	return &apiclient.GetPolymarketFIFAEventConfigResponse{Config: fifaEventConfigItem(config)}, nil
}

func (s *Service) UpdatePolymarketFIFAEventConfig(ctx context.Context, req *apiclient.UpdatePolymarketFIFAEventConfigRequest) (*apiclient.UpdatePolymarketFIFAEventConfigResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket store is required")
	}
	if req == nil || req.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "config is required")
	}
	wormEventID := strings.TrimSpace(req.Config.WormEventID)
	if wormEventID == "" {
		return nil, status.Error(codes.InvalidArgument, "worm_event_id is required")
	}
	eventRef := strings.TrimSpace(req.Config.EventRef)
	if eventRef == "" {
		return nil, status.Error(codes.InvalidArgument, "event_ref is required")
	}
	config, err := s.store.UpdatePolymarketFIFAEventConfig(ctx, polymarketstore.FIFAEventConfig{
		WormEventID: wormEventID,
		EventRef:    eventRef,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update polymarket FIFA event config: %v", err)
	}
	if config == nil {
		return nil, status.Error(codes.FailedPrecondition, "polymarket FIFA event config is not initialized")
	}
	return &apiclient.UpdatePolymarketFIFAEventConfigResponse{Config: fifaEventConfigItem(config)}, nil
}

func fifaEventConfigItem(config *polymarketstore.FIFAEventConfig) *v1alpha1.PolymarketFIFAEventConfig {
	return &v1alpha1.PolymarketFIFAEventConfig{
		WormEventID: config.WormEventID,
		EventRef:    config.EventRef,
	}
}
