package phantomauth

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/useryege/athena/internal/walletsecret"
)

const walletSecretChallengePrefix = "wallet-secret-solana-reauth|"

var (
	errWalletSecretChallengeNotFound    = errors.New("wallet-secret Solana challenge is missing or expired")
	errWalletSecretChallengeUnavailable = errors.New("wallet-secret Solana challenge storage is unavailable")
)

type walletSecretChallenge struct {
	Address          string    `json:"address"`
	Message          string    `json:"message"`
	Nonce            string    `json:"nonce"`
	AccountID        string    `json:"accountId"`
	SessionJTIDigest string    `json:"sessionJtiDigest"`
	AccessRevision   uint64    `json:"accessRevision"`
	CreatedAt        time.Time `json:"createdAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type walletSecretChallengeStore struct {
	redis *redis.Client
}

func newWalletSecretChallengeStore(redisClient *redis.Client) (*walletSecretChallengeStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("wallet-secret Solana Redis client is required")
	}
	return &walletSecretChallengeStore{redis: redisClient}, nil
}

func (s *walletSecretChallengeStore) create(ctx context.Context, id string, value walletSecretChallenge) error {
	if err := validateWalletSecretChallenge(value); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode wallet-secret Solana challenge: %w", err)
	}
	if err := walletsecret.CreateRateLimitedState(
		ctx,
		s.redis,
		walletSecretChallengeKey(id),
		encoded,
		challengeTTL,
		value.AccountID,
	); err != nil {
		return fmt.Errorf("%w: %v", errWalletSecretChallengeUnavailable, err)
	}
	return nil
}

func (s *walletSecretChallengeStore) consume(ctx context.Context, id string) (walletSecretChallenge, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	encoded, err := s.redis.Eval(ctx, script, []string{walletSecretChallengeKey(id)}).Text()
	if errors.Is(err, redis.Nil) {
		return walletSecretChallenge{}, errWalletSecretChallengeNotFound
	}
	if err != nil {
		return walletSecretChallenge{}, fmt.Errorf("%w: %v", errWalletSecretChallengeUnavailable, err)
	}
	var value walletSecretChallenge
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return walletSecretChallenge{}, fmt.Errorf("decode wallet-secret Solana challenge: %w", err)
	}
	if err := validateWalletSecretChallenge(value); err != nil {
		return walletSecretChallenge{}, errWalletSecretChallengeNotFound
	}
	now := time.Now().UTC()
	if value.CreatedAt.After(now.Add(time.Minute)) || !value.ExpiresAt.After(now) {
		return walletSecretChallenge{}, errWalletSecretChallengeNotFound
	}
	return value, nil
}

func validateWalletSecretChallenge(value walletSecretChallenge) error {
	if value.Address == "" || value.Message == "" || value.Nonce == "" || value.AccountID == "" || value.SessionJTIDigest == "" || value.AccessRevision == 0 || value.CreatedAt.IsZero() || value.ExpiresAt.IsZero() {
		return fmt.Errorf("wallet-secret Solana challenge is incomplete")
	}
	if !value.ExpiresAt.After(value.CreatedAt) || value.ExpiresAt.Sub(value.CreatedAt) != challengeTTL {
		return fmt.Errorf("wallet-secret Solana challenge lifetime is invalid")
	}
	return nil
}

func walletSecretChallengeKey(id string) string {
	digest := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%s%x", walletSecretChallengePrefix, digest[:])
}
