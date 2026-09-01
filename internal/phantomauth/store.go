package phantomauth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
)

const (
	challengePrefix           = "phantom-auth-challenge|"
	challengeRateGlobalKey    = "phantom-auth-challenge-rate|global"
	challengeRateClientPrefix = "phantom-auth-challenge-rate|client|"
	challengeTTL              = 5 * time.Minute
	challengeRateWindow       = time.Minute
	challengeGlobalRateLimit  = 120
	challengeClientRateLimit  = 20
)

var (
	errChallengeNotFound    = errors.New("Phantom challenge is missing or expired")
	errChallengeUnavailable = errors.New("Phantom challenge storage is unavailable")
	errChallengeRateLimited = errors.New("Phantom challenge rate limit exceeded")
)

type challenge struct {
	Address   string    `json:"address"`
	Message   string    `json:"message"`
	Nonce     string    `json:"nonce"`
	ReturnTo  string    `json:"returnTo"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

type challengeStore struct {
	redis *redis.Client
}

func newChallengeStore(redisClient *redis.Client) (*challengeStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Phantom authentication Redis client is required")
	}
	return &challengeStore{redis: redisClient}, nil
}

func (s *challengeStore) Create(ctx context.Context, id, clientRateKey string, value challenge) error {
	if err := validateChallenge(value); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Phantom challenge: %w", err)
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
			challengeRateGlobalKey,
			challengeRateClientPrefix + clientRateKey,
			challengeKey(id),
		},
		encoded,
		int64(challengeRateWindow/time.Second),
		challengeGlobalRateLimit,
		challengeClientRateLimit,
		int64(challengeTTL/time.Second),
	).Int64()
	if err != nil {
		return fmt.Errorf("%w: %v", errChallengeUnavailable, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return errChallengeRateLimited
	case -1:
		return fmt.Errorf("Phantom challenge collision")
	default:
		return fmt.Errorf("%w: unexpected challenge result %d", errChallengeUnavailable, result)
	}
}

// Consume atomically removes a challenge before signature parsing and
// verification, making every verification request a one-time attempt.
func (s *challengeStore) Consume(ctx context.Context, id string) (challenge, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	value, err := s.redis.Eval(ctx, script, []string{challengeKey(id)}).Text()
	if errors.Is(err, redis.Nil) {
		return challenge{}, errChallengeNotFound
	}
	if err != nil {
		return challenge{}, fmt.Errorf("%w: %v", errChallengeUnavailable, err)
	}
	var decoded challenge
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return challenge{}, fmt.Errorf("decode Phantom challenge: %w", err)
	}
	if err := validateChallenge(decoded); err != nil {
		return challenge{}, err
	}
	now := time.Now().UTC()
	if decoded.CreatedAt.After(now) || !decoded.ExpiresAt.After(now) || decoded.ExpiresAt.After(decoded.CreatedAt.Add(challengeTTL)) {
		return challenge{}, errChallengeNotFound
	}
	return decoded, nil
}

func validateChallenge(value challenge) error {
	if value.Address == "" || value.Message == "" || value.Nonce == "" || value.ReturnTo == "" || value.ReturnTo != authregistration.ReturnToForRealm(value.ReturnTo, accountcredentials.ApplicationRealmMember) || value.CreatedAt.IsZero() || value.ExpiresAt.IsZero() {
		return fmt.Errorf("Phantom challenge is incomplete")
	}
	nonce, err := hex.DecodeString(value.Nonce)
	if err != nil || len(nonce) != 16 || hex.EncodeToString(nonce) != value.Nonce {
		return fmt.Errorf("Phantom challenge nonce is invalid")
	}
	if !value.ExpiresAt.After(value.CreatedAt) || value.ExpiresAt.Sub(value.CreatedAt) != challengeTTL {
		return fmt.Errorf("Phantom challenge lifetime is invalid")
	}
	return nil
}

func challengeKey(id string) string {
	digest := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%s%x", challengePrefix, digest[:])
}
