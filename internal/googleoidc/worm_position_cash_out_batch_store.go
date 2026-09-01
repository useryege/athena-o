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
	wormPositionCashOutBatchStatePrefix             = "wcob."
	wormPositionCashOutBatchTransactionPrefix       = "worm-position-cash-out-batch-google-proof|"
	wormPositionCashOutBatchTransactionTTL          = 5 * time.Minute
	wormPositionCashOutBatchRateGlobalKey           = "worm-position-cash-out-batch-google-proof-rate|global"
	wormPositionCashOutBatchRateAccountPrefix       = "worm-position-cash-out-batch-google-proof-rate|account|"
	wormPositionCashOutBatchRateWindow              = time.Minute
	wormPositionCashOutBatchRateGlobalLimit   int64 = 120
	wormPositionCashOutBatchRateAccountLimit  int64 = 20
)

var (
	errWormPositionCashOutBatchTransactionNotFound = errors.New(
		"Worm position Cash Out Batch Google transaction is missing or expired",
	)
	errWormPositionCashOutBatchTransactionUnavailable = errors.New(
		"Worm position Cash Out Batch Google transaction storage is unavailable",
	)
)

type wormPositionCashOutBatchTransaction struct {
	Nonce                  string    `json:"nonce"`
	Verifier               string    `json:"verifier"`
	ReturnTo               string    `json:"returnTo"`
	BatchID                string    `json:"batchId"`
	CommandID              string    `json:"commandId"`
	ExpectedRevision       int64     `json:"expectedRevision"`
	AccountID              string    `json:"accountId"`
	SessionJTIDigestSHA256 string    `json:"sessionJtiDigestSha256"`
	AccessRevision         uint64    `json:"accessRevision"`
	IntentDigestSHA256     string    `json:"intentDigestSha256"`
	CreatedAt              time.Time `json:"createdAt"`
}

type wormPositionCashOutBatchTransactionStore struct {
	redis *redis.Client
}

func newWormPositionCashOutBatchTransactionStore(
	redisClient *redis.Client,
) (*wormPositionCashOutBatchTransactionStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Worm position Cash Out Batch Google Redis client is required")
	}
	return &wormPositionCashOutBatchTransactionStore{redis: redisClient}, nil
}

func (s *wormPositionCashOutBatchTransactionStore) create(
	ctx context.Context,
	state string,
	value wormPositionCashOutBatchTransaction,
) error {
	if err := validateWormPositionCashOutBatchTransaction(state, value); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Worm position Cash Out Batch Google transaction: %w", err)
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
			wormPositionCashOutBatchRateGlobalKey,
			wormPositionCashOutBatchRateAccountPrefix + hex.EncodeToString(accountDigest[:]),
			wormPositionCashOutBatchTransactionKey(state),
		},
		encoded,
		int64(wormPositionCashOutBatchRateWindow/time.Second),
		wormPositionCashOutBatchRateGlobalLimit,
		wormPositionCashOutBatchRateAccountLimit,
		int64(wormPositionCashOutBatchTransactionTTL/time.Second),
	).Int64()
	if err != nil {
		return fmt.Errorf("%w: %v", errWormPositionCashOutBatchTransactionUnavailable, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("%w: proof transaction creation is rate limited", errWormPositionCashOutBatchTransactionUnavailable)
	case -1:
		return fmt.Errorf("%w: proof transaction state collision", errWormPositionCashOutBatchTransactionUnavailable)
	default:
		return fmt.Errorf("%w: unexpected transaction result %d", errWormPositionCashOutBatchTransactionUnavailable, result)
	}
}

func (s *wormPositionCashOutBatchTransactionStore) consume(
	ctx context.Context,
	state string,
) (wormPositionCashOutBatchTransaction, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	encoded, err := s.redis.Eval(ctx, script, []string{wormPositionCashOutBatchTransactionKey(state)}).Text()
	if errors.Is(err, redis.Nil) {
		return wormPositionCashOutBatchTransaction{}, errWormPositionCashOutBatchTransactionNotFound
	}
	if err != nil {
		return wormPositionCashOutBatchTransaction{}, fmt.Errorf("%w: %v", errWormPositionCashOutBatchTransactionUnavailable, err)
	}
	var value wormPositionCashOutBatchTransaction
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return wormPositionCashOutBatchTransaction{}, fmt.Errorf("decode Worm position Cash Out Batch Google transaction: %w", err)
	}
	if err := validateWormPositionCashOutBatchTransaction(state, value); err != nil {
		return wormPositionCashOutBatchTransaction{}, errWormPositionCashOutBatchTransactionNotFound
	}
	now := time.Now().UTC()
	if value.CreatedAt.After(now.Add(time.Minute)) || now.Sub(value.CreatedAt) > wormPositionCashOutBatchTransactionTTL {
		return wormPositionCashOutBatchTransaction{}, errWormPositionCashOutBatchTransactionNotFound
	}
	return value, nil
}

func validateWormPositionCashOutBatchTransaction(
	state string,
	value wormPositionCashOutBatchTransaction,
) error {
	if !validWormPositionCashOutBatchState(state) || value.Nonce == "" || value.Verifier == "" ||
		value.ReturnTo == "" || value.BatchID == "" || value.CommandID == "" ||
		value.ExpectedRevision <= 0 || value.AccountID == "" || value.SessionJTIDigestSHA256 == "" ||
		value.AccessRevision == 0 || value.IntentDigestSHA256 == "" || value.CreatedAt.IsZero() {
		return fmt.Errorf("Worm position Cash Out Batch Google transaction is incomplete")
	}
	if _, err := canonicalWormPositionCashOutBatchDigest(value.SessionJTIDigestSHA256); err != nil {
		return fmt.Errorf("Worm position Cash Out Batch Google transaction session digest is invalid")
	}
	if _, err := canonicalWormPositionCashOutBatchDigest(value.IntentDigestSHA256); err != nil {
		return fmt.Errorf("Worm position Cash Out Batch Google transaction intent digest is invalid")
	}
	if batchID, err := canonicalWormPositionCashOutBatchID(value.BatchID, "batchId"); err != nil || batchID != value.BatchID {
		return fmt.Errorf("Worm position Cash Out Batch Google transaction operation binding is invalid")
	}
	if commandID, err := canonicalWormPositionCashOutBatchID(value.CommandID, "commandId"); err != nil || commandID != value.CommandID {
		return fmt.Errorf("Worm position Cash Out Batch Google transaction command binding is invalid")
	}
	if accountID, err := accountcredentials.CanonicalAccountID(value.AccountID); err != nil || accountID != value.AccountID {
		return fmt.Errorf("Worm position Cash Out Batch Google transaction account binding is invalid")
	}
	if value.ReturnTo != validateWormPositionCashOutBatchReturnTo(value.ReturnTo) {
		return fmt.Errorf("Worm position Cash Out Batch Google transaction return path is invalid")
	}
	return nil
}

func newWormPositionCashOutBatchState() (string, error) {
	opaque, err := authregistration.RandomOpaqueValue()
	if err != nil {
		return "", err
	}
	return wormPositionCashOutBatchStatePrefix + opaque, nil
}

func validWormPositionCashOutBatchState(value string) bool {
	return strings.HasPrefix(value, wormPositionCashOutBatchStatePrefix) &&
		authregistration.ValidOpaqueValue(strings.TrimPrefix(value, wormPositionCashOutBatchStatePrefix))
}

func wormPositionCashOutBatchTransactionKey(state string) string {
	digest := sha256.Sum256([]byte(state))
	return fmt.Sprintf("%s%x", wormPositionCashOutBatchTransactionPrefix, digest[:])
}
