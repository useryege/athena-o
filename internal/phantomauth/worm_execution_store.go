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
	wormExecutionChallengePrefix            = "worm-execution-solana-proof|"
	wormExecutionChallengeTTL               = 5 * time.Minute
	wormExecutionChallengeRateGlobalKey     = "worm-execution-solana-proof-rate|global"
	wormExecutionChallengeRateAccountPrefix = "worm-execution-solana-proof-rate|account|"
	wormExecutionChallengeRateWindow        = time.Minute
	wormExecutionChallengeRateGlobalLimit   = int64(120)
	wormExecutionChallengeRateAccountLimit  = int64(20)
)

var (
	errWormExecutionChallengeNotFound    = errors.New("Worm execution Solana challenge is missing or expired")
	errWormExecutionChallengeUnavailable = errors.New("Worm execution Solana challenge storage is unavailable")
)

type wormExecutionChallenge struct {
	Address                string    `json:"address"`
	Message                string    `json:"message"`
	Nonce                  string    `json:"nonce"`
	ReturnTo               string    `json:"returnTo"`
	RunID                  string    `json:"runId"`
	CommandID              string    `json:"commandId"`
	ExpectedRevision       int64     `json:"expectedRevision"`
	AccountID              string    `json:"accountId"`
	SessionJTIDigestSHA256 string    `json:"sessionJtiDigestSha256"`
	AccessRevision         uint64    `json:"accessRevision"`
	PlanDigestSHA256       string    `json:"planDigestSha256"`
	CreatedAt              time.Time `json:"createdAt"`
	ExpiresAt              time.Time `json:"expiresAt"`
}

type wormExecutionChallengeStore struct {
	redis *redis.Client
}

func newWormExecutionChallengeStore(redisClient *redis.Client) (*wormExecutionChallengeStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Worm execution Solana Redis client is required")
	}
	return &wormExecutionChallengeStore{redis: redisClient}, nil
}

func (s *wormExecutionChallengeStore) create(ctx context.Context, id string, value wormExecutionChallenge) error {
	if !authregistration.ValidOpaqueValue(id) {
		return fmt.Errorf("Worm execution Solana challenge ID is invalid")
	}
	if err := validateWormExecutionChallenge(value); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Worm execution Solana challenge: %w", err)
	}
	accountDigest := sha256.Sum256([]byte(value.AccountID))
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
	result, err := s.redis.Eval(
		ctx,
		script,
		[]string{
			wormExecutionChallengeRateGlobalKey,
			wormExecutionChallengeRateAccountPrefix + hex.EncodeToString(accountDigest[:]),
			wormExecutionChallengeKey(id),
		},
		encoded,
		int64(wormExecutionChallengeRateWindow/time.Second),
		wormExecutionChallengeRateGlobalLimit,
		wormExecutionChallengeRateAccountLimit,
		int64(wormExecutionChallengeTTL/time.Second),
	).Int64()
	if err != nil {
		return fmt.Errorf("%w: %v", errWormExecutionChallengeUnavailable, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("%w: proof challenge creation is rate limited", errWormExecutionChallengeUnavailable)
	case -1:
		return fmt.Errorf("%w: proof challenge state collision", errWormExecutionChallengeUnavailable)
	default:
		return fmt.Errorf("%w: unexpected challenge result %d", errWormExecutionChallengeUnavailable, result)
	}
}

func (s *wormExecutionChallengeStore) consume(ctx context.Context, id string) (wormExecutionChallenge, error) {
	if !authregistration.ValidOpaqueValue(id) {
		return wormExecutionChallenge{}, errWormExecutionChallengeNotFound
	}
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	encoded, err := s.redis.Eval(ctx, script, []string{wormExecutionChallengeKey(id)}).Text()
	if errors.Is(err, redis.Nil) {
		return wormExecutionChallenge{}, errWormExecutionChallengeNotFound
	}
	if err != nil {
		return wormExecutionChallenge{}, fmt.Errorf("%w: %v", errWormExecutionChallengeUnavailable, err)
	}
	var value wormExecutionChallenge
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return wormExecutionChallenge{}, fmt.Errorf("decode Worm execution Solana challenge: %w", err)
	}
	if err := validateWormExecutionChallenge(value); err != nil {
		return wormExecutionChallenge{}, errWormExecutionChallengeNotFound
	}
	now := time.Now().UTC()
	if value.CreatedAt.After(now.Add(time.Minute)) || !value.ExpiresAt.After(now) {
		return wormExecutionChallenge{}, errWormExecutionChallengeNotFound
	}
	return value, nil
}

func validateWormExecutionChallenge(value wormExecutionChallenge) error {
	if value.Address == "" || value.Message == "" || value.Nonce == "" || value.ReturnTo == "" || value.RunID == "" ||
		value.CommandID == "" || value.ExpectedRevision <= 0 || value.AccountID == "" || value.SessionJTIDigestSHA256 == "" ||
		value.AccessRevision == 0 || value.PlanDigestSHA256 == "" || value.CreatedAt.IsZero() || value.ExpiresAt.IsZero() {
		return fmt.Errorf("Worm execution Solana challenge is incomplete")
	}
	address, err := accountcredentials.NormalizeIdentitySubject(accountcredentials.IdentityProviderSolanaWallet, value.Address)
	if err != nil || address != value.Address {
		return fmt.Errorf("Worm execution Solana challenge address is invalid")
	}
	if runID, err := canonicalWormExecutionID(value.RunID, "runId"); err != nil || runID != value.RunID {
		return fmt.Errorf("Worm execution Solana challenge Run binding is invalid")
	}
	if commandID, err := canonicalWormExecutionID(value.CommandID, "commandId"); err != nil || commandID != value.CommandID {
		return fmt.Errorf("Worm execution Solana challenge command binding is invalid")
	}
	if accountID, err := accountcredentials.CanonicalAccountID(value.AccountID); err != nil || accountID != value.AccountID {
		return fmt.Errorf("Worm execution Solana challenge account binding is invalid")
	}
	if value.ReturnTo != validateWormExecutionReturnTo(value.ReturnTo, value.RunID) {
		return fmt.Errorf("Worm execution Solana challenge return path is invalid")
	}
	if _, err := canonicalWormExecutionDigest(value.SessionJTIDigestSHA256); err != nil {
		return fmt.Errorf("Worm execution Solana challenge session digest is invalid")
	}
	if _, err := canonicalWormExecutionDigest(value.PlanDigestSHA256); err != nil {
		return fmt.Errorf("Worm execution Solana challenge plan digest is invalid")
	}
	nonce, err := hex.DecodeString(value.Nonce)
	if err != nil || len(nonce) != 16 || hex.EncodeToString(nonce) != value.Nonce {
		return fmt.Errorf("Worm execution Solana challenge nonce is invalid")
	}
	if !value.ExpiresAt.After(value.CreatedAt) || value.ExpiresAt.Sub(value.CreatedAt) != wormExecutionChallengeTTL {
		return fmt.Errorf("Worm execution Solana challenge lifetime is invalid")
	}
	return nil
}

func wormExecutionChallengeKey(id string) string {
	digest := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%s%x", wormExecutionChallengePrefix, digest[:])
}
