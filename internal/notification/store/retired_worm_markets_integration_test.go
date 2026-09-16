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

var wormMarketsNotificationSources = []string{
	"worm-markets.new-event",
	"worm-markets.live-event",
	"worm-markets.price-alert-80-20",
	"worm-markets.price-alert-90-10",
	"worm-markets.price-alert-95-5",
}

func wormMarketsFixture(t *testing.T, source string) (*SQLStore, delivery.WorkRef) {
	t.Helper()
	s, ref := attemptFixture(t, "system")
	_, err := s.pool.Exec(context.Background(), "UPDATE system_notification_deliveries SET source=$1 WHERE id=$2", source, ref.ID)
	require.NoError(t, err)
	return s, ref
}

func cloneSystemDelivery(t *testing.T, s *SQLStore, ref delivery.WorkRef, source, status string) int64 {
	t.Helper()
	var id int64
	err := s.pool.QueryRow(context.Background(), `
		INSERT INTO system_notification_deliveries(
			source,severity,title,body,link,channel,status,provider_message_id,error_message,
			telegram_chat,topic_label,attempts,next_attempt_at,last_attempt_at,locked_at,locked_by,payload,payload_digest
		)
		SELECT $1,severity,title,body,link,channel,$2,provider_message_id,error_message,
			telegram_chat,topic_label,attempts,next_attempt_at,last_attempt_at,locked_at,locked_by,payload,payload_digest
		FROM system_notification_deliveries WHERE id=$3 RETURNING id`, source, status, ref.ID).Scan(&id)
	require.NoError(t, err)
	return id
}

func TestRetireWormMarketsPendingBecomesCancelled(t *testing.T) {
	s, ref := wormMarketsFixture(t, "worm-markets.new-event")
	ctx := context.Background()

	counts, err := s.RetireWormMarketsNotifications(ctx, 100)
	require.NoError(t, err)
	require.EqualValues(t, 1, counts.Cancelled)
	require.Zero(t, counts.Pending)
	require.Zero(t, counts.Sending)
	require.Equal(t, "cancelled", deliveryState(t, s, ref))
	var reason string
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT error_message FROM system_notification_deliveries WHERE id=$1`, ref.ID).Scan(&reason))
	require.Equal(t, "WORM_MARKETS_RETIRED", reason)
	_, err = s.Authorize(ctx, testPermitCandidate(ref), uuid.New(), nil)
	require.ErrorIs(t, err, ErrDeliveryNotEligible)
}

func TestRetireWormMarketsExactSourcesAndBoundedReentry(t *testing.T) {
	s, ref := wormMarketsFixture(t, wormMarketsNotificationSources[0])
	ctx := context.Background()
	for _, source := range wormMarketsNotificationSources[1:] {
		cloneSystemDelivery(t, s, ref, source, "pending")
	}
	nearSources := []string{
		"polymarket.worm-markets",
		"worm-markets",
		"worm-markets.new-event.extra",
		"worm-markets.price-alert-99-1",
		"trader-sync.activity",
	}
	for _, source := range nearSources {
		cloneSystemDelivery(t, s, ref, source, "pending")
	}
	otherCancellation := cloneSystemDelivery(t, s, ref, "worm-markets.new-event", "cancelled")
	_, err := s.pool.Exec(ctx, `UPDATE system_notification_deliveries SET error_message='USER_CANCELLED' WHERE id=$1`, otherCancellation)
	require.NoError(t, err)

	before, err := s.CountWormMarketsNotifications(ctx)
	require.NoError(t, err)
	require.Equal(t, WormMarketsRetirementCounts{Pending: 5}, before)
	for i, want := range []WormMarketsRetirementCounts{
		{Cancelled: 2, Pending: 3},
		{Cancelled: 2, Pending: 1},
		{Cancelled: 1},
		{},
	} {
		got, err := s.RetireWormMarketsNotifications(ctx, 2)
		require.NoError(t, err, "batch %d", i)
		require.Equal(t, want, got, "batch %d", i)
	}
	after, err := s.CountWormMarketsNotifications(ctx)
	require.NoError(t, err)
	require.Equal(t, WormMarketsRetirementCounts{Cancelled: 5}, after)
	for _, source := range nearSources {
		var status string
		require.NoError(t, s.pool.QueryRow(ctx, `SELECT status FROM system_notification_deliveries WHERE source=$1`, source).Scan(&status))
		require.Equal(t, "pending", status, source)
	}
	var otherReason string
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT error_message FROM system_notification_deliveries WHERE id=$1`, otherCancellation).Scan(&otherReason))
	require.Equal(t, "USER_CANCELLED", otherReason)
}

func TestRetireWormMarketsDoesNotRewriteSending(t *testing.T) {
	s, ref := attemptFixture(t, "system")
	ctx := context.Background()
	_, err := s.pool.Exec(ctx, "UPDATE system_notification_deliveries SET source=$1 WHERE id=$2", "worm-markets.new-event", ref.ID)
	require.NoError(t, err)
	_, err = s.Authorize(ctx, testPermitCandidate(ref), uuid.New(), nil)
	require.NoError(t, err)
	counts, err := s.RetireWormMarketsNotifications(ctx, 100)
	require.NoError(t, err)
	require.EqualValues(t, 1, counts.Sending)
	require.Equal(t, "sending", deliveryState(t, s, ref))
}

func TestRetireWormMarketsRetryReturnsToPendingThenCancels(t *testing.T) {
	s, ref := wormMarketsFixture(t, "worm-markets.live-event")
	ctx := context.Background()
	permit, err := s.Authorize(ctx, testPermitCandidate(ref), uuid.New(), nil)
	require.NoError(t, err)
	beforeTopics, beforeOffset := int64(0), int64(0)
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM system_notification_topics`).Scan(&beforeTopics))
	require.NoError(t, s.pool.QueryRow(ctx, `UPDATE telegram_polling_state SET next_update_id=321 RETURNING next_update_id`).Scan(&beforeOffset))
	require.NoError(t, s.RecordOutcome(ctx, permit, delivery.Outcome{Kind: "retryable", Code: "provider_429"}, time.Now()))
	require.Equal(t, "pending", deliveryState(t, s, ref))

	counts, err := s.RetireWormMarketsNotifications(ctx, 100)
	require.NoError(t, err)
	require.Equal(t, WormMarketsRetirementCounts{Cancelled: 1}, counts)
	var attempts, attemptRows int64
	var topics, offset int64
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT attempts FROM system_notification_deliveries WHERE id=$1`, ref.ID).Scan(&attempts))
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM notification_delivery_attempts WHERE work_kind='system' AND work_id=$1`, ref.ID).Scan(&attemptRows))
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT count(*) FROM system_notification_topics`).Scan(&topics))
	require.NoError(t, s.pool.QueryRow(ctx, `SELECT next_update_id FROM telegram_polling_state`).Scan(&offset))
	require.EqualValues(t, 1, attempts)
	require.EqualValues(t, 1, attemptRows)
	require.Equal(t, beforeTopics, topics)
	require.Equal(t, beforeOffset, offset)
}

func TestRetireWormMarketsPreservesTerminalRowsAndAttempts(t *testing.T) {
	for _, outcome := range []string{"sent", "failed", "unknown"} {
		t.Run(outcome, func(t *testing.T) {
			s, ref := wormMarketsFixture(t, "worm-markets.price-alert-90-10")
			ctx := context.Background()
			permit, err := s.Authorize(ctx, testPermitCandidate(ref), uuid.New(), nil)
			require.NoError(t, err)
			messageID := ""
			if outcome == "sent" {
				messageID = "provider-message-7"
			}
			require.NoError(t, s.RecordOutcome(ctx, permit, delivery.Outcome{Kind: outcome, MessageID: messageID, Code: "original-result"}, time.Now()))
			counts, err := s.RetireWormMarketsNotifications(ctx, 100)
			require.NoError(t, err)
			require.Zero(t, counts.Cancelled)
			require.Equal(t, outcome, deliveryState(t, s, ref))
			var storedOutcome, code string
			require.NoError(t, s.pool.QueryRow(ctx, `SELECT outcome,outcome_code FROM notification_delivery_attempts WHERE id=$1`, permit.AttemptID).Scan(&storedOutcome, &code))
			require.Equal(t, outcome, storedOutcome)
			require.Equal(t, "original-result", code)
		})
	}
}

func TestRetireWormMarketsCancellationWinsRowLockRace(t *testing.T) {
	s, ref := wormMarketsFixture(t, "worm-markets.price-alert-95-5")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	const gate int64 = 801955
	blocker, err := s.pool.Begin(ctx)
	require.NoError(t, err)
	defer blocker.Rollback(context.Background())
	_, err = blocker.Exec(ctx, `SELECT pg_advisory_lock($1)`, gate)
	require.NoError(t, err)
	_, err = s.pool.Exec(ctx, `
		CREATE FUNCTION block_worm_retirement() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN PERFORM pg_advisory_xact_lock(801955); RETURN NEW; END $$;
		CREATE TRIGGER block_worm_retirement BEFORE UPDATE ON system_notification_deliveries
		FOR EACH ROW WHEN (NEW.error_message = 'WORM_MARKETS_RETIRED') EXECUTE FUNCTION block_worm_retirement()`)
	require.NoError(t, err)

	retired := make(chan error, 1)
	go func() {
		_, err := s.RetireWormMarketsNotifications(ctx, 100)
		retired <- err
	}()
	waitForLockWaiters(t, ctx, s, 1)
	authorized := make(chan error, 1)
	go func() {
		_, err := s.Authorize(ctx, testPermitCandidate(ref), uuid.New(), nil)
		authorized <- err
	}()
	waitForLockWaiters(t, ctx, s, 2)
	_, err = blocker.Exec(ctx, `SELECT pg_advisory_unlock($1)`, gate)
	require.NoError(t, err)
	require.NoError(t, blocker.Commit(ctx))
	require.NoError(t, <-retired)
	require.ErrorIs(t, <-authorized, ErrDeliveryNotEligible)
	require.Equal(t, "cancelled", deliveryState(t, s, ref))
}

func TestRetireWormMarketsRejectsInvalidBatchAndCancelledCannotReauthorize(t *testing.T) {
	s, ref := wormMarketsFixture(t, "worm-markets.price-alert-80-20")
	for _, batchSize := range []int32{0, -1} {
		_, err := s.RetireWormMarketsNotifications(context.Background(), batchSize)
		require.Error(t, err)
	}
	_, err := s.RetireWormMarketsNotifications(context.Background(), 1)
	require.NoError(t, err)
	_, err = s.Authorize(context.Background(), testPermitCandidate(ref), uuid.New(), nil)
	require.True(t, errors.Is(err, ErrDeliveryNotEligible))

	cancelled, err := s.RetireWormMarketsNotifications(context.Background(), 1)
	require.NoError(t, err)
	require.Zero(t, cancelled.Cancelled)
}

func waitForLockWaiters(t *testing.T, ctx context.Context, s *SQLStore, want int) {
	t.Helper()
	for {
		var count int
		err := s.pool.QueryRow(ctx, `SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'`).Scan(&count)
		require.NoError(t, err)
		if count >= want {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("expected %d lock waiters: %v", want, ctx.Err())
		case <-time.After(5 * time.Millisecond):
		}
	}
}
