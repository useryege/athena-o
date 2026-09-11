package store

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/accountstate/txgate"
	q "github.com/useryege/athena/internal/tradersync/store/sqlc"
	tm "github.com/useryege/athena/internal/tradersync/types"
	"time"
)

// RegisterBaselineTx runs within the caller's account and wallet transaction.
// It publishes nothing in memory; a collector only consumes committed attempts.
func (s *SQLStore) RegisterBaselineTx(ctx context.Context, tx pgx.Tx, sub tm.Subscription, token, epoch uint64, observed tm.WalletObservation) error {
	owner, err := confirmationOwner(sub.OwnerID)
	if err != nil {
		return err
	}
	id, err := confirmationOwner(sub.ID)
	if err != nil {
		return err
	}
	generation, err := boundedCollectorValue(sub.Generation)
	if err != nil {
		return err
	}
	revision, err := boundedCollectorValue(sub.Revision)
	if err != nil {
		return err
	}
	high, err := boundedCollectorValue(observed.High)
	if err != nil {
		return err
	}
	seq, err := boundedCollectorValue(observed.Sequence)
	if err != nil {
		return err
	}
	queries := q.New(tx)
	if epoch != 0 {
		if err = checkCollectorFence(ctx, queries, token, epoch); err != nil {
			return err
		}
	} else {
		if _, err = queries.LockCollectorControl(ctx); err != nil {
			return err
		}
	}
	_, err = queries.CreateBaselineAttempt(ctx, q.CreateBaselineAttemptParams{OwnerID: owner, SubscriptionID: id, ActivationGeneration: generation, ExpectedRevision: revision, CollectorEpoch: pgtype.Int8{Int64: int64(epoch), Valid: epoch != 0}, RegisteredHigh: pgtype.Int8{Int64: high, Valid: epoch != 0}, RegisteredSequence: pgtype.Int8{Int64: seq, Valid: epoch != 0}})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrBaselineChanged
	}
	return err
}

type Baseline struct {
	ID                                                        string
	Subscription                                              tm.Subscription
	Epoch, FilterRevision, RegisteredHigh, RegisteredSequence uint64
	CandidateAt                                               *time.Time
	State                                                     string
}

func baselineSubscription(row q.TraderSyncSubscription) tm.Subscription {
	return tm.Subscription{ID: uuid.UUID(row.ID.Bytes).String(), OwnerID: uuid.UUID(row.OwnerID.Bytes).String(), Wallet: common.BytesToAddress(row.Wallet), DesiredState: row.DesiredState, ObservationState: row.ObservationState, Revision: uint64(row.Revision), Generation: uint64(row.ActivationGeneration), EffectiveAt: subscriptionTime(row.EffectiveAt), EndedAt: subscriptionTime(row.EndedAt)}
}
func (s *SQLStore) PendingBaselines(ctx context.Context) ([]Baseline, error) {
	queries := q.New(s.pool)
	rows, err := queries.ListPendingBaselines(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]Baseline, 0, len(rows))
	for _, row := range rows {
		sub, err := queries.GetBaselineSubscription(ctx, row.SubscriptionID)
		if err != nil {
			return nil, err
		}
		result = append(result, Baseline{ID: uuid.UUID(row.ID.Bytes).String(), Subscription: baselineSubscription(sub), Epoch: uint64(row.CollectorEpoch.Int64), FilterRevision: uint64(row.FilterRevision.Int64), RegisteredHigh: uint64(row.RegisteredHigh.Int64), RegisteredSequence: uint64(row.RegisteredSequence.Int64), CandidateAt: subscriptionTime(row.CandidateEffectiveAt), State: row.State})
	}
	return result, nil
}
func (s *SQLStore) CollectorTargets(ctx context.Context) ([]common.Address, error) {
	rows, err := q.New(s.pool).ListCollectorTargets(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]common.Address, len(rows))
	for i, row := range rows {
		result[i] = common.BytesToAddress(row)
	}
	return result, nil
}
func (s *SQLStore) SubscriptionsNeedingBaseline(ctx context.Context) ([]tm.Subscription, error) {
	rows, err := q.New(s.pool).ListSubscriptionsNeedingBaseline(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]tm.Subscription, len(rows))
	for i, row := range rows {
		result[i] = baselineSubscription(row)
	}
	return result, nil
}
func (s *SQLStore) BaselineNow(ctx context.Context) (time.Time, error) {
	row, err := q.New(s.pool).BaselineDatabaseNow(ctx)
	return row.Time, err
}
func (s *SQLStore) AckFilters(ctx context.Context, token, epoch, revision uint64) error {
	r, err := boundedCollectorValue(revision)
	if err != nil {
		return err
	}
	return s.collectorTx(ctx, token, epoch, func(queries *q.Queries) error {
		n, err := queries.AckCollectorFilters(ctx, q.AckCollectorFiltersParams{ID: int64(epoch), FilterRevision: r})
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrBaselineChanged
		}
		return nil
	})
}

// withBaseline never waits for an account while holding the control row. All
// intent/grant checks and the final interval publication share the account TX.
func (s *SQLStore) withBaseline(ctx context.Context, token uint64, id string, fn func(*q.Queries, q.TraderSyncBaselineAttempt) error) error {
	return s.withBaselineUsing(ctx, s.pool, token, id, fn)
}
func (s *SQLStore) withBaselineUsing(ctx context.Context, beginner txgate.Beginner, token uint64, id string, fn func(*q.Queries, q.TraderSyncBaselineAttempt) error) error {
	attemptID, err := confirmationOwner(id)
	if err != nil {
		return err
	}
	initial, err := q.New(s.pool).GetBaselineAttempt(ctx, attemptID)
	if err != nil {
		return err
	}
	return txgate.WithAccountTx(ctx, beginner, uuid.UUID(initial.OwnerID.Bytes).String(), func(tx pgx.Tx) error {
		queries := q.New(tx)
		row, err := queries.GetBaselineAttempt(ctx, attemptID)
		if err != nil {
			return err
		}
		sub, err := queries.GetBaselineSubscription(ctx, row.SubscriptionID)
		if err != nil {
			return err
		}
		if err = txgate.LockWallet(ctx, tx, common.BytesToAddress(sub.Wallet)); err != nil {
			return err
		}
		if err = checkCollectorFence(ctx, queries, token, uint64(row.CollectorEpoch.Int64)); err != nil {
			return err
		}
		if sub.DesiredState != "enabled" || sub.Revision != row.ExpectedRevision || sub.ActivationGeneration != row.ActivationGeneration {
			return ErrBaselineChanged
		}
		if err = s.RequireGrantTx(ctx, tx, uuid.UUID(row.OwnerID.Bytes).String()); err != nil {
			return err
		}
		return fn(queries, row)
	})
}
func (s *SQLStore) BindBaseline(ctx context.Context, token uint64, id string, epoch uint64, snapshot func() tm.WalletObservation) error {
	return s.withBaseline(ctx, token, id, func(queries *q.Queries, row q.TraderSyncBaselineAttempt) error {
		if err := checkCollectorFence(ctx, queries, token, epoch); err != nil {
			return err
		}
		observed := snapshot()
		high, err := boundedCollectorValue(observed.High)
		if err != nil {
			return err
		}
		seq, err := boundedCollectorValue(observed.Sequence)
		if err != nil {
			return err
		}
		n, err := queries.BindBaselineEpoch(ctx, q.BindBaselineEpochParams{ID: row.ID, CollectorEpoch: pgtype.Int8{Int64: int64(epoch), Valid: true}, RegisteredHigh: pgtype.Int8{Int64: high, Valid: true}, RegisteredSequence: pgtype.Int8{Int64: seq, Valid: true}})
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrBaselineChanged
		}
		return nil
	})
}
func (s *SQLStore) SaveBaselineBoundary(ctx context.Context, token uint64, id string, revision uint64, at time.Time) error {
	r, err := boundedCollectorValue(revision)
	if err != nil {
		return err
	}
	return s.withBaseline(ctx, token, id, func(queries *q.Queries, row q.TraderSyncBaselineAttempt) error {
		if !row.CollectorEpoch.Valid || row.State != "pending" || revision == 0 {
			return ErrBaselineChanged
		}
		epoch, err := queries.GetCollectorEpoch(ctx, row.CollectorEpoch.Int64)
		if err != nil {
			return err
		}
		if epoch.FilterRevision < r {
			return ErrBaselineChanged
		}
		n, err := queries.SaveBaselineBoundary(ctx, q.SaveBaselineBoundaryParams{ID: row.ID, FilterRevision: pgtype.Int8{Int64: r, Valid: true}, CandidateEffectiveAt: pgtype.Timestamptz{Time: at, Valid: true}})
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrBaselineChanged
		}
		return nil
	})
}
func (s *SQLStore) CompleteBaseline(ctx context.Context, token uint64, id string) error {
	return s.completeBaselineUsing(ctx, s.pool, token, id)
}
func (s *SQLStore) completeBaselineUsing(ctx context.Context, beginner txgate.Beginner, token uint64, id string) error {
	// Reading persisted success first also resolves a previous unknown COMMIT.
	successful := func(readCtx context.Context) (bool, error) {
		parsed, e := confirmationOwner(id)
		if e != nil {
			return false, e
		}
		row, e := q.New(s.pool).GetBaselineAttempt(readCtx, parsed)
		if e != nil {
			return false, e
		}
		if row.State != "succeeded" {
			return false, nil
		}
		return q.New(s.pool).HasBaselineInterval(readCtx, parsed)
	}
	if done, err := successful(ctx); err != nil || done {
		return err
	}
	err := s.withBaselineUsing(ctx, beginner, token, id, func(queries *q.Queries, row q.TraderSyncBaselineAttempt) error {
		if !row.CollectorEpoch.Valid || !row.FilterRevision.Valid {
			return ErrBaselineChanged
		}
		epoch, err := queries.GetCollectorEpoch(ctx, row.CollectorEpoch.Int64)
		if err != nil {
			return err
		}
		if epoch.FilterRevision < row.FilterRevision.Int64 {
			return ErrBaselineChanged
		}
		row, err = queries.SucceedBaselineAttempt(ctx, row.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBaselineChanged
		}
		if err != nil {
			return err
		}
		if err = queries.InsertBaselineInterval(ctx, row.ID); err != nil {
			return err
		}
		n, err := queries.ProjectBaselineHealthy(ctx, q.ProjectBaselineHealthyParams{ID: row.SubscriptionID, EffectiveAt: row.EffectiveAt, Revision: row.ExpectedRevision, ActivationGeneration: row.ActivationGeneration})
		if err != nil {
			return err
		}
		if n != 1 {
			return ErrBaselineChanged
		}
		return nil
	})
	if err != nil {
		readCtx, cancel := cleanupContext()
		defer cancel()
		if done, readErr := successful(readCtx); readErr == nil && done {
			return nil
		}
	}
	return err
}
func (s *SQLStore) FailBaseline(ctx context.Context, token uint64, id, reason string) error {
	return s.withBaseline(ctx, token, id, func(queries *q.Queries, row q.TraderSyncBaselineAttempt) error {
		_, err := queries.FailBaselineAttempt(ctx, q.FailBaselineAttemptParams{ID: row.ID, Reason: reason})
		return err
	})
}
func (s *SQLStore) CleanupStoppedBaselines(ctx context.Context, token uint64) error {
	owners, err := q.New(s.pool).ListClosedEpochOwners(ctx)
	if err != nil {
		return err
	}
	for _, owner := range owners {
		if err = txgate.WithAccountTx(ctx, s.pool, uuid.UUID(owner.Bytes).String(), func(tx pgx.Tx) error {
			queries := q.New(tx)
			if err := checkCollectorFence(ctx, queries, token, 0); err != nil {
				return err
			}
			if err := queries.ProjectStoppedEpochSubscriptions(ctx, owner); err != nil {
				return err
			}
			if err := queries.CloseStoppedEpochIntervals(ctx, owner); err != nil {
				return err
			}
			return queries.FailStoppedEpochBaselines(ctx, owner)
		}); err != nil {
			return err
		}
	}
	return nil
}

// RegisterNeededBaseline rechecks the live intent inside the same account TX as
// the Registrar. That callback takes wallet before capturing the receive cutoff.
func (s *SQLStore) RegisterNeededBaseline(ctx context.Context, expected tm.Subscription, register func(context.Context, pgx.Tx, tm.Subscription) error) error {
	return txgate.WithAccountTx(ctx, s.pool, expected.OwnerID, func(tx pgx.Tx) error {
		id, err := confirmationOwner(expected.ID)
		if err != nil {
			return err
		}
		row, err := q.New(tx).GetBaselineSubscription(ctx, id)
		if err != nil {
			return err
		}
		current := baselineSubscription(row)
		if current.DesiredState != "enabled" || current.Revision != expected.Revision || current.Generation != expected.Generation {
			return ErrBaselineChanged
		}
		if err = s.RequireGrantTx(ctx, tx, expected.OwnerID); err != nil {
			return err
		}
		return register(ctx, tx, current)
	})
}

func (s *SQLStore) abandonUnboundBaseline(ctx context.Context, token uint64, id pgtype.UUID) error {
	initial, err := q.New(s.pool).GetBaselineAttempt(ctx, id)
	if err != nil {
		return err
	}
	return txgate.WithAccountTx(ctx, s.pool, uuid.UUID(initial.OwnerID.Bytes).String(), func(tx pgx.Tx) error {
		queries := q.New(tx)
		sub, err := queries.GetBaselineSubscription(ctx, initial.SubscriptionID)
		if err != nil {
			return err
		}
		if err = txgate.LockWallet(ctx, tx, common.BytesToAddress(sub.Wallet)); err != nil {
			return err
		}
		if err = checkCollectorFence(ctx, queries, token, 0); err != nil {
			return err
		}
		row, err := queries.GetBaselineAttempt(ctx, id)
		if err != nil {
			return err
		}
		if row.State != "pending" || row.CollectorEpoch.Valid {
			return nil
		}
		_, err = queries.FailBaselineAttempt(ctx, q.FailBaselineAttemptParams{ID: id, Reason: "observation_ownership_changed"})
		return err
	})
}

var ErrCheckpointStale = errors.New("observation_checkpoint_stale")
var ErrObservationClock = errors.New("clock_order_uncertain")

func (s *SQLStore) CheckpointIntervals(ctx context.Context, epoch uint64, after string, limit int32) ([]tm.CheckpointInterval, error) {
	id, e := readUUID(after)
	if e != nil {
		return nil, e
	}
	if limit < 1 || limit > 100 {
		return nil, errors.New("checkpoint batch limit must be 1..100")
	}
	rows, e := q.New(s.pool).ListCheckpointIntervals(ctx, q.ListCheckpointIntervalsParams{Epoch: int64(epoch), AfterID: id, RowLimit: limit})
	if e != nil {
		return nil, e
	}
	out := make([]tm.CheckpointInterval, 0, len(rows))
	for _, r := range rows {
		out = append(out, tm.CheckpointInterval{ID: readID(r.ID), OwnerID: readID(r.OwnerID), SubscriptionID: readID(r.SubscriptionID), Wallet: common.BytesToAddress(r.Wallet), Generation: uint64(r.ActivationGeneration), Epoch: uint64(r.CollectorEpoch), FilterRevision: uint64(r.FilterRevision), EffectiveAt: r.EffectiveAt.Time, LastReliableAt: subscriptionTime(r.LastReliableAt)})
	}
	return out, nil
}
func (s *SQLStore) SaveObservationCheckpoint(ctx context.Context, target tm.CheckpointInterval, evidence tm.ObservationCheckpoint, alive func() bool) error {
	valid := func() bool {
		return alive != nil && alive() && !evidence.At.IsZero() && time.Now().Before(evidence.ExpiresAt) && !evidence.At.Before(evidence.CoveredAt)
	}
	if !valid() {
		return ErrCheckpointStale
	}
	owner, e := readUUID(target.OwnerID)
	if e != nil {
		return e
	}
	id, e := readUUID(target.ID)
	if e != nil {
		return e
	}
	return txgate.WithAccountTx(ctx, s.pool, target.OwnerID, func(tx pgx.Tx) error {
		if !valid() {
			return ErrCheckpointStale
		}
		queries := q.New(tx)
		if e := checkCollectorFence(ctx, queries, evidence.Token, evidence.Epoch); e != nil {
			return e
		}
		r, e := queries.GetCheckpointInterval(ctx, q.GetCheckpointIntervalParams{OwnerID: owner, ID: id})
		if e != nil {
			return e
		}
		covered := false
		for _, wallet := range evidence.Wallets {
			if wallet == common.BytesToAddress(r.Wallet) {
				covered = true
				break
			}
		}
		if !valid() || !covered || r.DesiredState != "enabled" || r.EndedAt.Valid || r.EpochEndedAt.Valid || uint64(r.CollectorEpoch) != evidence.Epoch || r.CurrentGeneration != r.ActivationGeneration || uint64(r.ActivationGeneration) != target.Generation || readID(r.SubscriptionID) != target.SubscriptionID || uint64(r.FilterRevision) > evidence.FilterRevision || evidence.At.Before(r.EffectiveAt.Time) {
			return ErrCheckpointStale
		}
		// The database stores UTC only. A regressing UTC cannot be repaired by
		// inventing a newer point or by calling GREATEST and claiming advancement.
		if r.LastReliableAt.Valid && evidence.At.Before(r.LastReliableAt.Time) {
			return ErrObservationClock
		}
		if evidence.At.UTC().After(time.Now().UTC()) {
			return ErrObservationClock
		}
		n, e := queries.SaveIntervalCheckpoint(ctx, q.SaveIntervalCheckpointParams{ObservedAt: pgtype.Timestamptz{Time: evidence.At.UTC(), Valid: true}, OwnerID: owner, ID: id, Epoch: int64(evidence.Epoch), Generation: int64(target.Generation)})
		if e != nil {
			return e
		}
		if n != 1 {
			return ErrCheckpointStale
		}
		return nil
	})
}
