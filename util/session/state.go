package session

import (
	"context"
	"strconv"
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
	redis               *redis.Client   // db
	revokedTokens       map[string]bool // revoked tokens
	recentRevokedTokens map[string]bool // recent revoked tokens
	lock                sync.RWMutex
	resyncDuration      time.Duration // resync duration
}

var _ UserStateStorage = &userStateStorage{}

func NewUserStateStorage(redis *redis.Client) *userStateStorage {
	return &userStateStorage{
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

func (storage *userStateStorage) CheckLoginRateLimit(ctx context.Context, rules []loginRateLimitRule) error {
	activeRules := activeLoginRateLimitRules(rules)
	if len(activeRules) == 0 {
		return nil
	}
	if storage.redis == nil {
		return errLoginRateLimited
	}

	pipe := storage.redis.TxPipeline()
	existsCommands := make([]*redis.IntCmd, 0, len(activeRules))
	counterKeys := make([]string, 0, len(activeRules))
	for _, rule := range activeRules {
		existsCommands = append(existsCommands, pipe.Exists(ctx, rule.LockKey))
		counterKeys = append(counterKeys, rule.CounterKey)
	}
	counterCommand := pipe.MGet(ctx, counterKeys...)
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}
	for _, cmd := range existsCommands {
		if cmd.Val() > 0 {
			return errLoginRateLimited
		}
	}
	for i, value := range counterCommand.Val() {
		count, ok := redisCounterValue(value)
		if ok && count > int64(activeRules[i].MaxFailures) {
			return errLoginRateLimited
		}
	}
	return nil
}

func redisCounterValue(value any) (int64, bool) {
	switch typedValue := value.(type) {
	case nil:
		return 0, false
	case int64:
		return typedValue, true
	case string:
		count, err := strconv.ParseInt(typedValue, 10, 64)
		return count, err == nil
	case []byte:
		count, err := strconv.ParseInt(string(typedValue), 10, 64)
		return count, err == nil
	default:
		return 0, false
	}
}

func (storage *userStateStorage) RecordLoginFailure(ctx context.Context, rules []loginRateLimitRule) error {
	activeRules := activeLoginRateLimitRules(rules)
	if len(activeRules) == 0 {
		return nil
	}
	if storage.redis == nil {
		return errLoginRateLimited
	}

	pipe := storage.redis.TxPipeline()
	incrCommands := make([]*redis.IntCmd, 0, len(activeRules))
	for _, rule := range activeRules {
		incrCommands = append(incrCommands, pipe.Incr(ctx, rule.CounterKey))
		if rule.Window > 0 {
			pipe.Expire(ctx, rule.CounterKey, rule.Window)
		}
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return err
	}

	lockPipe := storage.redis.TxPipeline()
	locked := false
	for i, cmd := range incrCommands {
		if int(cmd.Val()) >= activeRules[i].MaxFailures {
			lockPipe.Set(ctx, activeRules[i].LockKey, "1", activeRules[i].LockFor)
			locked = true
		}
	}
	if !locked {
		return nil
	}
	_, err := lockPipe.Exec(ctx)
	return err
}

func (storage *userStateStorage) ClearLoginFailures(ctx context.Context, rules []loginRateLimitRule) error {
	activeRules := activeLoginRateLimitRules(rules)
	if len(activeRules) == 0 {
		return nil
	}
	if storage.redis == nil {
		return errLoginRateLimited
	}

	keys := make([]string, 0, len(activeRules)*2)
	for _, rule := range activeRules {
		keys = append(keys, rule.CounterKey, rule.LockKey)
	}
	return storage.redis.Del(ctx, keys...).Err()
}

func (storage *userStateStorage) SaveCaptcha(ctx context.Context, id, answer string, ttl time.Duration) error {
	if storage.redis == nil {
		return errLoginRateLimited
	}
	return storage.redis.Set(ctx, captchaKeyPrefix+id, strings.ToUpper(answer), ttl).Err()
}

func (storage *userStateStorage) ConsumeCaptcha(ctx context.Context, id string) (string, error) {
	if storage.redis == nil {
		return "", errLoginRateLimited
	}
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	value, err := storage.redis.Eval(ctx, script, []string{captchaKeyPrefix + id}).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	answer, _ := value.(string)
	return answer, nil
}

func activeLoginRateLimitRules(rules []loginRateLimitRule) []loginRateLimitRule {
	activeRules := make([]loginRateLimitRule, 0, len(rules))
	for _, rule := range rules {
		if rule.MaxFailures <= 0 || rule.LockFor <= 0 {
			continue
		}
		activeRules = append(activeRules, rule)
	}
	return activeRules
}

type UserStateStorage interface {
	Init(ctx context.Context)
	// RevokeToken revokes token with given id (information about revocation expires after specified timeout)
	RevokeToken(ctx context.Context, id string, expiringAt time.Duration) error
	// IsTokenRevoked checks if given token is revoked
	IsTokenRevoked(id string) bool
	CheckLoginRateLimit(ctx context.Context, rules []loginRateLimitRule) error
	RecordLoginFailure(ctx context.Context, rules []loginRateLimitRule) error
	ClearLoginFailures(ctx context.Context, rules []loginRateLimitRule) error
	SaveCaptcha(ctx context.Context, id, answer string, ttl time.Duration) error
	ConsumeCaptcha(ctx context.Context, id string) (string, error)
}
