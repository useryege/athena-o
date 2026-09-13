package store

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
)

var ErrRuntimeFenced = errors.New("Trader Sync runtime no longer owns the database")

type RuntimeToken struct {
	OwnerID    uuid.UUID
	Generation uint64
}

// RuntimeWriteGate takes the runtime share lock before any business lock and
// retains it on the returned transaction until commit or rollback.
type RuntimeWriteGate struct {
	pool  *pgxpool.Pool
	token RuntimeToken
}

func NewRuntimeWriteGate(pool *pgxpool.Pool, token RuntimeToken) (*RuntimeWriteGate, error) {
	if pool == nil {
		return nil, errors.New("runtime transaction pool is required")
	}
	if token.OwnerID == uuid.Nil || token.Generation == 0 || token.Generation > math.MaxInt64 {
		return nil, errors.New("runtime owner and positive PostgreSQL bigint generation are required")
	}
	return &RuntimeWriteGate{pool: pool, token: token}, nil
}

func (g *RuntimeWriteGate) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	tx, err := g.pool.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	if err = g.CheckTx(ctx, tx); err != nil {
		cleanupCtx, cancel := cleanupContext()
		defer cancel()
		_ = tx.Rollback(cleanupCtx)
		return nil, err
	}
	return tx, nil
}

// CheckTx must run before account, wallet, source, or collector locks. It is for
// Trader Sync's own transactions, never caller-owned API/Notification adapters.
func (g *RuntimeWriteGate) CheckTx(ctx context.Context, tx pgx.Tx) error {
	if tx == nil {
		return errors.Join(ErrRuntimeFenced, errors.New("runtime transaction is required"))
	}
	_, err := q.New(tx).CheckRuntimeWrite(ctx, q.CheckRuntimeWriteParams{
		OwnerID: pgtype.UUID{Bytes: g.token.OwnerID, Valid: true}, Generation: int64(g.token.Generation),
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrRuntimeFenced
	}
	if err != nil {
		return errors.Join(ErrRuntimeFenced, fmt.Errorf("check runtime write: %w", err))
	}
	return nil
}
