package store

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/tradersync/activity"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
)

// FinalityObservationCutoff sees only committed rows, never sequence allocation.
func (s *SQLStore) FinalityObservationCutoff(ctx context.Context) (int64, error) {
	return q.New(s.pool).FinalityObservationCutoff(ctx)
}

// RecordFinalityObservation touches only one source row and never an account
// gate. A failed/unknown commit is returned to the instance's conservative gap
// handling; it does not request another RPC or fabricate another sample.
func (s *SQLStore) RecordFinalityObservation(ctx context.Context, id int64, o tm.FinalityRoundObservation) error {
	tx, err := s.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer rollbackRuntimeTx(tx)
	queries := q.New(tx)
	raw, err := queries.LockFinalityTiming(ctx, id)
	if err != nil {
		return err
	}
	var previous tm.FinalityTiming
	if err = json.Unmarshal(raw, &previous); err != nil {
		return err
	}
	next := activity.MergeFinalityObservation(previous, o)
	encoded, err := json.Marshal(next)
	if err != nil {
		return err
	}
	if err = queries.SaveFinalityTiming(ctx, q.SaveFinalityTimingParams{ID: id, FinalityTiming: encoded}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
