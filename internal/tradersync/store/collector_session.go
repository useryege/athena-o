package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
)

var ErrCollectorFenced = errors.New("collector no longer owns this epoch")
var ErrBaselineChanged = errors.New("baseline intent or boundary changed")

// CollectorSession owns one dedicated PostgreSQL connection. There is no TTL:
// replacement requires acquiring its session lock, then advancing the durable token.
// The caller must stop and join its receiver/writers before Close releases ownership.
type CollectorSession struct {
	Token   uint64
	mu      sync.Mutex
	conn    *pgxpool.Conn
	store   *SQLStore
	unbound []pgtype.UUID
}

const collectorLock = "athena:trader-sync:collector"

func (s *SQLStore) AcquireCollectorSession(ctx context.Context) (*CollectorSession, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	owned := false
	defer func() {
		if !owned {
			_ = conn.Conn().Close(context.Background())
			conn.Release()
		}
	}()
	var locked bool
	if err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock(hashtextextended($1,0))", collectorLock).Scan(&locked); err != nil {
		return nil, err
	}
	if !locked {
		return nil, errors.New("another collector owns the database")
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	queries := q.New(tx)
	control, err := queries.LockCollectorControl(ctx)
	if err != nil {
		return nil, err
	}
	if control.ActiveEpoch.Valid {
		if err = endEpoch(ctx, queries, control.ActiveEpoch.Int64, "collector_replaced"); err != nil {
			return nil, err
		}
	}
	control, err = queries.ClaimCollectorOwnership(ctx, pgtype.UUID{Bytes: uuid.New(), Valid: true})
	if err != nil {
		return nil, err
	}
	pending, err := queries.ListPendingBaselines(ctx)
	if err != nil {
		return nil, err
	}
	unbound := []pgtype.UUID{}
	for _, attempt := range pending {
		if !attempt.CollectorEpoch.Valid {
			unbound = append(unbound, attempt.ID)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	owned = true
	return &CollectorSession{Token: uint64(control.FencingToken), conn: conn, store: s, unbound: unbound}, nil
}

func (s *CollectorSession) Check(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return ErrCollectorFenced
	}
	var valid bool
	// The lock is checked on its owning backend, not inferred from a process lease.
	err := s.conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND pid=pg_backend_pid() AND granted AND classid=((hashtextextended($1,0)>>32)&4294967295)::oid AND objid=(hashtextextended($1,0)&4294967295)::oid AND objsubid=1) AND EXISTS(SELECT 1 FROM trader_sync_collector_control WHERE singleton AND fencing_token=$2 AND owner_id IS NOT NULL)`, collectorLock, int64(s.Token)).Scan(&valid)
	if err != nil {
		return err
	}
	if !valid {
		return ErrCollectorFenced
	}
	return nil
}

func (s *CollectorSession) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil
	}
	conn := s.conn
	s.conn = nil
	defer conn.Release()
	// Closing the physical connection also releases the advisory lock on every error.
	defer conn.Conn().Close(context.Background())
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	queries := q.New(tx)
	control, err := queries.LockCollectorControl(ctx)
	if err != nil {
		return err
	}
	if control.FencingToken != int64(s.Token) {
		return ErrCollectorFenced
	}
	if control.ActiveEpoch.Valid {
		if err = endEpoch(ctx, queries, control.ActiveEpoch.Int64, "collector_stopped"); err != nil {
			return err
		}
	}
	if _, err = queries.ReleaseCollectorOwnership(ctx, int64(s.Token)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func boundedCollectorValue(value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("collector value exceeds PostgreSQL bigint")
	}
	return int64(value), nil
}

// Fence is always last: account -> wallet -> control, or wallet -> control.
// Ownership changes commit under control alone before any per-account cleanup.
func checkCollectorFence(ctx context.Context, queries *q.Queries, token, epoch uint64) error {
	if token == 0 {
		return ErrCollectorFenced
	}
	t, err := boundedCollectorValue(token)
	if err != nil {
		return err
	}
	e, err := boundedCollectorValue(epoch)
	if err != nil {
		return err
	}
	control, err := queries.LockCollectorControl(ctx)
	if err != nil {
		return err
	}
	if !control.OwnerID.Valid || control.FencingToken != t {
		return ErrCollectorFenced
	}
	if epoch != 0 {
		if !control.ActiveEpoch.Valid || control.ActiveEpoch.Int64 != e {
			return ErrCollectorFenced
		}
		row, err := queries.GetCollectorEpoch(ctx, e)
		if err != nil {
			return err
		}
		if row.EndedAt.Valid || row.FencingToken != t {
			return ErrCollectorFenced
		}
	}
	return nil
}
func (s *SQLStore) collectorTx(ctx context.Context, token, epoch uint64, fn func(*q.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	queries := q.New(tx)
	if err = checkCollectorFence(ctx, queries, token, epoch); err != nil {
		return err
	}
	if err = fn(queries); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func endEpoch(ctx context.Context, queries *q.Queries, epoch int64, reason string) error {
	if _, err := queries.EndCollectorEpoch(ctx, q.EndCollectorEpochParams{ID: epoch, Reason: reason}); err != nil {
		return err
	}
	if err := queries.RecordCollectorInterruption(ctx, epoch); err != nil {
		return err
	}
	_, err := queries.ClearCollectorEpoch(ctx, pgtype.Int8{Int64: epoch, Valid: true})
	return err
}
func (s *SQLStore) StartCollectorEpoch(ctx context.Context, token uint64) (uint64, error) {
	var epoch uint64
	err := s.collectorTx(ctx, token, 0, func(queries *q.Queries) error {
		control, err := queries.LockCollectorControl(ctx)
		if err != nil {
			return err
		}
		if control.ActiveEpoch.Valid {
			return errors.New("previous collector epoch is still open")
		}
		row, err := queries.CreateCollectorEpoch(ctx, int64(token))
		if err != nil {
			return err
		}
		n, err := queries.ActivateCollectorEpoch(ctx, q.ActivateCollectorEpochParams{FencingToken: int64(token), ActiveEpoch: pgtype.Int8{Int64: row.ID, Valid: true}})
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrCollectorFenced
		}
		epoch = uint64(row.ID)
		return nil
	})
	return epoch, err
}
func (s *SQLStore) CloseCollectorEpoch(ctx context.Context, token, epoch uint64, reason string) error {
	return s.collectorTx(ctx, token, epoch, func(queries *q.Queries) error { return endEpoch(ctx, queries, int64(epoch), reason) })
}

func cleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// RecoverPending ends only the exact committed NULL-epoch attempts captured
// under control during Acquire. New registrations are absent from this snapshot.
func (s *CollectorSession) RecoverPending(ctx context.Context) error {
	for _, id := range s.unbound {
		if err := s.store.abandonUnboundBaseline(ctx, s.Token, id); err != nil {
			return err
		}
	}
	return nil
}
