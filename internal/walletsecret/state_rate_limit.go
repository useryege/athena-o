package walletsecret

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	stateCreationRateGlobalKey     = "wallet-secret-reauth-rate|global"
	stateCreationRateAccountPrefix = "wallet-secret-reauth-rate|account|"
	stateCreationRateWindow        = time.Minute
	stateCreationGlobalRateLimit   = 120
	stateCreationAccountRateLimit  = 20
)

var (
	errStateCreationRateLimited = errors.New("wallet-secret reauthentication state creation rate limit exceeded")
	errStateCreationCollision   = errors.New("wallet-secret reauthentication state already exists")
)

// CreateRateLimitedState atomically enforces the wallet-secret provider-state
// creation limits and stores one provider-owned state record. The rate-key
// namespace is independent from primary login, and its account suffix is a
// fixed-width digest rather than the authenticated account UUID.
func CreateRateLimitedState(ctx context.Context, redisClient *redis.Client, stateKey string, encoded []byte, ttl time.Duration, accountID string) error {
	if redisClient == nil {
		return fmt.Errorf("wallet-secret reauthentication Redis client is required")
	}
	stateKey = strings.TrimSpace(stateKey)
	accountID = strings.TrimSpace(accountID)
	ttlSeconds := int64(ttl / time.Second)
	if stateKey == "" || len(encoded) == 0 || accountID == "" || ttlSeconds <= 0 || time.Duration(ttlSeconds)*time.Second != ttl {
		return fmt.Errorf("wallet-secret reauthentication state is incomplete")
	}

	accountDigest := sha256.Sum256([]byte(accountID))
	accountRateKey := stateCreationRateAccountPrefix + hex.EncodeToString(accountDigest[:])
	const script = `
local globalCount = tonumber(redis.call("GET", KEYS[1]) or "0")
if globalCount >= tonumber(ARGV[3]) then
  return 0
end
local accountCount = redis.call("INCR", KEYS[2])
if accountCount == 1 then
  redis.call("EXPIRE", KEYS[2], ARGV[2])
end
if accountCount > tonumber(ARGV[4]) then
  return 0
end
globalCount = redis.call("INCR", KEYS[1])
if globalCount == 1 then
  redis.call("EXPIRE", KEYS[1], ARGV[2])
end
local created = redis.call("SET", KEYS[3], ARGV[1], "EX", ARGV[5], "NX")
if created then
  return 1
end
return -1
`
	result, err := redisClient.Eval(
		ctx,
		script,
		[]string{stateCreationRateGlobalKey, accountRateKey, stateKey},
		encoded,
		int64(stateCreationRateWindow/time.Second),
		stateCreationGlobalRateLimit,
		stateCreationAccountRateLimit,
		ttlSeconds,
	).Int64()
	if err != nil {
		return fmt.Errorf("store wallet-secret reauthentication state: %w", err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return errStateCreationRateLimited
	case -1:
		return errStateCreationCollision
	default:
		return fmt.Errorf("unexpected wallet-secret reauthentication state result %d", result)
	}
}
