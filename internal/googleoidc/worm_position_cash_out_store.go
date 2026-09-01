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
	wormPositionCashOutStatePrefix             = "wco."
	wormPositionCashOutTransactionPrefix       = "worm-position-cash-out-google-proof|"
	wormPositionCashOutTransactionTTL          = 5 * time.Minute
	wormPositionCashOutRateGlobalKey           = "worm-position-cash-out-google-proof-rate|global"
	wormPositionCashOutRateAccountPrefix       = "worm-position-cash-out-google-proof-rate|account|"
	wormPositionCashOutRateWindow              = time.Minute
	wormPositionCashOutRateGlobalLimit   int64 = 120
	wormPositionCashOutRateAccountLimit  int64 = 20
)

var (
	errWormPositionCashOutTransactionNotFound = errors.New(
		"Worm position Cash Out Google transaction is missing or expired",
	)
	errWormPositionCashOutTransactionUnavailable = errors.New(
		"Worm position Cash Out Google transaction storage is unavailable",
	)
)

type wormPositionCashOutTransaction struct {
	Nonce                  string    `json:"nonce"`
	Verifier               string    `json:"verifier"`
	ReturnTo               string    `json:"returnTo"`
	CashOutID              string    `json:"cashOutId"`
	CommandID              string    `json:"commandId"`
	ExpectedRevision       int64     `json:"expectedRevision"`
	AccountID              string    `json:"accountId"`
	SessionJTIDigestSHA256 string    `json:"sessionJtiDigestSha256"`
	AccessRevision         uint64    `json:"accessRevision"`
	IntentDigestSHA256     string    `json:"intentDigestSha256"`
	CreatedAt              time.Time `json:"createdAt"`
}

type wormPositionCashOutTransactionStore struct {
	redis *redis.Client
}

func newWormPositionCashOutTransactionStore(
	redisClient *redis.Client,
) (*wormPositionCashOutTransactionStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Worm position Cash Out Google Redis client is required")
	}
	return &wormPositionCashOutTransactionStore{redis: redisClient}, nil
}

func (s *wormPositionCashOutTransactionStore) create(
	ctx context.Context,
	state string,
	value wormPositionCashOutTransaction,
) error {
	if err := validateWormPositionCashOutTransaction(state, value); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Worm position Cash Out Google transaction: %w", err)
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
			wormPositionCashOutRateGlobalKey,
			wormPositionCashOutRateAccountPrefix + hex.EncodeToString(accountDigest[:]),
			wormPositionCashOutTransactionKey(state),
		},
		encoded,
		int64(wormPositionCashOutRateWindow/time.Second),
		wormPositionCashOutRateGlobalLimit,
		wormPositionCashOutRateAccountLimit,
		int64(wormPositionCashOutTransactionTTL/time.Second),
	).Int64()
	if err != nil {
		return fmt.Errorf("%w: %v", errWormPositionCashOutTransactionUnavailable, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return fmt.Errorf("%w: proof transaction creation is rate limited", errWormPositionCashOutTransactionUnavailable)
	case -1:
		return fmt.Errorf("%w: proof transaction state collision", errWormPositionCashOutTransactionUnavailable)
	default:
		return fmt.Errorf("%w: unexpected transaction result %d", errWormPositionCashOutTransactionUnavailable, result)
	}
}

func (s *wormPositionCashOutTransactionStore) consume(
	ctx context.Context,
	state string,
) (wormPositionCashOutTransaction, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	encoded, err := s.redis.Eval(ctx, script, []string{wormPositionCashOutTransactionKey(state)}).Text()
	if errors.Is(err, redis.Nil) {
		return wormPositionCashOutTransaction{}, errWormPositionCashOutTransactionNotFound
	}
	if err != nil {
		return wormPositionCashOutTransaction{}, fmt.Errorf("%w: %v", errWormPositionCashOutTransactionUnavailable, err)
	}
	var value wormPositionCashOutTransaction
	if err := json.Unmarshal([]byte(encoded), &value); err != nil {
		return wormPositionCashOutTransaction{}, fmt.Errorf("decode Worm position Cash Out Google transaction: %w", err)
	}
	if err := validateWormPositionCashOutTransaction(state, value); err != nil {
		return wormPositionCashOutTransaction{}, errWormPositionCashOutTransactionNotFound
	}
	now := time.Now().UTC()
	if value.CreatedAt.After(now.Add(time.Minute)) || now.Sub(value.CreatedAt) > wormPositionCashOutTransactionTTL {
		return wormPositionCashOutTransaction{}, errWormPositionCashOutTransactionNotFound
	}
	return value, nil
}

func validateWormPositionCashOutTransaction(
	state string,
	value wormPositionCashOutTransaction,
) error {
	if !validWormPositionCashOutState(state) || value.Nonce == "" || value.Verifier == "" ||
		value.ReturnTo == "" || value.CashOutID == "" || value.CommandID == "" ||
		value.ExpectedRevision <= 0 || value.AccountID == "" || value.SessionJTIDigestSHA256 == "" ||
		value.AccessRevision == 0 || value.IntentDigestSHA256 == "" || value.CreatedAt.IsZero() {
		return fmt.Errorf("Worm position Cash Out Google transaction is incomplete")
	}
	if _, err := canonicalWormPositionCashOutDigest(value.SessionJTIDigestSHA256); err != nil {
		return fmt.Errorf("Worm position Cash Out Google transaction session digest is invalid")
	}
	if _, err := canonicalWormPositionCashOutDigest(value.IntentDigestSHA256); err != nil {
		return fmt.Errorf("Worm position Cash Out Google transaction intent digest is invalid")
	}
	if cashOutID, err := canonicalWormPositionCashOutID(value.CashOutID, "cashOutId"); err != nil || cashOutID != value.CashOutID {
		return fmt.Errorf("Worm position Cash Out Google transaction operation binding is invalid")
	}
	if commandID, err := canonicalWormPositionCashOutID(value.CommandID, "commandId"); err != nil || commandID != value.CommandID {
		return fmt.Errorf("Worm position Cash Out Google transaction command binding is invalid")
	}
	if accountID, err := accountcredentials.CanonicalAccountID(value.AccountID); err != nil || accountID != value.AccountID {
		return fmt.Errorf("Worm position Cash Out Google transaction account binding is invalid")
	}
	if value.ReturnTo != validateWormPositionCashOutReturnTo(value.ReturnTo) {
		return fmt.Errorf("Worm position Cash Out Google transaction return path is invalid")
	}
	return nil
}

func newWormPositionCashOutState() (string, error) {
	opaque, err := authregistration.RandomOpaqueValue()
	if err != nil {
		return "", err
	}
	return wormPositionCashOutStatePrefix + opaque, nil
}

func validWormPositionCashOutState(value string) bool {
	return strings.HasPrefix(value, wormPositionCashOutStatePrefix) &&
		authregistration.ValidOpaqueValue(strings.TrimPrefix(value, wormPositionCashOutStatePrefix))
}

func wormPositionCashOutTransactionKey(state string) string {
	digest := sha256.Sum256([]byte(state))
	return fmt.Sprintf("%s%x", wormPositionCashOutTransactionPrefix, digest[:])
}
