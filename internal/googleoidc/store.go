package googleoidc

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	transactionPrefix          = "google-oidc-transaction|"
	transactionRateGlobalKey   = "google-oidc-login-rate|global"
	transactionRateClientKey   = "google-oidc-login-rate|client|"
	transactionTTL             = 5 * time.Minute
	transactionRateWindow      = time.Minute
	transactionGlobalRateLimit = 120
	transactionClientRateLimit = 20
	registrationPrefix         = "google-oidc-registration|"
	registrationClaimPrefix    = "google-oidc-registration-claim|"
	registrationTTL            = 15 * time.Minute
	registrationClaimTTL       = time.Minute
)

var (
	errTransactionNotFound     = errors.New("Google OIDC transaction is missing or expired")
	errTransactionUnavailable  = errors.New("Google OIDC transaction storage is unavailable")
	errTransactionRateLimited  = errors.New("Google OIDC login rate limit exceeded")
	errRegistrationNotFound    = errors.New("Google registration is missing or expired")
	errRegistrationInProgress  = errors.New("Google registration is already being submitted")
	errRegistrationUnavailable = errors.New("Google registration storage is unavailable")
)

type transaction struct {
	Nonce     string    `json:"nonce"`
	Verifier  string    `json:"verifier"`
	ReturnTo  string    `json:"returnTo"`
	CreatedAt time.Time `json:"createdAt"`
}

// registrationTicket contains a verified Google identity while the user
// chooses the immutable Athena username. It never crosses the public API.
type registrationTicket struct {
	Subject                string    `json:"subject"`
	VerifiedEmail          string    `json:"verifiedEmail"`
	AdministratorCandidate bool      `json:"administratorCandidate"`
	ReturnTo               string    `json:"returnTo"`
	CSRFSecret             string    `json:"csrfSecret"`
	CreatedAt              time.Time `json:"createdAt"`
}

// TransactionStore owns short-lived, one-time Google OAuth transactions.
type TransactionStore struct {
	redis *redis.Client
}

func NewTransactionStore(redisClient *redis.Client) (*TransactionStore, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("Google OIDC transaction Redis client is required")
	}
	return &TransactionStore{redis: redisClient}, nil
}

func (s *TransactionStore) Create(ctx context.Context, state, clientRateKey string, value transaction) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Google OIDC transaction: %w", err)
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
			transactionRateGlobalKey,
			transactionRateClientKey + clientRateKey,
			transactionPrefix + state,
		},
		encoded,
		int64(transactionRateWindow/time.Second),
		transactionGlobalRateLimit,
		transactionClientRateLimit,
		int64(transactionTTL/time.Second),
	).Int64()
	if err != nil {
		return fmt.Errorf("store Google OIDC transaction: %w", err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return errTransactionRateLimited
	case -1:
		return fmt.Errorf("Google OIDC state collision")
	default:
		return fmt.Errorf("unexpected Google OIDC transaction result %d", result)
	}
}

// Consume atomically reads and removes a transaction so every callback,
// including failures, makes the state unusable for replay.
func (s *TransactionStore) Consume(ctx context.Context, state string) (transaction, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if value then
  redis.call("DEL", KEYS[1])
end
return value
`
	value, err := s.redis.Eval(ctx, script, []string{transactionPrefix + state}).Text()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return transaction{}, errTransactionNotFound
		}
		return transaction{}, fmt.Errorf("%w: %v", errTransactionUnavailable, err)
	}
	var decoded transaction
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return transaction{}, fmt.Errorf("decode Google OIDC transaction: %w", err)
	}
	if decoded.Nonce == "" || decoded.Verifier == "" || decoded.ReturnTo == "" || decoded.CreatedAt.IsZero() {
		return transaction{}, fmt.Errorf("Google OIDC transaction is incomplete")
	}
	return decoded, nil
}

// CreateRegistration stores a verified identity without creating an Athena
// account. The browser receives only the opaque ticket identifier.
func (s *TransactionStore) CreateRegistration(ctx context.Context, ticketID string, value registrationTicket) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode Google registration: %w", err)
	}
	created, err := s.redis.SetNX(ctx, registrationKey(ticketID), encoded, registrationTTL).Result()
	if err != nil {
		return fmt.Errorf("%w: %v", errRegistrationUnavailable, err)
	}
	if !created {
		return fmt.Errorf("Google registration ticket collision")
	}
	return nil
}

// GetRegistration reads a registration ticket without consuming it. The
// final successful account creation deletes it explicitly.
func (s *TransactionStore) GetRegistration(ctx context.Context, ticketID string) (registrationTicket, error) {
	value, err := s.redis.Get(ctx, registrationKey(ticketID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return registrationTicket{}, errRegistrationNotFound
	}
	if err != nil {
		return registrationTicket{}, fmt.Errorf("%w: %v", errRegistrationUnavailable, err)
	}
	var decoded registrationTicket
	if err := json.Unmarshal(value, &decoded); err != nil {
		return registrationTicket{}, fmt.Errorf("decode Google registration: %w", err)
	}
	if decoded.Subject == "" || decoded.VerifiedEmail == "" || decoded.ReturnTo == "" || decoded.CSRFSecret == "" || decoded.CreatedAt.IsZero() {
		return registrationTicket{}, fmt.Errorf("Google registration ticket is incomplete")
	}
	if !registrationFresh(decoded.CreatedAt) {
		_ = s.redis.Del(ctx, registrationKey(ticketID)).Err()
		return registrationTicket{}, errRegistrationNotFound
	}
	return decoded, nil
}

// ClaimRegistration serializes submissions for one browser ticket while
// leaving validation and database conflicts retryable. claimID is an opaque
// per-request owner token used to prevent a slow request from releasing a
// newer claim after its own lease expires.
func (s *TransactionStore) ClaimRegistration(ctx context.Context, ticketID, claimID string) (registrationTicket, error) {
	const script = `
local value = redis.call("GET", KEYS[1])
if not value then
  return "__ATHENA_REGISTRATION_MISSING__"
end
local claimed = redis.call("SET", KEYS[2], ARGV[1], "EX", ARGV[2], "NX")
if not claimed then
  return "__ATHENA_REGISTRATION_BUSY__"
end
return value
`
	value, err := s.redis.Eval(
		ctx,
		script,
		[]string{registrationKey(ticketID), registrationClaimKey(ticketID)},
		claimID,
		int64(registrationClaimTTL/time.Second),
	).Text()
	if err != nil {
		return registrationTicket{}, fmt.Errorf("%w: %v", errRegistrationUnavailable, err)
	}
	switch value {
	case "__ATHENA_REGISTRATION_MISSING__":
		return registrationTicket{}, errRegistrationNotFound
	case "__ATHENA_REGISTRATION_BUSY__":
		return registrationTicket{}, errRegistrationInProgress
	}
	ticket, decodeErr := decodeRegistrationTicket([]byte(value))
	if decodeErr != nil {
		_ = s.ReleaseRegistrationClaim(ctx, ticketID, claimID)
		return registrationTicket{}, decodeErr
	}
	return ticket, nil
}

// ReleaseRegistrationClaim makes a failed, retryable submission available
// again without allowing one request to release another request's lease.
func (s *TransactionStore) ReleaseRegistrationClaim(ctx context.Context, ticketID, claimID string) error {
	const script = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`
	if _, err := s.redis.Eval(ctx, script, []string{registrationClaimKey(ticketID)}, claimID).Result(); err != nil {
		return fmt.Errorf("%w: %v", errRegistrationUnavailable, err)
	}
	return nil
}

// CompleteRegistration consumes the browser ticket after the account
// aggregate commits. Only the current claim owner may consume it, so an
// expired slow request cannot delete a ticket claimed by a newer submission.
func (s *TransactionStore) CompleteRegistration(ctx context.Context, ticketID, claimID string) error {
	const script = `
if not redis.call("GET", KEYS[1]) then
  return -1
end
if redis.call("GET", KEYS[2]) ~= ARGV[1] then
  return 0
end
redis.call("DEL", KEYS[1], KEYS[2])
return 1
`
	result, err := s.redis.Eval(
		ctx,
		script,
		[]string{registrationKey(ticketID), registrationClaimKey(ticketID)},
		claimID,
	).Int64()
	if err != nil {
		return fmt.Errorf("%w: %v", errRegistrationUnavailable, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return errRegistrationInProgress
	case -1:
		return errRegistrationNotFound
	default:
		return fmt.Errorf("%w: unexpected completion result %d", errRegistrationUnavailable, result)
	}
}

// DeleteRegistration makes a registration ticket unusable for replay.
func (s *TransactionStore) DeleteRegistration(ctx context.Context, ticketID string) error {
	if err := s.redis.Del(ctx, registrationKey(ticketID), registrationClaimKey(ticketID)).Err(); err != nil {
		return fmt.Errorf("%w: %v", errRegistrationUnavailable, err)
	}
	return nil
}

func registrationKey(ticketID string) string {
	digest := sha256.Sum256([]byte(ticketID))
	return fmt.Sprintf("%s%x", registrationPrefix, digest[:])
}

func registrationClaimKey(ticketID string) string {
	digest := sha256.Sum256([]byte(ticketID))
	return fmt.Sprintf("%s%x", registrationClaimPrefix, digest[:])
}

func decodeRegistrationTicket(value []byte) (registrationTicket, error) {
	var decoded registrationTicket
	if err := json.Unmarshal(value, &decoded); err != nil {
		return registrationTicket{}, fmt.Errorf("decode Google registration: %w", err)
	}
	if decoded.Subject == "" || decoded.VerifiedEmail == "" || decoded.ReturnTo == "" || decoded.CSRFSecret == "" || decoded.CreatedAt.IsZero() {
		return registrationTicket{}, fmt.Errorf("Google registration ticket is incomplete")
	}
	if !registrationFresh(decoded.CreatedAt) {
		return registrationTicket{}, errRegistrationNotFound
	}
	return decoded, nil
}

func registrationFresh(createdAt time.Time) bool {
	now := time.Now().UTC()
	return !createdAt.After(now.Add(time.Minute)) && now.Sub(createdAt) <= registrationTTL
}
