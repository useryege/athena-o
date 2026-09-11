package txgate

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Beginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// AccountSession owns one physical connection until a confirmed advisory unlock.
// It is not shared concurrently; the summary coordinator transfers ownership at
// the actual-start handshake. Acquisition failure also discards the connection:
// a lost server acknowledgement cannot prove the session never took the lock.
type AccountSession struct {
	Conn  *pgxpool.Conn
	owner string
}

func AcquireAccountSession(ctx context.Context, pool *pgxpool.Pool, owner string) (*AccountSession, error) {
	if pool == nil {
		return nil, fmt.Errorf("account session pool is required")
	}
	canonical, e := canonicalAccountID(owner)
	if e != nil {
		return nil, e
	}
	began := time.Now()
	c, e := pool.Acquire(ctx)
	observeTiming(ctx, "session", "begin", began, e)
	if e != nil {
		return nil, e
	}
	began = time.Now()
	_, e = c.Exec(ctx, "SELECT pg_advisory_lock(hashtextextended('athena:account:' || $1::text,0))", canonical)
	observeTiming(ctx, "session", "advisory", began, e)
	if e != nil {
		discardAccountConnection(c)
		return nil, fmt.Errorf("acquire account session: %w", e)
	}
	return &AccountSession{Conn: c, owner: canonical}, nil
}
func (s *AccountSession) Release(ctx context.Context) error {
	if s == nil || s.Conn == nil {
		return nil
	}
	c := s.Conn
	s.Conn = nil
	var unlocked bool
	e := c.QueryRow(ctx, "SELECT pg_advisory_unlock(hashtextextended('athena:account:' || $1::text,0))", s.owner).Scan(&unlocked)
	if e != nil || !unlocked {
		discardAccountConnection(c)
		return fmt.Errorf("release account session (unlocked=%t): %w", unlocked, e)
	}
	c.Release()
	return nil
}
func discardAccountConnection(c *pgxpool.Conn) {
	raw := c.Hijack()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = raw.Close(ctx)
}

func WithAccountTx(ctx context.Context, pool Beginner, accountID string, fn func(pgx.Tx) error) error {
	// Preserve nil pgxpool validation after accepting transaction decorators.
	concretePool, isConcretePool := pool.(*pgxpool.Pool)
	if pool == nil || (isConcretePool && concretePool == nil) {
		return fmt.Errorf("account transaction pool is required")
	}
	canonicalAccountID, err := canonicalAccountID(accountID)
	if err != nil {
		return err
	}
	began := time.Now()
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	observeTiming(ctx, "transaction", "begin", began, err)
	if err != nil {
		return fmt.Errorf("begin account transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(context.Background())
		}
	}()
	began = time.Now()
	_, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended('athena:account:' || $1::text, 0))", canonicalAccountID)
	observeTiming(ctx, "transaction", "advisory", began, err)
	if err != nil {
		return fmt.Errorf("lock account %q: %w", canonicalAccountID, err)
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit account transaction: %w", err)
	}
	committed = true
	return nil
}

func LockWallet(ctx context.Context, tx pgx.Tx, wallet common.Address) error {
	if tx == nil {
		return fmt.Errorf("wallet transaction is required")
	}
	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended('athena:wallet:' || $1::text, 0))", wallet.Hex()); err != nil {
		return fmt.Errorf("lock wallet %q: %w", wallet.Hex(), err)
	}
	return nil
}

func canonicalAccountID(value string) (string, error) {
	if len(value) != 36 {
		return "", fmt.Errorf("account ID %q is not a UUID", value)
	}
	for index, character := range value {
		if index == 8 || index == 13 || index == 18 || index == 23 {
			if character != '-' {
				return "", fmt.Errorf("account ID %q is not a UUID", value)
			}
			continue
		}
		if !(character >= '0' && character <= '9') && !(character >= 'a' && character <= 'f') && !(character >= 'A' && character <= 'F') {
			return "", fmt.Errorf("account ID %q is not a UUID", value)
		}
	}
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil {
		return "", fmt.Errorf("account ID %q is not a non-zero UUID", value)
	}
	return parsed.String(), nil
}

// Timing reports one acquisition phase. Observers must only assign local values;
// they must not perform SQL, logging, blocking sends, or acquire other locks.
type Timing struct {
	Kind, Phase string
	Duration    time.Duration
	Succeeded   bool
}
type timingKey struct{}

func WithTiming(ctx context.Context, observe func(Timing)) context.Context {
	if observe == nil {
		return ctx
	}
	if prior, ok := ctx.Value(timingKey{}).(func(Timing)); ok {
		next := observe
		observe = func(v Timing) { prior(v); next(v) }
	}
	return context.WithValue(ctx, timingKey{}, observe)
}
func observeTiming(ctx context.Context, kind, phase string, began time.Time, err error) {
	if observe, ok := ctx.Value(timingKey{}).(func(Timing)); ok {
		observe(Timing{Kind: kind, Phase: phase, Duration: time.Since(began), Succeeded: err == nil})
	}
}
