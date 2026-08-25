package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"

	utilio "github.com/useryege/athena/util/io"
)

const (
	revokedTokenPrefix = "revoked-token|"
	newRevokedTokenKey = "new-revoked-token"
)

type userStateStorage struct {
	redis              *redis.Client
	revokedTokens      map[string]bool
	pendingRevocations map[string]time.Time
	initialized        bool
	lock               sync.RWMutex
	resyncDuration     time.Duration
}

type revocationNotice struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expiresAt"`
}

var _ UserStateStorage = &userStateStorage{}

func NewUserStateStorage(redisClient *redis.Client) *userStateStorage {
	return &userStateStorage{
		revokedTokens:      map[string]bool{},
		pendingRevocations: map[string]time.Time{},
		resyncDuration:     15 * time.Second,
		redis:              redisClient,
	}
}

// Init establishes the initial Redis-backed revocation snapshot before
// returning, then starts background synchronization. A fresh process therefore
// never serves authenticated traffic from an empty revocation cache.
func (storage *userStateStorage) Init(ctx context.Context) {
	if storage.redis == nil {
		log.Warn("Session revocation Redis client is not configured")
		return
	}
	go storage.watchRevokedTokens(ctx)
	storage.loadRevokedTokensSafe(ctx)
	if ctx.Err() != nil {
		return
	}
	ticker := time.NewTicker(storage.resyncDuration)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				storage.loadRevokedTokensSafe(ctx)
			}
		}
	}()
	go func() {
		<-ctx.Done()
		ticker.Stop()
	}()
}

func (storage *userStateStorage) watchRevokedTokens(ctx context.Context) {
	pubsub := storage.redis.Subscribe(ctx, newRevokedTokenKey)
	defer utilio.Close(pubsub)
	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case value, ok := <-ch:
			if !ok {
				return
			}
			var notice revocationNotice
			if err := json.Unmarshal([]byte(value.Payload), &notice); err != nil || notice.ID == "" || notice.ExpiresAt.IsZero() {
				log.Warn("Ignored malformed session revocation notification")
				continue
			}
			if !notice.ExpiresAt.After(time.Now()) {
				continue
			}
			storage.lock.Lock()
			storage.revokedTokens[notice.ID] = true
			storage.pendingRevocations[notice.ID] = notice.ExpiresAt
			storage.lock.Unlock()
		}
	}
}

func (storage *userStateStorage) loadRevokedTokensSafe(ctx context.Context) {
	for {
		if err := storage.loadRevokedTokens(ctx); err == nil {
			return
		} else {
			log.Warnf("Failed to resync revoked tokens; retrying in 1 minute: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Minute):
		}
	}
}

func (storage *userStateStorage) loadRevokedTokens(ctx context.Context) error {
	if err := storage.persistPendingRevocations(ctx); err != nil {
		return err
	}
	redisRevokedTokens := map[string]bool{}
	iterator := storage.redis.Scan(ctx, 0, revokedTokenPrefix+"*", 10000).Iterator()
	for iterator.Next(ctx) {
		id, ok := strings.CutPrefix(iterator.Val(), revokedTokenPrefix)
		if !ok || id == "" {
			log.Warnf("Unexpected Redis key prefixed with %q: %q", revokedTokenPrefix, iterator.Val())
			continue
		}
		redisRevokedTokens[id] = true
	}
	if iterator.Err() != nil {
		return iterator.Err()
	}

	storage.lock.Lock()
	defer storage.lock.Unlock()
	storage.revokedTokens = redisRevokedTokens
	now := time.Now()
	for id, expiresAt := range storage.pendingRevocations {
		if !expiresAt.After(now) {
			delete(storage.pendingRevocations, id)
			continue
		}
		if redisRevokedTokens[id] {
			delete(storage.pendingRevocations, id)
			continue
		}
		storage.revokedTokens[id] = true
	}
	storage.initialized = true
	return nil
}

func (storage *userStateStorage) persistPendingRevocations(ctx context.Context) error {
	storage.lock.RLock()
	pending := make(map[string]time.Time, len(storage.pendingRevocations))
	for id, expiresAt := range storage.pendingRevocations {
		pending[id] = expiresAt
	}
	storage.lock.RUnlock()

	now := time.Now()
	for id, expiresAt := range pending {
		if ttl := expiresAt.Sub(now); ttl > 0 {
			if err := storage.redis.Set(ctx, revokedTokenPrefix+id, "", ttl).Err(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (storage *userStateStorage) RevokeToken(ctx context.Context, id string, expiringAt time.Duration) error {
	if storage.redis == nil {
		return fmt.Errorf("session revocation Redis client is not configured")
	}
	if id == "" || expiringAt <= 0 {
		return fmt.Errorf("session revocation requires a token JTI and positive lifetime")
	}
	expiresAt := time.Now().Add(expiringAt)
	storage.lock.Lock()
	storage.revokedTokens[id] = true
	storage.pendingRevocations[id] = expiresAt
	storage.lock.Unlock()
	if err := storage.redis.Set(ctx, revokedTokenPrefix+id, "", expiringAt).Err(); err != nil {
		return err
	}
	notice, err := json.Marshal(revocationNotice{ID: id, ExpiresAt: expiresAt})
	if err != nil {
		return fmt.Errorf("encode session revocation notification: %w", err)
	}
	return storage.redis.Publish(ctx, newRevokedTokenKey, notice).Err()
}

func (storage *userStateStorage) IsTokenRevoked(id string) bool {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	return !storage.initialized || storage.revokedTokens[id]
}

type UserStateStorage interface {
	Init(ctx context.Context)
	RevokeToken(ctx context.Context, id string, expiringAt time.Duration) error
	IsTokenRevoked(id string) bool
}
