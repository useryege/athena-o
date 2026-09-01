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
	wormPositionCashOutBatchChallengePrefix            = "worm-position-cash-out-batch-solana-proof|"
	wormPositionCashOutBatchChallengeTTL               = 5 * time.Minute
	wormPositionCashOutBatchChallengeRateGlobalKey     = "worm-position-cash-out-batch-solana-proof-rate|global"
	wormPositionCashOutBatchChallengeRateAccountPrefix = "worm-position-cash-out-batch-solana-proof-rate|account|"
	wormPositionCashOutBatchChallengeRateWindow        = time.Minute
	wormPositionCashOutBatchChallengeRateGlobalLimit   = int64(120)
	wormPositionCashOutBatchChallengeRateAccountLimit  = int64(20)
)

var (
	errWormPositionCashOutBatchChallengeNotFound = errors.New(
		"Worm position Cash Out Batch Solana challenge is missing or expired",
	)
	errWormPositionCashOutBatchChallengeUnavailable = errors.New(
		"Worm position Cash Out Batch Solana challenge storage is unavailable",
	)
)

type wormPositionCashOutBatchChallenge struct {
	Address                string    `json:"address"`
	Message                string    `json:"message"`
	Nonce                  string    `json:"nonce"`
	ReturnTo               string    `json:"returnTo"`
	BatchID                string    `json:"batchId"`
	CommandID              string    `json:"commandId"`
	ExpectedRevision       int64     `json:"expectedRevision"`
	AccountID              string    `json:"accountId"`
	SessionJTIDigestSHA256 string    `json:"sessionJtiDigestSha256"`
	AccessRevision         uint64    `json:"accessRevision"`
	IntentDigestSHA256     string    `json:"intentDigestSha256"`
	CreatedAt              time.Time `json:"createdAt"`
	ExpiresAt              time.Time `json:"expiresAt"`
}

type wormPositionCashOutBatchChallengeStore struct {
	redis *redis.Client
}

func newWormPositionCashOutBatchChallengeStore(
	redisClient *redis.Client,
) (*wormPositionCashOutBatchChallengeStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Worm position Cash Out Batch Solana Redis client is required")
	}
	return &wormPositionCashOutBatchChallengeStore{redis: redisClient}, nil
}

func (s *wormPositionCashOutBatchChallengeStore) create(
	ctx context.Context,
	id string,
	value wormPositionCashOutBatchChallenge,
) error {
	if !authregistration.ValidOpaqueValue(id) {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge ID is invalid")
	}
	if err := validateWormPositionCashOutBatchChallenge(value); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Worm position Cash Out Batch Solana challenge: %w", err)
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
			wormPositionCashOutBatchChallengeRateGlobalKey,
			wormPositionCashOutBatchChallengeRateAccountPrefix + hex.EncodeToString(accountDigest[:]),
			wormPositionCashOutBatchChallengeKey(id),
		},
		encoded,
		int64(wormPositionCashOutBatchChallengeRateWindow/time.Second),
		wormPositionCashOutBatchChallengeRateGlobalLimit,
		wormPositionCashOutBatchChallengeRateAccountLimit,
		int64(wormPositionCashOutBatchChallengeTTL/time.Second),
	).Int64()
	if err != nil {
		return fmt.Errorf("%w: %v", errWormPositionCashOutBatchChallengeUnavailable, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("%w: proof challenge creation is rate limited", errWormPositionCashOutBatchChallengeUnavailable)
	case -1:
		return fmt.Errorf("%w: proof challenge state collision", errWormPositionCashOutBatchChallengeUnavailable)
	default:
		return fmt.Errorf("%w: unexpected challenge result %d", errWormPositionCashOutBatchChallengeUnavailable, result)
	}
}

func (s *wormPositionCashOutBatchChallengeStore) consume(
	ctx context.Context,
	id string,
) (wormPositionCashOutBatchChallenge, error) {
	if !authregistration.ValidOpaqueValue(id) {
		return wormPositionCashOutBatchChallenge{}, errWormPositionCashOutBatchChallengeNotFound
	}
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	encoded, err := s.redis.Eval(ctx, script, []string{wormPositionCashOutBatchChallengeKey(id)}).Text()
	if errors.Is(err, redis.Nil) {
		return wormPositionCashOutBatchChallenge{}, errWormPositionCashOutBatchChallengeNotFound
	}
	if err != nil {
		return wormPositionCashOutBatchChallenge{}, fmt.Errorf("%w: %v", errWormPositionCashOutBatchChallengeUnavailable, err)
	}
	var value wormPositionCashOutBatchChallenge
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return wormPositionCashOutBatchChallenge{}, fmt.Errorf("decode Worm position Cash Out Batch Solana challenge: %w", err)
	}
	if err := validateWormPositionCashOutBatchChallenge(value); err != nil {
		return wormPositionCashOutBatchChallenge{}, errWormPositionCashOutBatchChallengeNotFound
	}
	now := time.Now().UTC()
	if value.CreatedAt.After(now.Add(time.Minute)) || !value.ExpiresAt.After(now) {
		return wormPositionCashOutBatchChallenge{}, errWormPositionCashOutBatchChallengeNotFound
	}
	return value, nil
}

func validateWormPositionCashOutBatchChallenge(value wormPositionCashOutBatchChallenge) error {
	if value.Address == "" || value.Message == "" || value.Nonce == "" || value.ReturnTo == "" ||
		value.BatchID == "" || value.CommandID == "" || value.ExpectedRevision <= 0 || value.AccountID == "" ||
		value.SessionJTIDigestSHA256 == "" || value.AccessRevision == 0 || value.IntentDigestSHA256 == "" ||
		value.CreatedAt.IsZero() || value.ExpiresAt.IsZero() {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge is incomplete")
	}
	address, err := accountcredentials.NormalizeIdentitySubject(
		accountcredentials.IdentityProviderSolanaWallet,
		value.Address,
	)
	if err != nil || address != value.Address {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge address is invalid")
	}
	if batchID, err := canonicalWormPositionCashOutBatchID(value.BatchID, "batchId"); err != nil || batchID != value.BatchID {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge operation binding is invalid")
	}
	if commandID, err := canonicalWormPositionCashOutBatchID(value.CommandID, "commandId"); err != nil || commandID != value.CommandID {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge command binding is invalid")
	}
	if accountID, err := accountcredentials.CanonicalAccountID(value.AccountID); err != nil || accountID != value.AccountID {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge account binding is invalid")
	}
	if value.ReturnTo != validateWormPositionCashOutBatchReturnTo(value.ReturnTo) {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge return path is invalid")
	}
	if _, err := canonicalWormPositionCashOutBatchDigest(value.SessionJTIDigestSHA256); err != nil {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge session digest is invalid")
	}
	if _, err := canonicalWormPositionCashOutBatchDigest(value.IntentDigestSHA256); err != nil {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge intent digest is invalid")
	}
	nonce, err := hex.DecodeString(value.Nonce)
	if err != nil || len(nonce) != 16 || hex.EncodeToString(nonce) != value.Nonce {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge nonce is invalid")
	}
	if !value.ExpiresAt.After(value.CreatedAt) ||
		value.ExpiresAt.Sub(value.CreatedAt) != wormPositionCashOutBatchChallengeTTL {
		return fmt.Errorf("Worm position Cash Out Batch Solana challenge lifetime is invalid")
	}
	return nil
}

func wormPositionCashOutBatchChallengeKey(id string) string {
	digest := sha256.Sum256([]byte(id))
	return fmt.Sprintf("%s%x", wormPositionCashOutBatchChallengePrefix, digest[:])
}
