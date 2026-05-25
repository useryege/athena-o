package worm

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"github.com/useryege/athena/internal/worm/apiclient"
	wormstore "github.com/useryege/athena/internal/worm/store"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultWormMarketsLimit = 20
	maxWormMarketsLimit     = 100

	wormMarketsCategory = "sports"
	wormMarketsSort     = "leverage"
)

type wormMarketClient interface {
	ListMarkets(context.Context, utilworm.ListMarketsOptions) (*utilworm.ListMarketsResponse, error)
}

type Service struct {
	apiclient.UnimplementedWormServiceServer
	store        *wormstore.SQLStore
	wormClient   wormMarketClient
	assetBaseURL string
	startStopMu  sync.Mutex
	started      bool
}

func NewService(store *wormstore.SQLStore, wormClient wormMarketClient, assetBaseURL string) *Service {
	assetBaseURL = strings.TrimSpace(assetBaseURL)
	if assetBaseURL == "" {
		assetBaseURL = utilworm.DefaultBaseURL
	}
	return &Service{store: store, wormClient: wormClient, assetBaseURL: assetBaseURL}
}

func (s *Service) Start() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	if s.started {
		return nil
	}
	if s.store == nil {
		return status.Error(codes.FailedPrecondition, "worm store is required")
	}
	if s.wormClient == nil {
		return status.Error(codes.FailedPrecondition, "worm API client is required")
	}
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = false
	return nil
}

func (s *Service) GetWormStatus(context.Context, *apiclient.GetWormStatusRequest) (*apiclient.GetWormStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &apiclient.GetWormStatusResponse{
		Started: started,
		Status:  statusText,
	}, nil
}

func (s *Service) ListWormMarkets(ctx context.Context, req *apiclient.ListWormMarketsRequest) (*apiclient.ListWormMarketsResponse, error) {
	if s.wormClient == nil {
		return nil, status.Error(codes.FailedPrecondition, "worm API client is required")
	}

	limit := defaultWormMarketsLimit
	cursor := ""
	if req != nil {
		if req.GetLimit() != 0 {
			limit = int(req.GetLimit())
		}
		cursor = req.GetCursor()
	}
	if limit < 1 || limit > maxWormMarketsLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit must be between 1 and %d", maxWormMarketsLimit)
	}

	markets, err := s.wormClient.ListMarkets(ctx, utilworm.ListMarketsOptions{
		PageOptions: utilworm.PageOptions{
			Limit:  limit,
			Cursor: cursor,
		},
		Category: wormMarketsCategory,
		Sort:     wormMarketsSort,
	})
	if err != nil {
		return nil, err
	}

	resp := &apiclient.ListWormMarketsResponse{}
	if markets.Meta.NextCursor != nil {
		resp.NextCursor = *markets.Meta.NextCursor
	}
	resp.Markets = make([]*apiclient.WormMarketSummary, 0, len(markets.Markets))
	for i := range markets.Markets {
		resp.Markets = append(resp.Markets, s.toAPIMarketSummary(markets.Markets[i]))
	}
	return resp, nil
}

func (s *Service) toAPIMarketSummary(market utilworm.MarketSummary) *apiclient.WormMarketSummary {
	item := &apiclient.WormMarketSummary{
		ConditionId:    market.ConditionID,
		Title:          market.Title,
		Description:    stringValue(market.Description),
		Logo:           s.normalizeAssetURL(stringValue(market.Logo)),
		LastTradePrice: stringValue(market.LastTradePrice),
		State:          market.State,
		Category:       market.Category,
		Created:        int64Value(market.Created),
		MarginEnabled:  market.MarginEnabled,
	}
	if market.Event != nil {
		item.EventTitle = market.Event.Title
		item.EventConditionId = market.Event.ConditionID
		item.EventLogo = s.normalizeAssetURL(stringValue(market.Event.Logo))
	}
	return item
}

func (s *Service) normalizeAssetURL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	assetURL, err := url.Parse(value)
	if err != nil || assetURL.IsAbs() {
		return value
	}
	baseURL, err := url.Parse(s.assetBaseURL)
	if err != nil || baseURL.Scheme == "" || baseURL.Host == "" {
		return value
	}
	return baseURL.ResolveReference(assetURL).String()
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func int64Value(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
