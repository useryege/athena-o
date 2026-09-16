//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

func TestRunReadOnlyApplyReentryAndMutualExclusion(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	dsn, err := url.Parse(db.DSN)
	require.NoError(t, err)
	q := dsn.Query()
	q.Set("pool_max_conns", "1")
	dsn.RawQuery = q.Encode()
	t.Setenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN", dsn.String())
	ctx := context.Background()
	_, topicErr := db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','default',1)`)
	require.NoError(t, topicErr)
	_, err = db.Pool.Exec(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('polymarket.sports-live-score','info','hello','telegram','pending','test','default','{}',decode(repeat('ab',32),'hex'))`)
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, run(ctx, nil, &out))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	var fields map[string]any
	require.NoError(t, json.Unmarshal(out.Bytes(), &fields))
	require.NotContains(t, fields, "retired_total")
	require.Equal(t, "read_only", report.Status)
	require.True(t, report.CountsVerified)
	require.EqualValues(t, 1, report.Pending)
	require.NotEmpty(t, report.Database)
	var state string
	require.NoError(t, db.Pool.QueryRow(ctx, `SELECT status FROM system_notification_deliveries`).Scan(&state))
	require.Equal(t, "pending", state)
	lock, err := db.Pool.Acquire(ctx)
	require.NoError(t, err)
	_, err = lock.Exec(ctx, `SELECT pg_advisory_lock(hashtextextended('athena:retire-sports-notifications',0))`)
	require.NoError(t, err)
	out.Reset()
	require.Error(t, run(ctx, []string{"--apply"}, &out))
	require.NoError(t, lock.Hijack().Close(ctx))
	for _, wantCancelled := range []int64{1, 0} {
		out.Reset()
		require.NoError(t, run(ctx, []string{"--apply"}, &out))
		require.NoError(t, json.Unmarshal(out.Bytes(), &report))
		require.Equal(t, "completed", report.Status)
		require.Equal(t, wantCancelled, report.Cancelled)
		require.Zero(t, report.Pending)
		require.Zero(t, report.Sending)
	}
}

func TestRunSendingTimeoutAndInterruptedReentry(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	t.Setenv("ATHENA_ACCOUNT_STATE_POSTGRES_DSN", db.DSN)
	ctx := context.Background()
	_, topicErr := db.Pool.Exec(ctx, `INSERT INTO system_notification_topics(telegram_chat,label,message_thread_id) VALUES('test','default',1)`)
	require.NoError(t, topicErr)
	_, err := db.Pool.Exec(ctx, `INSERT INTO system_notification_deliveries(source,severity,body,channel,status,telegram_chat,topic_label,payload,payload_digest) VALUES('polymarket.sports-live-score','info','hello','telegram','sending','test','default','{}',decode(repeat('ab',32),'hex'))`)
	require.NoError(t, err)
	var out bytes.Buffer
	require.Error(t, run(ctx, []string{"--apply", "--timeout=200ms"}, &out))
	var report result
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "incomplete", report.Status)
	require.EqualValues(t, 1, report.Sending)
	interrupted, cancel := context.WithCancel(ctx)
	go func() { time.Sleep(100 * time.Millisecond); cancel() }()
	out.Reset()
	require.Error(t, run(interrupted, []string{"--apply"}, &out))
	// Model the existing sender's genuine retryable result; the tool never changes sending.
	_, err = db.Pool.Exec(ctx, `UPDATE system_notification_deliveries SET status='pending'`)
	require.NoError(t, err)
	out.Reset()
	require.NoError(t, run(ctx, []string{"--apply"}, &out))
	require.NoError(t, json.Unmarshal(out.Bytes(), &report))
	require.Equal(t, "completed", report.Status)
	require.EqualValues(t, 1, report.Cancelled)
}
