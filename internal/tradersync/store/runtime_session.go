package store

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
)

// RuntimeSession owns the dedicated PostgreSQL advisory connection and both
// durable tokens. Its owner must join all workers before releasing the session.
type RuntimeSession struct {
	mu             sync.Mutex
	conn           *pgxpool.Conn
	token          RuntimeToken
	collectorToken uint64
	store          *SQLStore
	unbound        []pgtype.UUID
}

const collectorLock = "athena:trader-sync:collector"

func (s *SQLStore) AcquireRuntimeSession(ctx context.Context) (*RuntimeSession, error) {
	conn, err := s.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	owned := false
	defer func() {
		if !owned {
			discardRuntimeConnection(conn)
		}
	}()
	var locked bool
	if err = conn.QueryRow(ctx, "SELECT pg_try_advisory_lock(hashtextextended($1,0))", collectorLock).Scan(&locked); err != nil {
		return nil, err
	}
	if !locked {
		return nil, errors.New("another Trader Sync runtime owns the database")
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer rollbackRuntimeTx(tx)
	queries := q.New(tx)
	// Ownership operations use runtime -> collector only. No account lock or
	// per-account recovery can occur until this transaction has committed.
	if _, err = queries.LockRuntimeControl(ctx); err != nil {
		return nil, err
	}
	control, err := queries.LockCollectorControl(ctx)
	if err != nil {
		return nil, err
	}
	if control.ActiveEpoch.Valid {
		if err = endEpoch(ctx, queries, control.ActiveEpoch.Int64, "collector_replaced"); err != nil {
			return nil, err
		}
	}
	ownerID := pgtype.UUID{Bytes: uuid.New(), Valid: true}
	runtime, err := queries.ClaimRuntimeOwnership(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("claim runtime ownership: %w", err)
	}
	control, err = queries.ClaimCollectorOwnership(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("claim collector ownership: %w", err)
	}
	pending, err := queries.ListPendingBaselines(ctx)
	if err != nil {
		return nil, err
	}
	var unbound []pgtype.UUID
	for _, attempt := range pending {
		if !attempt.CollectorEpoch.Valid {
			unbound = append(unbound, attempt.ID)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	owned = true
	return &RuntimeSession{conn: conn, token: RuntimeToken{OwnerID: uuid.UUID(ownerID.Bytes), Generation: uint64(runtime.Generation)}, collectorToken: uint64(control.FencingToken), store: s, unbound: unbound}, nil
}

func (s *RuntimeSession) RuntimeToken() RuntimeToken { return s.token }
func (s *RuntimeSession) CollectorToken() uint64     { return s.collectorToken }

func (s *RuntimeSession) Check(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return ErrRuntimeFenced
	}
	var valid bool
	// Query the owning backend's actual advisory lock as well as durable identity.
	err := s.conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND pid=pg_backend_pid() AND granted AND classid=((hashtextextended($1,0)>>32)&4294967295)::oid AND objid=(hashtextextended($1,0)&4294967295)::oid AND objsubid=1)
 AND EXISTS(SELECT 1 FROM trader_sync_runtime_control WHERE singleton AND owner_id=$2 AND generation=$3)
 AND EXISTS(SELECT 1 FROM trader_sync_collector_control WHERE singleton AND owner_id=$2 AND fencing_token=$4)`, collectorLock, s.token.OwnerID, int64(s.token.Generation), int64(s.collectorToken)).Scan(&valid)
	if err != nil {
		return errors.Join(ErrRuntimeFenced, err)
	}
	if !valid {
		return ErrRuntimeFenced
	}
	return nil
}

func (s *RuntimeSession) CloseAfterWorkers(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn == nil {
		return nil
	}
	conn := s.conn
	s.conn = nil
	// Always discard this dedicated physical connection, including ambiguous
	// rollback/commit errors. A session lock must never re-enter the pool.
	defer discardRuntimeConnection(conn)
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer rollbackRuntimeTx(tx)
	queries := q.New(tx)
	runtime, err := queries.LockRuntimeControl(ctx)
	if err != nil {
		return err
	}
	if !runtime.OwnerID.Valid || runtime.OwnerID.Bytes != [16]byte(s.token.OwnerID) || runtime.Generation != int64(s.token.Generation) {
		return ErrRuntimeFenced
	}
	control, err := queries.LockCollectorControl(ctx)
	if err != nil {
		return err
	}
	if !control.OwnerID.Valid || control.OwnerID.Bytes != [16]byte(s.token.OwnerID) || control.FencingToken != int64(s.collectorToken) {
		return ErrRuntimeFenced
	}
	if control.ActiveEpoch.Valid {
		if err = endEpoch(ctx, queries, control.ActiveEpoch.Int64, "collector_stopped"); err != nil {
			return err
		}
	}
	if _, err = queries.ReleaseCollectorOwnership(ctx, int64(s.collectorToken)); err != nil {
		return err
	}
	n, err := queries.ReleaseRuntimeOwnership(ctx, q.ReleaseRuntimeOwnershipParams{OwnerID: runtime.OwnerID, Generation: runtime.Generation})
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrRuntimeFenced
	}
	return tx.Commit(ctx)
}

func discardRuntimeConnection(conn *pgxpool.Conn) {
	raw := conn.Hijack()
	ctx, cancel := cleanupContext()
	defer cancel()
	_ = raw.Close(ctx)
}

func rollbackRuntimeTx(tx pgx.Tx) {
	ctx, cancel := cleanupContext()
	defer cancel()
	_ = tx.Rollback(ctx)
}

// RecoverPending ends only committed NULL-epoch attempts captured during claim.
// It runs after ownership commits; new registrations are absent from the snapshot.
func (s *RuntimeSession) RecoverPending(ctx context.Context) error {
	for _, id := range s.unbound {
		if err := s.store.abandonUnboundBaseline(ctx, s.collectorToken, id); err != nil {
			return err
		}
	}
	return nil
}
