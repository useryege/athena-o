//go:build integration

package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/notification/delivery"
)

func retiredSportsFixture(t *testing.T) (*SQLStore, delivery.WorkRef) {
	t.Helper()
	s, ref := attemptFixture(t, "system")
	_, err := s.pool.Exec(context.Background(), `UPDATE system_notification_deliveries SET source='polymarket.sports-live-score' WHERE id=$1`, ref.ID)
	require.NoError(t, err)
	return s, ref
}

func TestRetiredSportsPendingCancellationPreservesLedger(t *testing.T) {
	s, ref := retiredSportsFixture(t)
	ctx := context.Background()
	n, err := s.CancelRetiredSportsPending(ctx, 100)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
	require.Equal(t, "cancelled", deliveryState(t, s, ref))
	var reason string
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT error_message FROM system_notification_deliveries WHERE id=$1`, ref.ID).Scan(&reason))
	require.Equal(t, "source retired: sports removal", reason)
	n, err = s.CancelRetiredSportsPending(ctx, 100)
	require.NoError(t, err)
	require.Zero(t, n)
	_, err = s.Authorize(ctx, testPermitCandidate(ref), uuid.New(), nil)
	require.True(t, errors.Is(err, ErrDeliveryNotEligible))
}

func TestRetiredSportsExactSourcesAndBatches(t *testing.T) {
	s, ref := retiredSportsFixture(t)
	ctx := context.Background()
	sources := []string{"polymarket.sports-live-price-alert-85-15", "polymarket.sports-live-price-alert-90-10", "polymarket.sports-live-price-alert-95-5", "polymarket.sports-live-price-alert-97-3", "polymarket.sports-live-price-alert-99-1", "polymarket.worm-markets", "trader-sync.activity", "polymarket.sports-live-score.extra", "polymarket.sports-live-price-alert-99-2"}
	for _, source := range sources {
		_, err := s.pool.Exec(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) SELECT $1,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest FROM system_notification_deliveries WHERE id=$2`, source, ref.ID)
		require.NoError(t, err)
	}
	counts, err := s.CountRetiredSports(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 6, counts.Pending)
	for _, want := range []int64{2, 2, 2, 0} {
		n, err := s.CancelRetiredSportsPending(ctx, 2)
		require.NoError(t, err)
		require.Equal(t, want, n)
	}
	var pending, total int
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FILTER (WHERE status='pending'),count(*) FROM system_notification_deliveries`).Scan(&pending, &total))
	require.Equal(t, 4, pending)
	require.Equal(t, 10, total)
}

func TestRetiredSportsSendingWaitsForRealOutcome(t *testing.T) {
	for _, outcome := range []string{"retryable", "sent", "failed", "unknown"} {
		t.Run(outcome, func(t *testing.T) {
			s, ref := retiredSportsFixture(t)
			ctx := context.Background()
			p, err := s.Authorize(ctx, testPermitCandidate(ref), uuid.New(), nil)
			require.NoError(t, err)
			n, err := s.CancelRetiredSportsPending(ctx, 100)
			require.NoError(t, err)
			require.Zero(t, n)
			counts, err := s.CountRetiredSports(ctx)
			require.NoError(t, err)
			require.EqualValues(t, 1, counts.Sending)
			require.NoError(t, s.RecordOutcome(ctx, p, delivery.Outcome{Kind: outcome, MessageID: "123", Code: "real_result"}, time.Now()))
			n, err = s.CancelRetiredSportsPending(ctx, 100)
			require.NoError(t, err)
			want := outcome
			if outcome == "retryable" {
				want = "cancelled"
				require.EqualValues(t, 1, n)
			} else {
				require.Zero(t, n)
			}
			require.Equal(t, want, deliveryState(t, s, ref))
			var result string
			require.NoError(t, s.pool.QueryRow(ctx, `SELECT outcome FROM notification_delivery_attempts WHERE id=$1`, p.AttemptID).Scan(&result))
			require.Equal(t, outcome, result)
		})
	}
}

func TestRetiredSportsPermitRowLockAndReentry(t *testing.T) {
	s, ref := retiredSportsFixture(t)
	ctx := context.Background()
	tx, err := s.pool.Begin(ctx)
	require.NoError(t, err)
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `SELECT id FROM system_notification_deliveries WHERE id=$1 FOR UPDATE`, ref.ID)
	require.NoError(t, err)
	n, err := s.CancelRetiredSportsPending(ctx, 100)
	require.NoError(t, err)
	require.Zero(t, n)
	require.NoError(t, tx.Rollback(ctx))
	n, err = s.CancelRetiredSportsPending(ctx, 100)
	require.NoError(t, err)
	require.EqualValues(t, 1, n)
}

func TestRetiredSportsTimeoutAndInvalidBatch(t *testing.T) {
	s, ref := retiredSportsFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.CancelRetiredSportsPending(ctx, 100)
	require.Error(t, err)
	_, err = s.CountRetiredSports(ctx)
	require.Error(t, err)
	for _, limit := range []int32{0, -1, 101} {
		_, err = s.CancelRetiredSportsPending(context.Background(), limit)
		require.Error(t, err)
	}
	tx, err := s.pool.Begin(context.Background())
	require.NoError(t, err)
	defer tx.Rollback(context.Background())
	_, err = tx.Exec(context.Background(), `LOCK TABLE system_notification_deliveries IN ACCESS EXCLUSIVE MODE`)
	require.NoError(t, err)
	ctx, cancel = context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err = s.CancelRetiredSportsPending(ctx, 100)
	require.Error(t, err)
	require.NoError(t, tx.Rollback(context.Background()))
	require.Equal(t, "pending", deliveryState(t, s, ref))
}
