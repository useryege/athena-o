package authregistration

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
	registrationPrefix      = "auth-registration|"
	registrationClaimPrefix = "auth-registration-claim|"
	registrationTTL         = 15 * time.Minute
	registrationClaimTTL    = time.Minute
)

var (
	ErrRegistrationNotFound    = errors.New("registration is missing or expired")
	ErrRegistrationInProgress  = errors.New("registration is already being submitted")
	ErrRegistrationUnavailable = errors.New("registration storage is unavailable")
)

// Ticket holds a verified identity while its browser chooses a permanent
// username. It is stored only in Redis and never sent to the browser verbatim.
type Ticket struct {
	Identity   Identity  `json:"identity"`
	ReturnTo   string    `json:"returnTo"`
	CSRFSecret string    `json:"csrfSecret"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Store owns short-lived, one-browser registration tickets and submission
// claims. Ticket identifiers and claim owners are opaque random values.
type Store struct {
	redis *redis.Client
}

func NewStore(redisClient *redis.Client) (*Store, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("registration Redis client is required")
	}
	return &Store{redis: redisClient}, nil
}

func (s *Store) Create(ctx context.Context, ticketID string, ticket Ticket) error {
	if err := validateTicket(ticket); err != nil {
		return err
	}
	encoded, err := json.Marshal(ticket)
	if err != nil {
		return fmt.Errorf("encode registration ticket: %w", err)
	}
	created, err := s.redis.SetNX(ctx, registrationKey(ticketID), encoded, registrationTTL).Result()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRegistrationUnavailable, err)
	}
	if !created {
		return fmt.Errorf("registration ticket collision")
	}
	return nil
}

func (s *Store) Get(ctx context.Context, ticketID string) (Ticket, error) {
	value, err := s.redis.Get(ctx, registrationKey(ticketID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return Ticket{}, ErrRegistrationNotFound
	}
	if err != nil {
		return Ticket{}, fmt.Errorf("%w: %v", ErrRegistrationUnavailable, err)
	}
	return decodeTicket(value)
}

// Claim serializes submissions for one browser ticket. The decoded ticket is
// checked for freshness after the atomic claim, so stale values cannot proceed
// even if a Redis TTL is unexpectedly extended.
func (s *Store) Claim(ctx context.Context, ticketID, claimID string) (Ticket, error) {
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
		return Ticket{}, fmt.Errorf("%w: %v", ErrRegistrationUnavailable, err)
	}
	switch value {
	case "__ATHENA_REGISTRATION_MISSING__":
		return Ticket{}, ErrRegistrationNotFound
	case "__ATHENA_REGISTRATION_BUSY__":
		return Ticket{}, ErrRegistrationInProgress
	}
	ticket, err := decodeTicket([]byte(value))
	if err != nil {
		_ = s.ReleaseClaim(ctx, ticketID, claimID)
		return Ticket{}, err
	}
	return ticket, nil
}

// ReleaseClaim is owner-safe: a slow request cannot release a newer claim
// after its own lease expires.
func (s *Store) ReleaseClaim(ctx context.Context, ticketID, claimID string) error {
	const script = `
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`
	if _, err := s.redis.Eval(ctx, script, []string{registrationClaimKey(ticketID)}, claimID).Result(); err != nil {
		return fmt.Errorf("%w: %v", ErrRegistrationUnavailable, err)
	}
	return nil
}

// Complete consumes a ticket only when claimID still owns its live claim.
// This makes a database commit followed by an expired lease recoverable through
// the provider's known-identity login path without deleting another request's
// ticket.
func (s *Store) Complete(ctx context.Context, ticketID, claimID string) error {
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
		return fmt.Errorf("%w: %v", ErrRegistrationUnavailable, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return ErrRegistrationInProgress
	case -1:
		return ErrRegistrationNotFound
	default:
		return fmt.Errorf("%w: unexpected completion result %d", ErrRegistrationUnavailable, result)
	}
}

func (s *Store) Delete(ctx context.Context, ticketID string) error {
	const script = `
if not redis.call("GET", KEYS[1]) then
  return -1
end
if redis.call("GET", KEYS[2]) then
  return 0
end
redis.call("DEL", KEYS[1])
return 1
`
	result, err := s.redis.Eval(
		ctx,
		script,
		[]string{registrationKey(ticketID), registrationClaimKey(ticketID)},
	).Int64()
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRegistrationUnavailable, err)
	}
	switch result {
	case 1:
		return nil
	case 0:
		return ErrRegistrationInProgress
	case -1:
		return ErrRegistrationNotFound
	default:
		return fmt.Errorf("%w: unexpected deletion result %d", ErrRegistrationUnavailable, result)
	}
}

func registrationKey(ticketID string) string {
	digest := sha256.Sum256([]byte(ticketID))
	return fmt.Sprintf("%s%x", registrationPrefix, digest[:])
}

func registrationClaimKey(ticketID string) string {
	digest := sha256.Sum256([]byte(ticketID))
	return fmt.Sprintf("%s%x", registrationClaimPrefix, digest[:])
}

func decodeTicket(value []byte) (Ticket, error) {
	var ticket Ticket
	if err := json.Unmarshal(value, &ticket); err != nil {
		return Ticket{}, fmt.Errorf("decode registration ticket: %w", err)
	}
	if err := validateTicket(ticket); err != nil {
		return Ticket{}, err
	}
	if !registrationFresh(ticket.CreatedAt) {
		return Ticket{}, ErrRegistrationNotFound
	}
	return ticket, nil
}

func validateTicket(ticket Ticket) error {
	if err := ticket.Identity.Validate(); err != nil {
		return fmt.Errorf("registration ticket has an invalid identity: %w", err)
	}
	if ticket.ReturnTo == "" || ticket.ReturnTo != ValidateReturnTo(ticket.ReturnTo) || ticket.CSRFSecret == "" || ticket.CreatedAt.IsZero() {
		return fmt.Errorf("registration ticket is incomplete")
	}
	return nil
}

func registrationFresh(createdAt time.Time) bool {
	now := time.Now().UTC()
	return !createdAt.After(now.Add(time.Minute)) && now.Sub(createdAt) <= registrationTTL
}
