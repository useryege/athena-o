package session

import (
	"context"
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
	redis               *redis.Client            // db
	attempts            map[string]LoginAttempts // login attempts
	revokedTokens       map[string]bool          // revoked tokens
	recentRevokedTokens map[string]bool          // recent revoked tokens
	lock                sync.RWMutex
	resyncDuration      time.Duration // resync duration
}

var _ UserStateStorage = &userStateStorage{}

func NewUserStateStorage(redis *redis.Client) *userStateStorage {
	return &userStateStorage{
		attempts:            map[string]LoginAttempts{},
		revokedTokens:       map[string]bool{},
		recentRevokedTokens: map[string]bool{},
		resyncDuration:      time.Second * 15, // every 15 seconds to resync the revoked tokens
		redis:               redis,            // db
	}
}

// Init sets up watches on the revoked tokens and starts a ticker to periodically resync the revoked tokens from Redis.
// Don't call this until after setting up all hooks on the Redis client, or you might encounter race conditions.
func (storage *userStateStorage) Init(ctx context.Context) {
	go storage.watchRevokedTokens(ctx)

	ticker := time.NewTicker(storage.resyncDuration)
	go func() {
		storage.loadRevokedTokensSafe()
		for range ticker.C {
			storage.loadRevokedTokensSafe()
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
		case val := <-ch:
			storage.lock.Lock()
			storage.revokedTokens[val.Payload] = true
			storage.recentRevokedTokens[val.Payload] = true
			storage.lock.Unlock()
		}
	}
}

func (storage *userStateStorage) loadRevokedTokensSafe() {
	err := storage.loadRevokedTokens()
	for err != nil {
		log.Warnf("Failed to resync revoked tokens. retrying again in 1 minute: %v", err) // every 1 minute to retry
		time.Sleep(time.Minute)
		err = storage.loadRevokedTokens()
	}
}

func (storage *userStateStorage) loadRevokedTokens() error {
	redisRevokedTokens := map[string]bool{}
	iterator := storage.redis.Scan(context.Background(), 0, revokedTokenPrefix+"*", 10000).Iterator()
	for iterator.Next(context.Background()) {
		parts := strings.Split(iterator.Val(), "|")
		if len(parts) != 2 {
			log.Warnf("Unexpected redis key prefixed with '%s'. Must have token id after the prefix but got: '%s'.",
				revokedTokenPrefix,
				iterator.Val())
			continue
		}
		redisRevokedTokens[parts[1]] = true
	}
	if iterator.Err() != nil {
		return iterator.Err()
	}

	storage.lock.Lock()
	defer storage.lock.Unlock()
	storage.revokedTokens = redisRevokedTokens
	for recentRevokedToken := range storage.recentRevokedTokens {
		storage.revokedTokens[recentRevokedToken] = true
	}
	storage.recentRevokedTokens = map[string]bool{}

	return nil
}

func (storage *userStateStorage) GetLoginAttempts() map[string]LoginAttempts {
	return storage.attempts
}

func (storage *userStateStorage) SetLoginAttempts(attempts map[string]LoginAttempts) error {
	storage.attempts = attempts
	return nil
}

func (storage *userStateStorage) RevokeToken(ctx context.Context, id string, expiringAt time.Duration) error {
	storage.lock.Lock()
	storage.revokedTokens[id] = true
	storage.recentRevokedTokens[id] = true
	storage.lock.Unlock()
	if err := storage.redis.Set(ctx, revokedTokenPrefix+id, "", expiringAt).Err(); err != nil {
		return err
	}
	return storage.redis.Publish(ctx, newRevokedTokenKey, id).Err()
}

func (storage *userStateStorage) IsTokenRevoked(id string) bool {
	storage.lock.RLock()
	defer storage.lock.RUnlock()
	return storage.revokedTokens[id]
}

func (storage *userStateStorage) GetLockObject() *sync.RWMutex {
	return &storage.lock
}

type UserStateStorage interface {
	Init(ctx context.Context)
	// GetLoginAttempts return number of concurrent login attempts
	GetLoginAttempts() map[string]LoginAttempts
	// SetLoginAttempts sets number of concurrent login attempts
	SetLoginAttempts(attempts map[string]LoginAttempts) error
	// RevokeToken revokes token with given id (information about revocation expires after specified timeout)
	RevokeToken(ctx context.Context, id string, expiringAt time.Duration) error
	// IsTokenRevoked checks if given token is revoked
	IsTokenRevoked(id string) bool
	// GetLockObject returns a lock used by the storage
	GetLockObject() *sync.RWMutex
}
