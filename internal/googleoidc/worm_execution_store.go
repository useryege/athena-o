package googleoidc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/useryege/athena/internal/accountcredentials"
	"github.com/useryege/athena/internal/authregistration"
)

const (
	wormExecutionStatePrefix             = "wex."
	wormExecutionTransactionPrefix       = "worm-execution-google-proof|"
	wormExecutionTransactionTTL          = 5 * time.Minute
	wormExecutionRateGlobalKey           = "worm-execution-google-proof-rate|global"
	wormExecutionRateAccountPrefix       = "worm-execution-google-proof-rate|account|"
	wormExecutionRateWindow              = time.Minute
	wormExecutionRateGlobalLimit   int64 = 120
	wormExecutionRateAccountLimit  int64 = 20
)

var (
	errWormExecutionTransactionNotFound    = errors.New("Worm execution Google transaction is missing or expired")
	errWormExecutionTransactionUnavailable = errors.New("Worm execution Google transaction storage is unavailable")
)

type wormExecutionTransaction struct {
	Nonce                  string    `json:"nonce"`
	Verifier               string    `json:"verifier"`
	ReturnTo               string    `json:"returnTo"`
	RunID                  string    `json:"runId"`
	CommandID              string    `json:"commandId"`
	ExpectedRevision       int64     `json:"expectedRevision"`
	AccountID              string    `json:"accountId"`
	SessionJTIDigestSHA256 string    `json:"sessionJtiDigestSha256"`
	AccessRevision         uint64    `json:"accessRevision"`
	CreatedAt              time.Time `json:"createdAt"`
}

type wormExecutionTransactionStore struct {
	redis *redis.Client
}

func newWormExecutionTransactionStore(redisClient *redis.Client) (*wormExecutionTransactionStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Worm execution Google Redis client is required")
	}
	return &wormExecutionTransactionStore{redis: redisClient}, nil
}

func (s *wormExecutionTransactionStore) create(ctx context.Context, state string, value wormExecutionTransaction) error {
	if err := validateWormExecutionTransaction(state, value); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Worm execution Google transaction: %w", err)
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
			wormExecutionRateGlobalKey,
			wormExecutionRateAccountPrefix + hex.EncodeToString(accountDigest[:]),
			wormExecutionTransactionKey(state),
		},
		encoded,
		int64(wormExecutionRateWindow/time.Second),
		wormExecutionRateGlobalLimit,
		wormExecutionRateAccountLimit,
		int64(wormExecutionTransactionTTL/time.Second),
	).Int64()
	if err != nil {
		return fmt.Errorf("%w: %v", errWormExecutionTransactionUnavailable, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("%w: proof transaction creation is rate limited", errWormExecutionTransactionUnavailable)
	case -1:
		return fmt.Errorf("%w: proof transaction state collision", errWormExecutionTransactionUnavailable)
	default:
		return fmt.Errorf("%w: unexpected transaction result %d", errWormExecutionTransactionUnavailable, result)
	}
}

func (s *wormExecutionTransactionStore) consume(ctx context.Context, state string) (wormExecutionTransaction, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	encoded, err := s.redis.Eval(ctx, script, []string{wormExecutionTransactionKey(state)}).Text()
	if errors.Is(err, redis.Nil) {
		return wormExecutionTransaction{}, errWormExecutionTransactionNotFound
	}
	if err != nil {
		return wormExecutionTransaction{}, fmt.Errorf("%w: %v", errWormExecutionTransactionUnavailable, err)
	}
	var value wormExecutionTransaction
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return wormExecutionTransaction{}, fmt.Errorf("decode Worm execution Google transaction: %w", err)
	}
	if err := validateWormExecutionTransaction(state, value); err != nil {
		return wormExecutionTransaction{}, errWormExecutionTransactionNotFound
	}
	now := time.Now().UTC()
	if value.CreatedAt.After(now.Add(time.Minute)) || now.Sub(value.CreatedAt) > wormExecutionTransactionTTL {
		return wormExecutionTransaction{}, errWormExecutionTransactionNotFound
	}
	return value, nil
}

func validateWormExecutionTransaction(state string, value wormExecutionTransaction) error {
	if !validWormExecutionState(state) || value.Nonce == "" || value.Verifier == "" || value.ReturnTo == "" ||
		value.RunID == "" || value.CommandID == "" || value.ExpectedRevision <= 0 || value.AccountID == "" ||
		value.SessionJTIDigestSHA256 == "" || value.AccessRevision == 0 || value.CreatedAt.IsZero() {
		return fmt.Errorf("Worm execution Google transaction is incomplete")
	}
	digest, err := hex.DecodeString(value.SessionJTIDigestSHA256)
	if err != nil || len(digest) != sha256.Size || hex.EncodeToString(digest) != value.SessionJTIDigestSHA256 {
		return fmt.Errorf("Worm execution Google transaction session digest is invalid")
	}
	if runID, err := canonicalWormExecutionID(value.RunID, "runId"); err != nil || runID != value.RunID {
		return fmt.Errorf("Worm execution Google transaction Run binding is invalid")
	}
	if commandID, err := canonicalWormExecutionID(value.CommandID, "commandId"); err != nil || commandID != value.CommandID {
		return fmt.Errorf("Worm execution Google transaction command binding is invalid")
	}
	if accountID, err := accountcredentials.CanonicalAccountID(value.AccountID); err != nil || accountID != value.AccountID {
		return fmt.Errorf("Worm execution Google transaction account binding is invalid")
	}
	if value.ReturnTo != validateWormExecutionReturnTo(value.ReturnTo, value.RunID) {
		return fmt.Errorf("Worm execution Google transaction return path is invalid")
	}
	return nil
}

func newWormExecutionState() (string, error) {
	opaque, err := authregistration.RandomOpaqueValue()
	if err != nil {
		return "", err
	}
	return wormExecutionStatePrefix + opaque, nil
}

func validWormExecutionState(value string) bool {
	return strings.HasPrefix(value, wormExecutionStatePrefix) && authregistration.ValidOpaqueValue(strings.TrimPrefix(value, wormExecutionStatePrefix))
}

func wormExecutionTransactionKey(state string) string {
	digest := sha256.Sum256([]byte(state))
	return fmt.Sprintf("%s%x", wormExecutionTransactionPrefix, digest[:])
}
