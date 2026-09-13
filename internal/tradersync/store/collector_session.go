package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/accountstate/txgate"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
)

var ErrCollectorFenced = errors.New("collector no longer owns this epoch")
var ErrBaselineChanged = errors.New("baseline intent or boundary changed")

func boundedCollectorValue(value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("collector value exceeds PostgreSQL bigint")
	}
	return int64(value), nil
}

// Inside the runtime share lock, the collector fence remains last:
// account -> wallet -> control, or wallet -> control. Ownership changes commit
// under runtime -> control before any per-account cleanup.
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
	return s.collectorTxUsing(ctx, s.pool, token, epoch, fn)
}
func (s *SQLStore) collectorTxUsing(ctx context.Context, beginner txgate.Beginner, token, epoch uint64, fn func(*q.Queries) error) error {
	tx, err := s.runtimeTransactions(beginner).BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollbackRuntimeTx(tx)
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
	return s.startCollectorEpochUsing(ctx, s.pool, token)
}
func (s *SQLStore) startCollectorEpochUsing(ctx context.Context, beginner txgate.Beginner, token uint64) (uint64, error) {
	var epoch uint64
	err := s.collectorTxUsing(ctx, beginner, token, 0, func(queries *q.Queries) error {
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
	if err != nil && epoch != 0 {
		// A lost COMMIT response is not evidence of rollback. The control row lock
		// waits out the original transaction and verifies this exact active epoch.
		readCtx, stop := cleanupContext()
		defer stop()
		readErr := s.collectorTx(readCtx, token, epoch, func(*q.Queries) error { return nil })
		if readErr == nil {
			return epoch, nil
		}
		err = errors.Join(err, fmt.Errorf("confirm created collector epoch: %w", readErr))
	}
	return epoch, err
}
func (s *SQLStore) CloseCollectorEpoch(ctx context.Context, token, epoch uint64, reason string) error {
	return s.collectorTx(ctx, token, epoch, func(queries *q.Queries) error { return endEpoch(ctx, queries, int64(epoch), reason) })
}

func cleanupContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
