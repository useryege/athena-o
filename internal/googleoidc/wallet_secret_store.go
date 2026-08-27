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
	walletSecretStatePrefix       = "ws."
	walletSecretTransactionPrefix = "wallet-secret-google-reauth|"
	walletSecretTransactionTTL    = 5 * time.Minute
)

var (
	errWalletSecretTransactionNotFound    = errors.New("wallet-secret Google transaction is missing or expired")
	errWalletSecretTransactionUnavailable = errors.New("wallet-secret Google transaction storage is unavailable")
)

type walletSecretTransaction struct {
	Nonce            string    `json:"nonce"`
	Verifier         string    `json:"verifier"`
	ReturnTo         string    `json:"returnTo"`
	AccountID        string    `json:"accountId"`
	SessionJTIDigest string    `json:"sessionJtiDigest"`
	AccessRevision   uint64    `json:"accessRevision"`
	CreatedAt        time.Time `json:"createdAt"`
}

type walletSecretTransactionStore struct {
	redis *redis.Client
}

func newWalletSecretTransactionStore(redisClient *redis.Client) (*walletSecretTransactionStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("wallet-secret Google Redis client is required")
	}
	return &walletSecretTransactionStore{redis: redisClient}, nil
}

func (s *walletSecretTransactionStore) create(ctx context.Context, state string, value walletSecretTransaction) error {
	if !validWalletSecretState(state) || value.Nonce == "" || value.Verifier == "" || value.ReturnTo == "" || value.AccountID == "" || value.SessionJTIDigest == "" || value.AccessRevision == 0 || value.CreatedAt.IsZero() {
		return fmt.Errorf("wallet-secret Google transaction is incomplete")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode wallet-secret Google transaction: %w", err)
	}
	if err := walletsecret.CreateRateLimitedState(
		ctx,
		s.redis,
		walletSecretTransactionKey(state),
		encoded,
		walletSecretTransactionTTL,
		value.AccountID,
	); err != nil {
		return fmt.Errorf("%w: %v", errWalletSecretTransactionUnavailable, err)
	}
	return nil
}

func (s *walletSecretTransactionStore) consume(ctx context.Context, state string) (walletSecretTransaction, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	encoded, err := s.redis.Eval(ctx, script, []string{walletSecretTransactionKey(state)}).Text()
	if errors.Is(err, redis.Nil) {
		return walletSecretTransaction{}, errWalletSecretTransactionNotFound
	}
	if err != nil {
		return walletSecretTransaction{}, fmt.Errorf("%w: %v", errWalletSecretTransactionUnavailable, err)
	}
	var value walletSecretTransaction
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return walletSecretTransaction{}, fmt.Errorf("decode wallet-secret Google transaction: %w", err)
	}
	if value.Nonce == "" || value.Verifier == "" || value.ReturnTo == "" || value.AccountID == "" || value.SessionJTIDigest == "" || value.AccessRevision == 0 || value.CreatedAt.IsZero() {
		return walletSecretTransaction{}, errWalletSecretTransactionNotFound
	}
	now := time.Now().UTC()
	if value.CreatedAt.After(now.Add(time.Minute)) || now.Sub(value.CreatedAt) > walletSecretTransactionTTL {
		return walletSecretTransaction{}, errWalletSecretTransactionNotFound
	}
	return value, nil
}

func newWalletSecretState() (string, error) {
	opaque, err := authregistration.RandomOpaqueValue()
	if err != nil {
		return "", err
	}
	return walletSecretStatePrefix + opaque, nil
}

func validWalletSecretState(value string) bool {
	return strings.HasPrefix(value, walletSecretStatePrefix) && authregistration.ValidOpaqueValue(strings.TrimPrefix(value, walletSecretStatePrefix))
}

func walletSecretTransactionKey(state string) string {
	digest := sha256.Sum256([]byte(state))
	return fmt.Sprintf("%s%x", walletSecretTransactionPrefix, digest[:])
}
