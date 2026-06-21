package fifa

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	polymarketapiclient "github.com/useryege/athena/internal/polymarket/apiclient"
	wormapiclient "github.com/useryege/athena/internal/worm/apiclient"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const refreshInterval = time.Second

var errCacheNotReady = errors.New("FIFA event cache is not ready")

// Cache refreshes and stores the currently configured FIFA event data for the
// public Worm and Polymarket APIs.
type Cache struct {
	wormClientSet       wormapiclient.Clientset
	polymarketClientSet polymarketapiclient.Clientset
	refreshCh           chan struct{}

	mu        sync.RWMutex
	revision  uint64
	config    *v1alpha1.PolymarketFIFAEventConfig
	configErr error

	wormItem      *v1alpha1.WormEventItem
	wormFetchedAt int64
	wormErr       error

	polymarketItem      *v1alpha1.PolymarketFIFAMoneylineEventItem
	polymarketFetchedAt int64
	polymarketErr       error
}

type clients struct {
	worm       wormapiclient.WormServiceClient
	wormCloser utilio.Closer

	polymarket       polymarketapiclient.PolymarketServiceClient
	polymarketCloser utilio.Closer
}

func NewCache(wormClientSet wormapiclient.Clientset, polymarketClientSet polymarketapiclient.Clientset) *Cache {
	return &Cache{
		wormClientSet:       wormClientSet,
		polymarketClientSet: polymarketClientSet,
		refreshCh:           make(chan struct{}, 1),
		configErr:           errCacheNotReady,
		wormErr:             errCacheNotReady,
		polymarketErr:       errCacheNotReady,
	}
}

// Run refreshes once immediately and then at most once for each one-second
// interval. Refresh signals that arrive during a refresh are coalesced.
func (c *Cache) Run(ctx context.Context) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	clients := &clients{}
	defer clients.close()

	for {
		c.refresh(ctx, clients)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-c.refreshCh:
		}

	drainSignals:
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			case <-c.refreshCh:
			default:
				break drainSignals
			}
		}
	}
}

// NotifyConfig synchronously switches the accepted public event identifiers
// and schedules an immediate background refresh.
func (c *Cache) NotifyConfig(config *v1alpha1.PolymarketFIFAEventConfig) {
	if normalized, ok := normalizeConfig(config); ok {
		c.applyConfig(normalized)
	} else {
		c.failConfig(c.currentRevision(), errors.New("FIFA event config is invalid"))
	}
	c.triggerRefresh()
}

func (c *Cache) GetWormEvent(conditionID string) (*v1alpha1.WormEventItem, int64, error) {
	conditionID = strings.TrimSpace(conditionID)
	if conditionID == "" {
		return nil, 0, status.Error(codes.InvalidArgument, "condition_id is required")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.configErr != nil || c.config == nil {
		return nil, 0, unavailableError("FIFA event config cache", c.configErr)
	}
	if conditionID != c.config.WormEventID {
		return nil, 0, status.Errorf(codes.InvalidArgument, "worm event %q is not the current configured FIFA event", conditionID)
	}
	if c.wormErr != nil || c.wormItem == nil {
		return nil, 0, unavailableError("Worm FIFA event cache", c.wormErr)
	}
	return cloneWormEvent(c.wormItem), c.wormFetchedAt, nil
}

func (c *Cache) GetPolymarketEvent(eventRef string) (*v1alpha1.PolymarketFIFAMoneylineEventItem, int64, error) {
	eventRef = strings.TrimSpace(eventRef)
	if eventRef == "" {
		return nil, 0, status.Error(codes.InvalidArgument, "event_ref is required")
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.configErr != nil || c.config == nil {
		return nil, 0, unavailableError("FIFA event config cache", c.configErr)
	}
	if eventRef != c.config.EventRef {
		return nil, 0, status.Errorf(codes.InvalidArgument, "polymarket event %q is not the current configured FIFA event", eventRef)
	}
	if c.polymarketErr != nil || c.polymarketItem == nil {
		return nil, 0, unavailableError("Polymarket FIFA event cache", c.polymarketErr)
	}
	return clonePolymarketEvent(c.polymarketItem), c.polymarketFetchedAt, nil
}

func (c *Cache) refresh(ctx context.Context, clients *clients) {
	revision := c.currentRevision()
	polymarketClient, err := c.ensurePolymarketClient(clients)
	if err != nil {
		c.failConfig(revision, err)
		return
	}

	configResp, err := polymarketClient.GetPolymarketFIFAEventConfig(ctx, &polymarketapiclient.GetPolymarketFIFAEventConfigRequest{})
	if c.currentRevision() != revision {
		return
	}
	if err != nil {
		c.failConfig(revision, err)
		return
	}
	config, ok := normalizeConfig(configResp.GetConfig())
	if !ok {
		c.failConfig(revision, errors.New("FIFA event config is empty"))
		return
	}
	c.applyConfig(config)

	wormClient, wormClientErr := c.ensureWormClient(clients)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if wormClientErr != nil {
			c.storeWormFailure(config.WormEventID, wormClientErr)
			return
		}
		resp, err := wormClient.GetWormEvent(ctx, &wormapiclient.GetWormEventRequest{ConditionId: config.WormEventID})
		if err != nil {
			c.storeWormFailure(config.WormEventID, err)
			return
		}
		c.storeWormSuccess(config.WormEventID, resp.GetEvent(), resp.GetFetchedAt())
	}()
	go func() {
		defer wg.Done()
		resp, err := polymarketClient.GetPolymarketFIFAMoneylineEvent(ctx, &polymarketapiclient.GetPolymarketFIFAMoneylineEventRequest{EventRef: config.EventRef})
		if err != nil {
			c.storePolymarketFailure(config.EventRef, err)
			return
		}
		c.storePolymarketSuccess(config.EventRef, resp.GetItem(), resp.GetFetchedAt())
	}()
	wg.Wait()
}

func (c *Cache) ensureWormClient(clients *clients) (wormapiclient.WormServiceClient, error) {
	if clients.worm != nil {
		return clients.worm, nil
	}
	closer, client, err := c.wormClientSet.NewWormServiceClient()
	if err != nil {
		return nil, err
	}
	clients.wormCloser = closer
	clients.worm = client
	return client, nil
}

func (c *Cache) ensurePolymarketClient(clients *clients) (polymarketapiclient.PolymarketServiceClient, error) {
	if clients.polymarket != nil {
		return clients.polymarket, nil
	}
	closer, client, err := c.polymarketClientSet.NewPolymarketServiceClient()
	if err != nil {
		return nil, err
	}
	clients.polymarketCloser = closer
	clients.polymarket = client
	return client, nil
}

func (c *Cache) applyConfig(config *v1alpha1.PolymarketFIFAEventConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	changed := c.config == nil || c.config.WormEventID != config.WormEventID || c.config.EventRef != config.EventRef
	if changed {
		c.revision++
		c.wormItem = nil
		c.wormFetchedAt = 0
		c.wormErr = errCacheNotReady
		c.polymarketItem = nil
		c.polymarketFetchedAt = 0
		c.polymarketErr = errCacheNotReady
	}
	c.config = cloneConfig(config)
	c.configErr = nil
}

func (c *Cache) failConfig(revision uint64, err error) {
	c.mu.Lock()
	if c.revision != revision {
		c.mu.Unlock()
		return
	}
	changed := errorText(c.configErr) != errorText(err)
	c.configErr = err
	c.wormItem = nil
	c.wormFetchedAt = 0
	c.wormErr = err
	c.polymarketItem = nil
	c.polymarketFetchedAt = 0
	c.polymarketErr = err
	c.mu.Unlock()
	if changed {
		log.WithError(err).Warn("FIFA event config cache refresh failed")
	}
}

func (c *Cache) storeWormSuccess(conditionID string, item *v1alpha1.WormEventItem, fetchedAt int64) {
	if item == nil {
		c.storeWormFailure(conditionID, errors.New("Worm FIFA event response is empty"))
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.configErr != nil || c.config == nil || c.config.WormEventID != conditionID {
		return
	}
	c.wormItem = cloneWormEvent(item)
	c.wormFetchedAt = fetchedAt
	c.wormErr = nil
}

func (c *Cache) storeWormFailure(conditionID string, err error) {
	c.mu.Lock()
	if c.configErr != nil || c.config == nil || c.config.WormEventID != conditionID {
		c.mu.Unlock()
		return
	}
	changed := errorText(c.wormErr) != errorText(err)
	c.wormItem = nil
	c.wormFetchedAt = 0
	c.wormErr = err
	c.mu.Unlock()
	if changed {
		log.WithError(err).Warn("Worm FIFA event cache refresh failed")
	}
}

func (c *Cache) storePolymarketSuccess(eventRef string, item *v1alpha1.PolymarketFIFAMoneylineEventItem, fetchedAt int64) {
	if item == nil {
		c.storePolymarketFailure(eventRef, errors.New("Polymarket FIFA event response is empty"))
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.configErr != nil || c.config == nil || c.config.EventRef != eventRef {
		return
	}
	c.polymarketItem = clonePolymarketEvent(item)
	c.polymarketFetchedAt = fetchedAt
	c.polymarketErr = nil
}

func (c *Cache) storePolymarketFailure(eventRef string, err error) {
	c.mu.Lock()
	if c.configErr != nil || c.config == nil || c.config.EventRef != eventRef {
		c.mu.Unlock()
		return
	}
	changed := errorText(c.polymarketErr) != errorText(err)
	c.polymarketItem = nil
	c.polymarketFetchedAt = 0
	c.polymarketErr = err
	c.mu.Unlock()
	if changed {
		log.WithError(err).Warn("Polymarket FIFA event cache refresh failed")
	}
}

func (c *Cache) currentRevision() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.revision
}

func (c *Cache) triggerRefresh() {
	select {
	case c.refreshCh <- struct{}{}:
	default:
	}
}

func (c *clients) close() {
	if c.wormCloser != nil {
		utilio.Close(c.wormCloser)
	}
	if c.polymarketCloser != nil {
		utilio.Close(c.polymarketCloser)
	}
}

func normalizeConfig(config *v1alpha1.PolymarketFIFAEventConfig) (*v1alpha1.PolymarketFIFAEventConfig, bool) {
	if config == nil {
		return nil, false
	}
	normalized := &v1alpha1.PolymarketFIFAEventConfig{
		WormEventID: strings.TrimSpace(config.WormEventID),
		EventRef:    strings.TrimSpace(config.EventRef),
	}
	return normalized, normalized.WormEventID != "" && normalized.EventRef != ""
}

func unavailableError(name string, err error) error {
	if err == nil {
		err = errCacheNotReady
	}
	return status.Errorf(codes.Unavailable, "%s is unavailable: %v", name, err)
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func cloneConfig(config *v1alpha1.PolymarketFIFAEventConfig) *v1alpha1.PolymarketFIFAEventConfig {
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
	cloned := *item
	if item.Teams != nil {
		cloned.Teams = make([]*v1alpha1.PolymarketSportsLiveTeamItem, len(item.Teams))
		for i, team := range item.Teams {
			if team != nil {
				clonedTeam := *team
				cloned.Teams[i] = &clonedTeam
			}
		}
	}
	if item.Options != nil {
		cloned.Options = make([]*v1alpha1.PolymarketFIFAMoneylineOptionItem, len(item.Options))
		for i, option := range item.Options {
			if option != nil {
				clonedOption := *option
				cloned.Options[i] = &clonedOption
			}
		}
	}
	return &cloned
}
