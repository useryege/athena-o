package wormpoly

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	wormapiclient "github.com/useryege/athena/internal/worm/apiclient"
	"github.com/useryege/athena/internal/wormpoly/apiclient"
	wormpolystore "github.com/useryege/athena/internal/wormpoly/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errCacheNotReady = errors.New("worm-poly FIFA dashboard cache is not ready")

func (s *Service) GetWormPolyFIFADashboard(ctx context.Context, req *apiclient.GetWormPolyFIFADashboardRequest) (*apiclient.GetWormPolyFIFADashboardResponse, error) {
	requester := ""
	if req != nil {
		requester = strings.TrimSpace(req.GetRequester())
	}

	dashboard := s.currentDashboard()
	holdings, holdingsFetchedAt, holdingsErr := s.dashboardWalletHoldings(ctx, requester)
	dashboard.WalletHoldings = holdings
	dashboard.WalletHoldingsFetchedAt = holdingsFetchedAt
	dashboard.WalletHoldingsError = errorText(holdingsErr)
	dashboard.FetchedAt = s.nowUnix()

	return &apiclient.GetWormPolyFIFADashboardResponse{Dashboard: dashboard}, nil
}

func (s *Service) UpdateWormPolyFIFAEventConfig(ctx context.Context, req *apiclient.UpdateWormPolyFIFAEventConfigRequest) (*apiclient.UpdateWormPolyFIFAEventConfigResponse, error) {
	if s.store == nil {
		return nil, status.Error(codes.FailedPrecondition, "worm-poly store is required")
	}
	if req == nil || req.Config == nil {
		return nil, status.Error(codes.InvalidArgument, "config is required")
	}
	config, ok := normalizeConfig(req.Config)
	if !ok {
		return nil, status.Error(codes.InvalidArgument, "worm_event_id and event_ref are required")
	}

	updated, err := s.store.UpdateWormPolyFIFAEventConfig(ctx, wormpolystore.FIFAEventConfig{
		WormEventID: config.WormEventID,
		EventRef:    config.EventRef,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "update worm-poly FIFA event config: %v", err)
	}
	if updated == nil {
		return nil, status.Error(codes.FailedPrecondition, "worm-poly FIFA event config is not initialized")
	}

	item := fifaEventConfigItem(updated)
	s.applyConfig(item)
	s.triggerRefresh()
	return &apiclient.UpdateWormPolyFIFAEventConfigResponse{Config: item}, nil
}

func (s *Service) runFIFADashboardRefreshLoop(ctx context.Context) {
	defer s.runWG.Done()

	interval := s.fifaDashboardRefreshInterval
	if interval <= 0 {
		interval = defaultFIFADashboardRefreshInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		s.refreshFIFADashboard(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-s.refreshCh:
		}

	drainSignals:
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			case <-s.refreshCh:
			default:
				break drainSignals
			}
		}
	}
}

func (s *Service) refreshFIFADashboard(ctx context.Context) {
	revision := s.currentRevision()
	if s.store == nil {
		s.failConfig(revision, errors.New("worm-poly store is required"))
		return
	}

	stored, err := s.store.GetWormPolyFIFAEventConfig(ctx)
	if s.currentRevision() != revision {
		return
	}
	if err != nil {
		s.failConfig(revision, err)
		return
	}
	config, ok := normalizeConfig(fifaEventConfigItem(stored))
	if !ok {
		s.failConfig(revision, errors.New("worm-poly FIFA event config is empty"))
		return
	}
	s.applyConfig(config)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		item, fetchedAt, err := s.fetchWormEvent(ctx, config.WormEventID)
		if err != nil {
			s.storeWormFailure(config.WormEventID, err)
			return
		}
		s.storeWormSuccess(config.WormEventID, item, fetchedAt)
	}()
	go func() {
		defer wg.Done()
		item, fetchedAt, err := s.getFIFAMoneylineEvent(ctx, config.EventRef)
		if err != nil {
			s.storePolymarketFailure(config.EventRef, err)
			return
		}
		s.storePolymarketSuccess(config.EventRef, item, fetchedAt)
	}()
	wg.Wait()
}

func (s *Service) fetchWormEvent(ctx context.Context, conditionID string) (*v1alpha1.WormEventItem, int64, error) {
	if s.wormClientset == nil {
		return nil, 0, status.Error(codes.FailedPrecondition, "worm clientset is required")
	}
	closer, client, err := s.wormClientset.NewWormServiceClient()
	if err != nil {
		return nil, 0, fmt.Errorf("create worm client: %w", err)
	}
	defer closer.Close()

	resp, err := client.GetWormEvent(ctx, &wormapiclient.GetWormEventRequest{ConditionId: conditionID})
	if err != nil {
		return nil, 0, err
	}
	return resp.GetEvent(), resp.GetFetchedAt(), nil
}

func (s *Service) currentDashboard() *v1alpha1.WormPolyFIFADashboard {
	s.cacheMu.RLock()
	config := cloneConfig(s.config)
	configErr := s.configErr
	wormItem := cloneWormEvent(s.wormItem)
	wormFetchedAt := s.wormFetchedAt
	wormErr := s.wormErr
	polymarketItem := clonePolymarketEvent(s.polymarketItem)
	polymarketFetchedAt := s.polymarketFetchedAt
	polymarketErr := s.polymarketErr
	s.cacheMu.RUnlock()

	walletBalances, walletBalancesFetchedAt := s.currentFIFAWalletBalances()
	return &v1alpha1.WormPolyFIFADashboard{
		Config:                  config,
		WormEvent:               wormItem,
		WormFetchedAt:           wormFetchedAt,
		WormError:               firstErrorText(configErr, wormErr),
		PolymarketEvent:         polymarketItem,
		PolymarketFetchedAt:     polymarketFetchedAt,
		PolymarketError:         firstErrorText(configErr, polymarketErr),
		WalletBalances:          walletBalances,
		WalletBalancesFetchedAt: walletBalancesFetchedAt,
		WalletBalancesError:     walletBalanceErrorText(walletBalances),
	}
}

func (s *Service) dashboardWalletHoldings(ctx context.Context, requester string) ([]*v1alpha1.PolymarketFIFAWalletHoldingItem, int64, error) {
	if requester == "" {
		return nil, 0, status.Error(codes.InvalidArgument, "requester is required")
	}

	items, fetchedAt, needsRefresh := s.currentFIFAWalletHoldings(requester)
	if needsRefresh && fetchedAt == 0 {
		result, err := s.loadFIFAWalletHoldings(ctx, requester)
		if err != nil {
			return items, fetchedAt, err
		}
		return result.items, result.fetchedAt, nil
	}
	if needsRefresh {
		s.refreshFIFAWalletHoldingsAsync(requester)
	}
	return items, fetchedAt, walletHoldingError(items)
}

func (s *Service) applyConfig(config *v1alpha1.WormPolyFIFAEventConfig) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	changed := s.config == nil || s.config.WormEventID != config.WormEventID || s.config.EventRef != config.EventRef
	if changed {
		s.revision++
		s.wormItem = nil
		s.wormFetchedAt = 0
		s.wormErr = errCacheNotReady
		s.polymarketItem = nil
		s.polymarketFetchedAt = 0
		s.polymarketErr = errCacheNotReady
	}
	s.config = cloneConfig(config)
	s.configErr = nil
}

func (s *Service) failConfig(revision uint64, err error) {
	s.cacheMu.Lock()
	if s.revision != revision {
		s.cacheMu.Unlock()
		return
	}
	changed := errorText(s.configErr) != errorText(err)
	s.configErr = err
	s.wormItem = nil
	s.wormFetchedAt = 0
	s.wormErr = err
	s.polymarketItem = nil
	s.polymarketFetchedAt = 0
	s.polymarketErr = err
	s.cacheMu.Unlock()
	if changed {
		log.WithError(err).Warn("worm-poly FIFA event config cache refresh failed")
	}
}

func (s *Service) storeWormSuccess(conditionID string, item *v1alpha1.WormEventItem, fetchedAt int64) {
	if item == nil {
		s.storeWormFailure(conditionID, errors.New("Worm FIFA event response is empty"))
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.configErr != nil || s.config == nil || s.config.WormEventID != conditionID {
		return
	}
	s.wormItem = cloneWormEvent(item)
	s.wormFetchedAt = fetchedAt
	s.wormErr = nil
}

func (s *Service) storeWormFailure(conditionID string, err error) {
	s.cacheMu.Lock()
	if s.configErr != nil || s.config == nil || s.config.WormEventID != conditionID {
		s.cacheMu.Unlock()
		return
	}
	changed := errorText(s.wormErr) != errorText(err)
	s.wormItem = nil
	s.wormFetchedAt = 0
	s.wormErr = err
	s.cacheMu.Unlock()
	if changed {
		log.WithError(err).Warn("worm-poly Worm event cache refresh failed")
	}
}

func (s *Service) storePolymarketSuccess(eventRef string, item *v1alpha1.PolymarketFIFAMoneylineEventItem, fetchedAt int64) {
	if item == nil {
		s.storePolymarketFailure(eventRef, errors.New("Polymarket FIFA event response is empty"))
		return
	}
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.configErr != nil || s.config == nil || s.config.EventRef != eventRef {
		return
	}
	s.polymarketItem = clonePolymarketEvent(item)
	s.polymarketFetchedAt = fetchedAt
	s.polymarketErr = nil
}

func (s *Service) storePolymarketFailure(eventRef string, err error) {
	s.cacheMu.Lock()
	if s.configErr != nil || s.config == nil || s.config.EventRef != eventRef {
		s.cacheMu.Unlock()
		return
	}
	changed := errorText(s.polymarketErr) != errorText(err)
	s.polymarketItem = nil
	s.polymarketFetchedAt = 0
	s.polymarketErr = err
	s.cacheMu.Unlock()
	if changed {
		log.WithError(err).Warn("worm-poly Polymarket event cache refresh failed")
	}
}

func (s *Service) currentRevision() uint64 {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return s.revision
}

func (s *Service) triggerRefresh() {
	select {
	case s.refreshCh <- struct{}{}:
	default:
	}
}

func normalizeConfig(config *v1alpha1.WormPolyFIFAEventConfig) (*v1alpha1.WormPolyFIFAEventConfig, bool) {
	if config == nil {
		return nil, false
	}
	normalized := &v1alpha1.WormPolyFIFAEventConfig{
		WormEventID: strings.TrimSpace(config.WormEventID),
		EventRef:    strings.TrimSpace(config.EventRef),
	}
	return normalized, normalized.WormEventID != "" && normalized.EventRef != ""
}

func fifaEventConfigItem(config *wormpolystore.FIFAEventConfig) *v1alpha1.WormPolyFIFAEventConfig {
	if config == nil {
		return nil
	}
	return &v1alpha1.WormPolyFIFAEventConfig{
		WormEventID: config.WormEventID,
		EventRef:    config.EventRef,
	}
}

func cloneConfig(config *v1alpha1.WormPolyFIFAEventConfig) *v1alpha1.WormPolyFIFAEventConfig {
	if config == nil {
		return nil
	}
	cloned := *config
	return &cloned
}

func cloneWormEvent(item *v1alpha1.WormEventItem) *v1alpha1.WormEventItem {
	if item == nil {
		return nil
	}
	return item.DeepCopy()
}

func clonePolymarketEvent(item *v1alpha1.PolymarketFIFAMoneylineEventItem) *v1alpha1.PolymarketFIFAMoneylineEventItem {
	if item == nil {
		return nil
	}
	return item.DeepCopy()
}

func firstErrorText(errors ...error) string {
	for _, err := range errors {
		if err != nil {
			return err.Error()
		}
	}
	return ""
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func walletBalanceErrorText(items []*v1alpha1.PolymarketFIFAWalletBalanceItem) string {
	var messages []string
	for _, item := range items {
		if item == nil || item.OK || strings.TrimSpace(item.ErrorMessage) == "" {
			continue
		}
		messages = append(messages, item.ErrorMessage)
	}
	return strings.Join(messages, "; ")
}

func walletHoldingError(items []*v1alpha1.PolymarketFIFAWalletHoldingItem) error {
	var messages []string
	for _, item := range items {
		if item == nil || item.OK || strings.TrimSpace(item.ErrorMessage) == "" {
			continue
		}
		messages = append(messages, item.ErrorMessage)
	}
	if len(messages) == 0 {
		return nil
	}
	return errors.New(strings.Join(messages, "; "))
}
