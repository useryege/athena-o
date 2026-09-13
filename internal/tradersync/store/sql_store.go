package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/accountstate/txgate"
)

var ErrRuntimeRequired = errors.New("Trader Sync runtime write gate is required")

type SQLStore struct {
	pool                  *pgxpool.Pool
	runtimeGate           *RuntimeWriteGate
	directoryTransactions txgate.Beginner
	activitySiteURL       string
}

// NewSQLStore supports reads and caller-owned transaction adapters only.
func NewSQLStore(pool *pgxpool.Pool) *SQLStore { return &SQLStore{pool: pool} }

func NewRuntimeSQLStore(pool *pgxpool.Pool, gate *RuntimeWriteGate) (*SQLStore, error) {
	if pool == nil || gate == nil || gate.pool != pool {
		return nil, ErrRuntimeRequired
	}
	return &SQLStore{pool: pool, runtimeGate: gate}, nil
}

// BeginTx establishes runtime ownership before any business lock.
func (s *SQLStore) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	if s.runtimeGate == nil {
		return nil, ErrRuntimeRequired
	}
	return s.runtimeGate.BeginTx(ctx, opts)
}

// runtimeTransactions guards dedicated connections and transaction decorators.
// Caller-owned API revocation and Notification transactions never use it.
func (s *SQLStore) runtimeTransactions(beginner txgate.Beginner) txgate.Beginner {
	return runtimeTransactions{store: s, beginner: beginner}
}

type runtimeTransactions struct {
	store    *SQLStore
	beginner txgate.Beginner
}

func (b runtimeTransactions) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	if b.store.runtimeGate == nil {
		return nil, ErrRuntimeRequired
	}
	tx, err := b.beginner.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err = b.store.runtimeGate.CheckTx(ctx, tx); err != nil {
		rollbackRuntimeTx(tx)
		return nil, err
	}
	return runtimeWriteTx{Tx: tx}, nil
}

// WithRuntime borrows this pool and preserves local activity configuration.
func (s *SQLStore) WithRuntime(token RuntimeToken) (*SQLStore, error) {
	gate, err := NewRuntimeWriteGate(s.pool, token)
	if err != nil {
		return nil, err
	}
	next, err := NewRuntimeSQLStore(s.pool, gate)
	if err != nil {
		return nil, err
	}
	next.activitySiteURL = s.activitySiteURL
	return next, nil
}
