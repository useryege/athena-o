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
	wormPositionCashOutChallengePrefix            = "worm-position-cash-out-solana-proof|"
	wormPositionCashOutChallengeTTL               = 5 * time.Minute
	wormPositionCashOutChallengeRateGlobalKey     = "worm-position-cash-out-solana-proof-rate|global"
	wormPositionCashOutChallengeRateAccountPrefix = "worm-position-cash-out-solana-proof-rate|account|"
	wormPositionCashOutChallengeRateWindow        = time.Minute
	wormPositionCashOutChallengeRateGlobalLimit   = int64(120)
	wormPositionCashOutChallengeRateAccountLimit  = int64(20)
)

var (
	errWormPositionCashOutChallengeNotFound = errors.New(
		"Worm position Cash Out Solana challenge is missing or expired",
	)
	errWormPositionCashOutChallengeUnavailable = errors.New(
		"Worm position Cash Out Solana challenge storage is unavailable",
	)
)

type wormPositionCashOutChallenge struct {
	Address                string    `json:"address"`
	Message                string    `json:"message"`
	Nonce                  string    `json:"nonce"`
	ReturnTo               string    `json:"returnTo"`
	CashOutID              string    `json:"cashOutId"`
	CommandID              string    `json:"commandId"`
	ExpectedRevision       int64     `json:"expectedRevision"`
	AccountID              string    `json:"accountId"`
	SessionJTIDigestSHA256 string    `json:"sessionJtiDigestSha256"`
	AccessRevision         uint64    `json:"accessRevision"`
	IntentDigestSHA256     string    `json:"intentDigestSha256"`
	CreatedAt              time.Time `json:"createdAt"`
	ExpiresAt              time.Time `json:"expiresAt"`
}

type wormPositionCashOutChallengeStore struct {
	redis *redis.Client
}

func newWormPositionCashOutChallengeStore(
	redisClient *redis.Client,
) (*wormPositionCashOutChallengeStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Worm position Cash Out Solana Redis client is required")
	}
	return &wormPositionCashOutChallengeStore{redis: redisClient}, nil
}

func (s *wormPositionCashOutChallengeStore) create(
	ctx context.Context,
	id string,
	value wormPositionCashOutChallenge,
) error {
	if !authregistration.ValidOpaqueValue(id) {
		return fmt.Errorf("Worm position Cash Out Solana challenge ID is invalid")
	}
	if err := validateWormPositionCashOutChallenge(value); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Worm position Cash Out Solana challenge: %w", err)
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
			wormPositionCashOutChallengeRateGlobalKey,
			wormPositionCashOutChallengeRateAccountPrefix + hex.EncodeToString(accountDigest[:]),
			wormPositionCashOutChallengeKey(id),
		},
		encoded,
		int64(wormPositionCashOutChallengeRateWindow/time.Second),
		wormPositionCashOutChallengeRateGlobalLimit,
		wormPositionCashOutChallengeRateAccountLimit,
		int64(wormPositionCashOutChallengeTTL/time.Second),
	).Int64()
	if err != nil {
		return fmt.Errorf("%w: %v", errWormPositionCashOutChallengeUnavailable, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("%w: proof challenge creation is rate limited", errWormPositionCashOutChallengeUnavailable)
	case -1:
		return fmt.Errorf("%w: proof challenge state collision", errWormPositionCashOutChallengeUnavailable)
	default:
		return fmt.Errorf("%w: unexpected challenge result %d", errWormPositionCashOutChallengeUnavailable, result)
	}
}

func (s *wormPositionCashOutChallengeStore) consume(
	ctx context.Context,
	id string,
) (wormPositionCashOutChallenge, error) {
	if !authregistration.ValidOpaqueValue(id) {
		return wormPositionCashOutChallenge{}, errWormPositionCashOutChallengeNotFound
	}
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	encoded, err := s.redis.Eval(ctx, script, []string{wormPositionCashOutChallengeKey(id)}).Text()
	if errors.Is(err, redis.Nil) {
		return wormPositionCashOutChallenge{}, errWormPositionCashOutChallengeNotFound
	}
	if err != nil {
		return wormPositionCashOutChallenge{}, fmt.Errorf("%w: %v", errWormPositionCashOutChallengeUnavailable, err)
	}
	var value wormPositionCashOutChallenge
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return wormPositionCashOutChallenge{}, fmt.Errorf("decode Worm position Cash Out Solana challenge: %w", err)
	}
	if err := validateWormPositionCashOutChallenge(value); err != nil {
		return wormPositionCashOutChallenge{}, errWormPositionCashOutChallengeNotFound
	}
	now := time.Now().UTC()
	if value.CreatedAt.After(now.Add(time.Minute)) || !value.ExpiresAt.After(now) {
		return wormPositionCashOutChallenge{}, errWormPositionCashOutChallengeNotFound
	}
	return value, nil
}

func validateWormPositionCashOutChallenge(value wormPositionCashOutChallenge) error {
	if value.Address == "" || value.Message == "" || value.Nonce == "" || value.ReturnTo == "" ||
		value.CashOutID == "" || value.CommandID == "" || value.ExpectedRevision <= 0 || value.AccountID == "" ||
		value.SessionJTIDigestSHA256 == "" || value.AccessRevision == 0 || value.IntentDigestSHA256 == "" ||
		value.CreatedAt.IsZero() || value.ExpiresAt.IsZero() {
		return fmt.Errorf("Worm position Cash Out Solana challenge is incomplete")
	}
	address, err := accountcredentials.NormalizeIdentitySubject(
		accountcredentials.IdentityProviderSolanaWallet,
		value.Address,
	)
	if err != nil || address != value.Address {
		return fmt.Errorf("Worm position Cash Out Solana challenge address is invalid")
	}
	if cashOutID, err := canonicalWormPositionCashOutID(value.CashOutID, "cashOutId"); err != nil || cashOutID != value.CashOutID {
		return fmt.Errorf("Worm position Cash Out Solana challenge operation binding is invalid")
	}
	if commandID, err := canonicalWormPositionCashOutID(value.CommandID, "commandId"); err != nil || commandID != value.CommandID {
		return fmt.Errorf("Worm position Cash Out Solana challenge command binding is invalid")
	}
	if accountID, err := accountcredentials.CanonicalAccountID(value.AccountID); err != nil || accountID != value.AccountID {
		return fmt.Errorf("Worm position Cash Out Solana challenge account binding is invalid")
	}
	if value.ReturnTo != validateWormPositionCashOutReturnTo(value.ReturnTo) {
		return fmt.Errorf("Worm position Cash Out Solana challenge return path is invalid")
	}
	if _, err := canonicalWormPositionCashOutDigest(value.SessionJTIDigestSHA256); err != nil {
		return fmt.Errorf("Worm position Cash Out Solana challenge session digest is invalid")
	}
	if _, err := canonicalWormPositionCashOutDigest(value.IntentDigestSHA256); err != nil {
		return fmt.Errorf("Worm position Cash Out Solana challenge intent digest is invalid")
	}
	nonce, err := hex.DecodeString(value.Nonce)
	if err != nil || len(nonce) != 16 || hex.EncodeToString(nonce) != value.Nonce {
		return fmt.Errorf("Worm position Cash Out Solana challenge nonce is invalid")
	}
	if !value.ExpiresAt.After(value.CreatedAt) ||
		value.ExpiresAt.Sub(value.CreatedAt) != wormPositionCashOutChallengeTTL {
		return fmt.Errorf("Worm position Cash Out Solana challenge lifetime is invalid")
	}
	return nil
}

func wormPositionCashOutChallengeKey(id string) string {
	digest := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%s%x", wormPositionCashOutChallengePrefix, digest[:])
}
