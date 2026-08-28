package googleoidc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/useryege/athena/internal/authregistration"
	"github.com/useryege/athena/internal/walletsecret"
)

const (
	wormCredentialStatePrefix       = "wc."
	wormCredentialTransactionPrefix = "worm-credential-google-reauth|"
	wormCredentialTransactionTTL    = 5 * time.Minute
)

var (
	errWormCredentialTransactionNotFound    = errors.New("Worm credential Google transaction is missing or expired")
	errWormCredentialTransactionUnavailable = errors.New("Worm credential Google transaction storage is unavailable")
)

type wormCredentialTransaction struct {
	Nonce            string    `json:"nonce"`
	Verifier         string    `json:"verifier"`
	ReturnTo         string    `json:"returnTo"`
	AccountID        string    `json:"accountId"`
	SessionJTIDigest string    `json:"sessionJtiDigest"`
	AccessRevision   uint64    `json:"accessRevision"`
	CreatedAt        time.Time `json:"createdAt"`
}

type wormCredentialTransactionStore struct {
	redis *redis.Client
}

func newWormCredentialTransactionStore(redisClient *redis.Client) (*wormCredentialTransactionStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Worm credential Google Redis client is required")
	}
	return &wormCredentialTransactionStore{redis: redisClient}, nil
}

func (s *wormCredentialTransactionStore) create(ctx context.Context, state string, value wormCredentialTransaction) error {
	if !validWormCredentialState(state) || value.Nonce == "" || value.Verifier == "" || value.ReturnTo == "" || value.AccountID == "" || value.SessionJTIDigest == "" || value.AccessRevision == 0 || value.CreatedAt.IsZero() {
		return fmt.Errorf("Worm credential Google transaction is incomplete")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Worm credential Google transaction: %w", err)
	}
	if err := walletsecret.CreateRateLimitedState(
		ctx,
		s.redis,
		wormCredentialTransactionKey(state),
		encoded,
		wormCredentialTransactionTTL,
		value.AccountID,
	); err != nil {
		return fmt.Errorf("%w: %v", errWormCredentialTransactionUnavailable, err)
	}
	return nil
}

func (s *wormCredentialTransactionStore) consume(ctx context.Context, state string) (wormCredentialTransaction, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	encoded, err := s.redis.Eval(ctx, script, []string{wormCredentialTransactionKey(state)}).Text()
	if errors.Is(err, redis.Nil) {
		return wormCredentialTransaction{}, errWormCredentialTransactionNotFound
	}
	if err != nil {
		return wormCredentialTransaction{}, fmt.Errorf("%w: %v", errWormCredentialTransactionUnavailable, err)
	}
	var value wormCredentialTransaction
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return wormCredentialTransaction{}, fmt.Errorf("decode Worm credential Google transaction: %w", err)
	}
	if value.Nonce == "" || value.Verifier == "" || value.ReturnTo == "" || value.AccountID == "" || value.SessionJTIDigest == "" || value.AccessRevision == 0 || value.CreatedAt.IsZero() {
		return wormCredentialTransaction{}, errWormCredentialTransactionNotFound
	}
	now := time.Now().UTC()
	if value.CreatedAt.After(now.Add(time.Minute)) || now.Sub(value.CreatedAt) > wormCredentialTransactionTTL {
		return wormCredentialTransaction{}, errWormCredentialTransactionNotFound
	}
	return value, nil
}

func newWormCredentialState() (string, error) {
	opaque, err := authregistration.RandomOpaqueValue()
	if err != nil {
		return "", err
	}
	return wormCredentialStatePrefix + opaque, nil
}

func validWormCredentialState(value string) bool {
	return strings.HasPrefix(value, wormCredentialStatePrefix) && authregistration.ValidOpaqueValue(strings.TrimPrefix(value, wormCredentialStatePrefix))
}

func wormCredentialTransactionKey(state string) string {
	digest := sha256.Sum256([]byte(state))
	return fmt.Sprintf("%s%x", wormCredentialTransactionPrefix, digest[:])
}
