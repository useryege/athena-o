package worm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/worm/apiclient"
	utilworm "github.com/useryege/athena/util/worm"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultWormUpstreamLimitPerMinute = 1000
	defaultWormListDefaultFreshTTL    = 5 * time.Second
	defaultWormListFreshTTL           = 15 * time.Second
	defaultWormDetailFreshTTL         = 10 * time.Second
	defaultWormStaleTTL               = 15 * time.Minute
	defaultWormRefreshWorkers         = 2
	defaultWormActiveRefreshInterval  = time.Second
	defaultWormActiveRefreshTTL       = 3 * time.Second

	wormRefreshLockTTL       = 30 * time.Second
	wormRefreshLockWait      = 2 * time.Second
	wormRefreshLockPoll      = 100 * time.Millisecond
	wormDefaultCooldown      = 12 * time.Second
	wormHotKeyRetention      = 10 * time.Minute
	wormWarmupHotListLimit   = 25
	wormWarmupHotDetailLimit = 25
)

type CacheConfig struct {
	UpstreamLimitPerMinute int
	ListDefaultFreshTTL    time.Duration
	ListFreshTTL           time.Duration
	DetailFreshTTL         time.Duration
	StaleTTL               time.Duration
	RefreshWorkers         int
	ActiveRefreshInterval  time.Duration
	ActiveRefreshTTL       time.Duration
	Now                    func() time.Time
}

func DefaultCacheConfig() CacheConfig {
	return CacheConfig{}.withDefaults()
}

func (c CacheConfig) withDefaults() CacheConfig {
	if c.UpstreamLimitPerMinute <= 0 {
		c.UpstreamLimitPerMinute = defaultWormUpstreamLimitPerMinute
	}
	if c.ListDefaultFreshTTL <= 0 {
		c.ListDefaultFreshTTL = defaultWormListDefaultFreshTTL
	}
	if c.ListFreshTTL <= 0 {
		c.ListFreshTTL = defaultWormListFreshTTL
	}
	if c.DetailFreshTTL <= 0 {
		c.DetailFreshTTL = defaultWormDetailFreshTTL
	}
	if c.StaleTTL <= 0 {
		c.StaleTTL = defaultWormStaleTTL
	}
	if c.RefreshWorkers <= 0 {
		c.RefreshWorkers = defaultWormRefreshWorkers
	}
	if c.ActiveRefreshInterval <= 0 {
		c.ActiveRefreshInterval = defaultWormActiveRefreshInterval
	}
	if c.ActiveRefreshTTL <= 0 {
		c.ActiveRefreshTTL = defaultWormActiveRefreshTTL
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	return c
}

type listWormMarketsParams struct {
	Limit        int    `json:"limit"`
	Cursor       string `json:"cursor"`
	SortOption   string `json:"sort_option"`
	CategorySlug string `json:"category_slug"`
}

type cachedListWormMarketsResponse struct {
	Response   *apiclient.ListWormMarketsResponse `json:"response"`
	FetchedAt  int64                              `json:"fetched_at"`
	ExpiresAt  int64                              `json:"expires_at"`
	StaleUntil int64                              `json:"stale_until"`
}

type cachedGetWormMarketResponse struct {
	Response   *apiclient.GetWormMarketResponse `json:"response"`
	FetchedAt  int64                            `json:"fetched_at"`
	ExpiresAt  int64                            `json:"expires_at"`
	StaleUntil int64                            `json:"stale_until"`
}

type refreshKind string

const (
	refreshKindList   refreshKind = "list"
	refreshKindDetail refreshKind = "detail"
)

type refreshRequest struct {
	kind        refreshKind
	listParams  listWormMarketsParams
	conditionID string
}

func (s *Service) listWormMarketsCached(ctx context.Context, params listWormMarketsParams) (*apiclient.ListWormMarketsResponse, error) {
	_ = s.recordHotList(ctx, params)
	key := wormListCacheKey(params)
	now := s.cacheConfig.Now().Unix()
	entry, found, err := s.getCachedList(ctx, key)
	if err != nil {
		return nil, err
	}
	if found && now <= entry.ExpiresAt {
		return listResponseFromCache(entry, false), nil
	}
	if found && now <= entry.StaleUntil {
		s.enqueueRefresh(refreshRequest{kind: refreshKindList, listParams: params})
		return listResponseFromCache(entry, true), nil
	}

	value, err, _ := s.refreshGroup.Do(key, func() (any, error) {
		if next, ok, getErr := s.getCachedList(ctx, key); getErr != nil {
			return nil, getErr
		} else if ok && s.cacheConfig.Now().Unix() <= next.ExpiresAt {
			return listResponseFromCache(next, false), nil
		}

		unlock, locked, lockErr := s.acquireRefreshLock(ctx, key)
		if lockErr != nil {
			if found {
				return listResponseFromCache(entry, true), nil
			}
			return nil, lockErr
		}
		if !locked {
			if resp, ok := s.waitForCachedList(ctx, key); ok {
				return resp, nil
			}
			if found {
				return listResponseFromCache(entry, true), nil
			}
			return nil, status.Error(codes.Unavailable, "worm cache refresh is already in progress")
		}
		defer unlock()

		if limitErr := s.reserveUpstream(ctx, 1); limitErr != nil {
			if found {
				return listResponseFromCache(entry, true), nil
			}
			return nil, limitErr
		}
		resp, fetchErr := s.fetchWormMarkets(ctx, params)
		if fetchErr != nil {
			s.markUpstreamFailure(ctx, fetchErr)
			if found {
				return listResponseFromCache(entry, true), nil
			}
			return nil, fetchErr
		}
		if setErr := s.setCachedList(ctx, key, params, resp); setErr != nil {
			return nil, setErr
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*apiclient.ListWormMarketsResponse), nil
}

func (s *Service) getWormMarketCached(ctx context.Context, conditionID string) (*apiclient.GetWormMarketResponse, error) {
	_ = s.recordHotDetail(ctx, conditionID)
	key := wormDetailCacheKey(conditionID)
	now := s.cacheConfig.Now().Unix()
	entry, found, err := s.getCachedDetail(ctx, key)
	if err != nil {
		return nil, err
	}
	if found && now <= entry.ExpiresAt {
		return detailResponseFromCache(entry, false), nil
	}
	if found && now <= entry.StaleUntil {
		s.enqueueRefresh(refreshRequest{kind: refreshKindDetail, conditionID: conditionID})
		return detailResponseFromCache(entry, true), nil
	}

	value, err, _ := s.refreshGroup.Do(key, func() (any, error) {
		if next, ok, getErr := s.getCachedDetail(ctx, key); getErr != nil {
			return nil, getErr
		} else if ok && s.cacheConfig.Now().Unix() <= next.ExpiresAt {
			return detailResponseFromCache(next, false), nil
		}

		unlock, locked, lockErr := s.acquireRefreshLock(ctx, key)
		if lockErr != nil {
			if found {
				return detailResponseFromCache(entry, true), nil
			}
			return nil, lockErr
		}
		if !locked {
			if resp, ok := s.waitForCachedDetail(ctx, key); ok {
				return resp, nil
			}
			if found {
				return detailResponseFromCache(entry, true), nil
			}
			return nil, status.Error(codes.Unavailable, "worm cache refresh is already in progress")
		}
		defer unlock()

		if limitErr := s.reserveUpstream(ctx, 1); limitErr != nil {
			if found {
				return detailResponseFromCache(entry, true), nil
			}
			return nil, limitErr
		}
		resp, fetchErr := s.fetchWormMarket(ctx, conditionID)
		if fetchErr != nil {
			s.markUpstreamFailure(ctx, fetchErr)
			if found {
				return detailResponseFromCache(entry, true), nil
			}
			return nil, fetchErr
		}
		if setErr := s.setCachedDetail(ctx, key, resp); setErr != nil {
			return nil, setErr
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return value.(*apiclient.GetWormMarketResponse), nil
}

func (s *Service) refreshWorker(ctx context.Context) {
	defer s.refreshWG.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-s.refreshCh:
			s.refreshCachedValue(ctx, req)
		}
	}
}

func (s *Service) warmupLoop(ctx context.Context) {
	defer s.refreshWG.Done()
	defaultParams := listWormMarketsParams{
		Limit:        defaultWormMarketsLimit,
		SortOption:   defaultWormMarketsSortOption,
		CategorySlug: defaultWormMarketsCategorySlug,
	}
	s.enqueueRefresh(refreshRequest{kind: refreshKindList, listParams: defaultParams})
	ticker := time.NewTicker(s.cacheConfig.ActiveRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.enqueueRefresh(refreshRequest{kind: refreshKindList, listParams: defaultParams})
			s.enqueueHotRefreshes(ctx)
		}
	}
}

func (s *Service) refreshCachedValue(ctx context.Context, req refreshRequest) {
	switch req.kind {
	case refreshKindList:
		s.refreshCachedList(ctx, req.listParams)
	case refreshKindDetail:
		s.refreshCachedDetail(ctx, req.conditionID)
	}
}

func (s *Service) refreshCachedList(ctx context.Context, params listWormMarketsParams) {
	key := wormListCacheKey(params)
	unlock, locked, err := s.acquireRefreshLock(ctx, key)
	if err != nil || !locked {
		return
	}
	defer unlock()
	if err := s.reserveUpstream(ctx, 1); err != nil {
		return
	}
	resp, err := s.fetchWormMarkets(ctx, params)
	if err != nil {
		s.markUpstreamFailure(ctx, err)
		log.Debugf("failed to refresh worm list cache %s: %v", key, err)
		return
	}
	if err := s.setCachedList(ctx, key, params, resp); err != nil {
		log.Warnf("failed to store worm list cache %s: %v", key, err)
	}
}

func (s *Service) refreshCachedDetail(ctx context.Context, conditionID string) {
	key := wormDetailCacheKey(conditionID)
	unlock, locked, err := s.acquireRefreshLock(ctx, key)
	if err != nil || !locked {
		return
	}
	defer unlock()
	if err := s.reserveUpstream(ctx, 1); err != nil {
		return
	}
	resp, err := s.fetchWormMarket(ctx, conditionID)
	if err != nil {
		s.markUpstreamFailure(ctx, err)
		log.Debugf("failed to refresh worm detail cache %s: %v", key, err)
		return
	}
	if err := s.setCachedDetail(ctx, key, resp); err != nil {
		log.Warnf("failed to store worm detail cache %s: %v", key, err)
	}
}

func (s *Service) enqueueRefresh(req refreshRequest) {
	select {
	case s.refreshCh <- req:
	default:
		log.Debug("worm refresh queue is full; dropping refresh request")
	}
}

func (s *Service) enqueueHotRefreshes(ctx context.Context) {
	if s.redisClient == nil {
		return
	}
	now := s.cacheConfig.Now().Unix()
	retentionMinScore := strconv.FormatInt(now-int64(wormHotKeyRetention.Seconds()), 10)
	_ = s.redisClient.ZRemRangeByScore(ctx, wormHotListKey(), "-inf", retentionMinScore).Err()
	_ = s.redisClient.ZRemRangeByScore(ctx, wormHotDetailKey(), "-inf", retentionMinScore).Err()

	activeMinScore := strconv.FormatInt(now-int64(s.cacheConfig.ActiveRefreshTTL.Seconds()), 10)
	activeRange := &redis.ZRangeBy{
		Min:    activeMinScore,
		Max:    "+inf",
		Offset: 0,
		Count:  wormWarmupHotListLimit,
	}
	lists, err := s.redisClient.ZRevRangeByScore(ctx, wormHotListKey(), activeRange).Result()
	if err == nil {
		for _, member := range lists {
			var params listWormMarketsParams
			if json.Unmarshal([]byte(member), &params) == nil && !params.isDefaultFirstPage() {
				s.enqueueRefresh(refreshRequest{kind: refreshKindList, listParams: params})
			}
		}
	}
	activeRange.Count = wormWarmupHotDetailLimit
	details, err := s.redisClient.ZRevRangeByScore(ctx, wormHotDetailKey(), activeRange).Result()
	if err == nil {
		for _, conditionID := range details {
			if strings.TrimSpace(conditionID) != "" {
				s.enqueueRefresh(refreshRequest{kind: refreshKindDetail, conditionID: conditionID})
			}
		}
	}
}

func (s *Service) getCachedList(ctx context.Context, key string) (*cachedListWormMarketsResponse, bool, error) {
	var entry cachedListWormMarketsResponse
	found, err := s.getJSON(ctx, key, &entry)
	if err != nil || !found || entry.Response == nil {
		return nil, false, err
	}
	return &entry, true, nil
}

func (s *Service) getCachedDetail(ctx context.Context, key string) (*cachedGetWormMarketResponse, bool, error) {
	var entry cachedGetWormMarketResponse
	found, err := s.getJSON(ctx, key, &entry)
	if err != nil || !found || entry.Response == nil {
		return nil, false, err
	}
	return &entry, true, nil
}

func (s *Service) setCachedList(ctx context.Context, key string, params listWormMarketsParams, resp *apiclient.ListWormMarketsResponse) error {
	now := s.cacheConfig.Now()
	freshTTL := s.cacheConfig.ListFreshTTL
	if params.isDefaultFirstPage() {
		freshTTL = s.cacheConfig.ListDefaultFreshTTL
	}
	resp = cloneListResponse(resp)
	resp.FetchedAt = now.Unix()
	resp.Stale = false
	entry := cachedListWormMarketsResponse{
		Response:   resp,
		FetchedAt:  resp.FetchedAt,
		ExpiresAt:  now.Add(freshTTL).Unix(),
		StaleUntil: now.Add(s.cacheConfig.StaleTTL).Unix(),
	}
	return s.setJSON(ctx, key, entry, s.cacheConfig.StaleTTL)
}

func (s *Service) setCachedDetail(ctx context.Context, key string, resp *apiclient.GetWormMarketResponse) error {
	now := s.cacheConfig.Now()
	resp = cloneDetailResponse(resp)
	resp.FetchedAt = now.Unix()
	resp.Stale = false
	entry := cachedGetWormMarketResponse{
		Response:   resp,
		FetchedAt:  resp.FetchedAt,
		ExpiresAt:  now.Add(s.cacheConfig.DetailFreshTTL).Unix(),
		StaleUntil: now.Add(s.cacheConfig.StaleTTL).Unix(),
	}
	return s.setJSON(ctx, key, entry, s.cacheConfig.StaleTTL)
}

func (s *Service) waitForCachedList(ctx context.Context, key string) (*apiclient.ListWormMarketsResponse, bool) {
	deadline := s.cacheConfig.Now().Add(wormRefreshLockWait)
	for s.cacheConfig.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, false
		case <-time.After(wormRefreshLockPoll):
		}
		entry, found, err := s.getCachedList(ctx, key)
		if err == nil && found {
			return listResponseFromCache(entry, s.cacheConfig.Now().Unix() > entry.ExpiresAt), true
		}
	}
	return nil, false
}

func (s *Service) waitForCachedDetail(ctx context.Context, key string) (*apiclient.GetWormMarketResponse, bool) {
	deadline := s.cacheConfig.Now().Add(wormRefreshLockWait)
	for s.cacheConfig.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, false
		case <-time.After(wormRefreshLockPoll):
		}
		entry, found, err := s.getCachedDetail(ctx, key)
		if err == nil && found {
			return detailResponseFromCache(entry, s.cacheConfig.Now().Unix() > entry.ExpiresAt), true
		}
	}
	return nil, false
}

func (s *Service) getJSON(ctx context.Context, key string, out any) (bool, error) {
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if stderrors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, status.Errorf(codes.Unavailable, "failed to read worm cache: %v", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return false, status.Errorf(codes.Internal, "failed to decode worm cache: %v", err)
	}
	return true, nil
}

func (s *Service) setJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to encode worm cache: %v", err)
	}
	if err := s.redisClient.Set(ctx, key, data, ttl).Err(); err != nil {
		return status.Errorf(codes.Unavailable, "failed to write worm cache: %v", err)
	}
	return nil
}

func (s *Service) acquireRefreshLock(ctx context.Context, key string) (func(), bool, error) {
	lockKey := wormRefreshLockKey(key)
	ok, err := s.redisClient.SetNX(ctx, lockKey, "1", wormRefreshLockTTL).Result()
	if err != nil {
		return nil, false, status.Errorf(codes.Unavailable, "failed to acquire worm refresh lock: %v", err)
	}
	if !ok {
		return nil, false, nil
	}
	return func() {
		_ = s.redisClient.Del(context.Background(), lockKey).Err()
	}, true, nil
}

func (s *Service) reserveUpstream(ctx context.Context, units int64) error {
	if s.redisClient == nil || s.cacheConfig.UpstreamLimitPerMinute <= 0 {
		return nil
	}
	if units < 1 {
		units = 1
	}
	cooldown, err := s.redisClient.TTL(ctx, wormUpstreamCooldownKey()).Result()
	if err == nil && cooldown > 0 {
		return status.Errorf(codes.Unavailable, "worm upstream is cooling down for %s", cooldown.Truncate(time.Second))
	}
	bucket := s.cacheConfig.Now().Unix() / 60
	key := wormUpstreamMinuteKey(bucket)
	pipe := s.redisClient.TxPipeline()
	incr := pipe.IncrBy(ctx, key, units)
	pipe.Expire(ctx, key, 2*time.Minute)
	if _, err := pipe.Exec(ctx); err != nil {
		return status.Errorf(codes.Unavailable, "failed to reserve worm upstream budget: %v", err)
	}
	if incr.Val() > int64(s.cacheConfig.UpstreamLimitPerMinute) {
		_ = s.redisClient.DecrBy(ctx, key, units).Err()
		return status.Error(codes.Unavailable, "worm upstream rate limit budget exhausted")
	}
	return nil
}

func (s *Service) markUpstreamFailure(ctx context.Context, err error) {
	var wormErr *utilworm.Error
	if !stderrors.As(err, &wormErr) {
		return
	}
	if wormErr.StatusCode != 429 && wormErr.Slug != "throttled" {
		return
	}
	cooldown := wormErr.RetryAfter
	if cooldown <= 0 {
		cooldown = wormDefaultCooldown
	}
	_ = s.redisClient.Set(ctx, wormUpstreamCooldownKey(), "1", cooldown).Err()
}

func (s *Service) recordHotList(ctx context.Context, params listWormMarketsParams) error {
	member, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return s.redisClient.ZAdd(ctx, wormHotListKey(), redis.Z{
		Score:  float64(s.cacheConfig.Now().Unix()),
		Member: string(member),
	}).Err()
}

func (s *Service) recordHotDetail(ctx context.Context, conditionID string) error {
	return s.redisClient.ZAdd(ctx, wormHotDetailKey(), redis.Z{
		Score:  float64(s.cacheConfig.Now().Unix()),
		Member: conditionID,
	}).Err()
}

func listResponseFromCache(entry *cachedListWormMarketsResponse, stale bool) *apiclient.ListWormMarketsResponse {
	resp := cloneListResponse(entry.Response)
	resp.FetchedAt = entry.FetchedAt
	resp.Stale = stale
	return resp
}

func detailResponseFromCache(entry *cachedGetWormMarketResponse, stale bool) *apiclient.GetWormMarketResponse {
	resp := cloneDetailResponse(entry.Response)
	resp.FetchedAt = entry.FetchedAt
	resp.Stale = stale
	return resp
}

func cloneListResponse(resp *apiclient.ListWormMarketsResponse) *apiclient.ListWormMarketsResponse {
	if resp == nil {
		return &apiclient.ListWormMarketsResponse{}
	}
	return proto.Clone(resp).(*apiclient.ListWormMarketsResponse)
}

func cloneDetailResponse(resp *apiclient.GetWormMarketResponse) *apiclient.GetWormMarketResponse {
	if resp == nil {
		return &apiclient.GetWormMarketResponse{}
	}
	return proto.Clone(resp).(*apiclient.GetWormMarketResponse)
}

func (p listWormMarketsParams) isDefaultFirstPage() bool {
	return p.Limit == defaultWormMarketsLimit &&
		strings.TrimSpace(p.Cursor) == "" &&
		p.SortOption == defaultWormMarketsSortOption &&
		p.CategorySlug == defaultWormMarketsCategorySlug
}

func wormListCacheKey(params listWormMarketsParams) string {
	cursorHash := sha256.Sum256([]byte(params.Cursor))
	return fmt.Sprintf(
		"worm:list:v1:%s:%s:%d:%s",
		params.SortOption,
		params.CategorySlug,
		params.Limit,
		hex.EncodeToString(cursorHash[:])[:16],
	)
}

func wormDetailCacheKey(conditionID string) string {
	return "worm:detail:v2:" + strings.TrimSpace(conditionID)
}

func wormRefreshLockKey(key string) string {
	return "worm:refresh-lock:" + key
}

func wormUpstreamMinuteKey(bucket int64) string {
	return fmt.Sprintf("worm:upstream:minute:%d", bucket)
}

func wormUpstreamCooldownKey() string {
	return "worm:upstream:cooldown"
}

func wormHotListKey() string {
	return "worm:hot:list"
}

func wormHotDetailKey() string {
	return "worm:hot:detail"
}
