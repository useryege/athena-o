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

const wormCredentialChallengePrefix = "worm-credential-solana-reauth|"

var (
	errWormCredentialChallengeNotFound    = errors.New("Worm credential Solana challenge is missing or expired")
	errWormCredentialChallengeUnavailable = errors.New("Worm credential Solana challenge storage is unavailable")
)

type wormCredentialChallenge struct {
	Address          string    `json:"address"`
	Message          string    `json:"message"`
	Nonce            string    `json:"nonce"`
	AccountID        string    `json:"accountId"`
	SessionJTIDigest string    `json:"sessionJtiDigest"`
	AccessRevision   uint64    `json:"accessRevision"`
	CreatedAt        time.Time `json:"createdAt"`
	ExpiresAt        time.Time `json:"expiresAt"`
}

type wormCredentialChallengeStore struct {
	redis *redis.Client
}

func newWormCredentialChallengeStore(redisClient *redis.Client) (*wormCredentialChallengeStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Worm credential Solana Redis client is required")
	}
	return &wormCredentialChallengeStore{redis: redisClient}, nil
}

func (s *wormCredentialChallengeStore) create(ctx context.Context, id string, value wormCredentialChallenge) error {
	if err := validateWormCredentialChallenge(value); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Worm credential Solana challenge: %w", err)
	}
	if err := walletsecret.CreateRateLimitedState(
		ctx,
		s.redis,
		wormCredentialChallengeKey(id),
		encoded,
		challengeTTL,
		value.AccountID,
	); err != nil {
		return fmt.Errorf("%w: %v", errWormCredentialChallengeUnavailable, err)
	}
	return nil
}

func (s *wormCredentialChallengeStore) consume(ctx context.Context, id string) (wormCredentialChallenge, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	encoded, err := s.redis.Eval(ctx, script, []string{wormCredentialChallengeKey(id)}).Text()
	if errors.Is(err, redis.Nil) {
		return wormCredentialChallenge{}, errWormCredentialChallengeNotFound
	}
	if err != nil {
		return wormCredentialChallenge{}, fmt.Errorf("%w: %v", errWormCredentialChallengeUnavailable, err)
	}
	var value wormCredentialChallenge
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return wormCredentialChallenge{}, fmt.Errorf("decode Worm credential Solana challenge: %w", err)
	}
	if err := validateWormCredentialChallenge(value); err != nil {
		return wormCredentialChallenge{}, errWormCredentialChallengeNotFound
	}
	now := time.Now().UTC()
	if value.CreatedAt.After(now.Add(time.Minute)) || !value.ExpiresAt.After(now) {
		return wormCredentialChallenge{}, errWormCredentialChallengeNotFound
	}
	return value, nil
}

func validateWormCredentialChallenge(value wormCredentialChallenge) error {
	if value.Address == "" || value.Message == "" || value.Nonce == "" || value.AccountID == "" || value.SessionJTIDigest == "" || value.AccessRevision == 0 || value.CreatedAt.IsZero() || value.ExpiresAt.IsZero() {
		return fmt.Errorf("Worm credential Solana challenge is incomplete")
	}
	if !value.ExpiresAt.After(value.CreatedAt) || value.ExpiresAt.Sub(value.CreatedAt) != challengeTTL {
		return fmt.Errorf("Worm credential Solana challenge lifetime is invalid")
	}
	return nil
}

func wormCredentialChallengeKey(id string) string {
	digest := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%s%x", wormCredentialChallengePrefix, digest[:])
}
