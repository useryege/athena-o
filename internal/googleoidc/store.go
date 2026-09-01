package googleoidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
)

const (
	transactionPrefix          = "google-oidc-transaction|"
	transactionRateGlobalKey   = "google-oidc-login-rate|global"
	transactionRateClientKey   = "google-oidc-login-rate|client|"
	transactionTTL             = 5 * time.Minute
	transactionRateWindow      = time.Minute
	transactionGlobalRateLimit = 120
	transactionClientRateLimit = 20
)

var (
	errTransactionNotFound    = errors.New("Google OIDC transaction is missing or expired")
	errTransactionUnavailable = errors.New("Google OIDC transaction storage is unavailable")
	errTransactionRateLimited = errors.New("Google OIDC login rate limit exceeded")
)

type transaction struct {
	Nonce     string                              `json:"nonce"`
	Verifier  string                              `json:"verifier"`
	ReturnTo  string                              `json:"returnTo"`
	Realm     accountcredentials.ApplicationRealm `json:"realm"`
	CreatedAt time.Time                           `json:"createdAt"`
}

// TransactionStore owns short-lived, one-time Google OAuth transactions.
type TransactionStore struct {
	redis *redis.Client
}

func NewTransactionStore(redisClient *redis.Client) (*TransactionStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Google OIDC transaction Redis client is required")
	}
	return &TransactionStore{redis: redisClient}, nil
}

func (s *TransactionStore) Create(ctx context.Context, state, clientRateKey string, value transaction) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Google OIDC transaction: %w", err)
	}
	const script = `
local globalCount = tonumber(redis.call("GET", KEYS[1]) or "0")
if globalCount >= tonumber(ARGV[3]) then
  return 0
end
local clientCount = redis.call("INCR", KEYS[2])
if clientCount == 1 then
  redis.call("EXPIRE", KEYS[2], ARGV[2])
end
if clientCount > tonumber(ARGV[4]) then
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
	result, err := s.redis.Eval(
		ctx,
		script,
		[]string{
			transactionRateGlobalKey,
			transactionRateClientKey + clientRateKey,
			transactionPrefix + state,
		},
		encoded,
		int64(transactionRateWindow/time.Second),
		transactionGlobalRateLimit,
		transactionClientRateLimit,
		int64(transactionTTL/time.Second),
	).Int64()
	if err != nil {
		return fmt.Errorf("store Google OIDC transaction: %w", err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return errTransactionRateLimited
	case -1:
		return fmt.Errorf("Google OIDC state collision")
	default:
		return fmt.Errorf("unexpected Google OIDC transaction result %d", result)
	}
}

// Consume atomically reads and removes a transaction so every callback,
// including failures, makes the state unusable for replay.
func (s *TransactionStore) Consume(ctx context.Context, state string) (transaction, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	value, err := s.redis.Eval(ctx, script, []string{transactionPrefix + state}).Text()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return transaction{}, errTransactionNotFound
		}
		return transaction{}, fmt.Errorf("%w: %v", errTransactionUnavailable, err)
	}
	var decoded transaction
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return transaction{}, fmt.Errorf("decode Google OIDC transaction: %w", err)
	}
	if _, err := accountcredentials.ParseApplicationRealm(string(decoded.Realm)); err != nil || decoded.Nonce == "" || decoded.Verifier == "" || decoded.ReturnTo == "" || decoded.ReturnTo != authregistration.ReturnToForRealm(decoded.ReturnTo, decoded.Realm) || decoded.CreatedAt.IsZero() {
		return transaction{}, fmt.Errorf("Google OIDC transaction is incomplete")
	}
	return decoded, nil
}
