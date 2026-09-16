//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestRunReadOnlyApplyAndRepeatedSnapshot(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	dsn, err := url.Parse(db.DSN)
	require.NoError(t, err)
	query := dsn.Query()
	query.Set("pool_max_conns", "1")
	dsn.RawQuery = query.Encode()
	t.Setenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN", dsn.String())
	ctx := context.Background()
	_, err = db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','default',1)`)
	require.NoError(t, err)
	for _, source := range []string{"worm-markets.new-event", "polymarket.worm-markets"} {
		_, err = db.Pool.Exec(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES($1,'info','hello','telegram','pending','test','default','{}',decode(repeat('ab',32),'hex'))`, source)
		require.NoError(t, err)
	}

	var out bytes.Buffer
	require.NoError(t, run(ctx, nil, &out))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "read_only", report.Status)
	require.True(t, report.CountsVerified)
	require.Zero(t, report.Cancelled)
	require.Zero(t, report.RetiredTotal)
	require.EqualValues(t, 1, report.Pending)
	require.NotEmpty(t, report.Database)
	var exactState string
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT status FROM system_notification_deliveries WHERE source='worm-markets.new-event'`).Scan(&exactState))
	require.Equal(t, "pending", exactState)
	lock, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	_, err = lock.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended('athena:retire-worm-markets-notifications',0))`)
	require.NoError(t, err)
	out.Reset()
	require.Error(t, run(ctx, []string{"--apply"}, &out))
	require.NoError(t, lock.Hijack().Close(ctx))

	for _, wantCancelled := range []int64{1, 0} {
		out.Reset()
		require.NoError(t, run(ctx, []string{"--apply", "--batch-size=1"}, &out))
		require.NoError(t, json.Unmarshal(out.Bytes(), &report))
		require.Equal(t, "completed", report.Status)
		require.True(t, report.CountsVerified)
		require.Equal(t, wantCancelled, report.Cancelled)
		require.EqualValues(t, 1, report.RetiredTotal)
		require.Zero(t, report.Pending)
		require.Zero(t, report.Sending)
	}
	var nearState string
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT status FROM system_notification_deliveries WHERE source='polymarket.worm-markets'`).Scan(&nearState))
	require.Equal(t, "pending", nearState)
}

func TestRunSendingTimeoutReportsLastVerifiedCountsThenCancelsRetry(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	t.Setenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN", db.DSN)
	ctx := context.Background()
	_, err := db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','default',1)`)
	require.NoError(t, err)
	_, err = db.Pool.Exec(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('worm-markets.live-event','info','hello','telegram','sending','test','default','{}',decode(repeat('ab',32),'hex'))`)
	require.NoError(t, err)

	var out bytes.Buffer
	require.Error(t, run(ctx, []string{"--apply", "--timeout=200ms"}, &out))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "incomplete", report.Status)
	require.True(t, report.CountsVerified)
	require.Zero(t, report.Cancelled)
	require.Zero(t, report.RetiredTotal)
	require.Zero(t, report.Pending)
	require.EqualValues(t, 1, report.Sending)

	// This models the existing sender's genuine retryable outcome. Retirement
	// never rewrites sending itself, but cancels it after it becomes pending.
	_, err = db.Pool.Exec(ctx, `UPDATE system_notification_deliveries SET status='pending'`)
	require.NoError(t, err)
	out.Reset()
	require.NoError(t, run(ctx, []string{"--apply"}, &out))
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "completed", report.Status)
	require.True(t, report.CountsVerified)
	require.EqualValues(t, 1, report.Cancelled)
	require.EqualValues(t, 1, report.RetiredTotal)
}
