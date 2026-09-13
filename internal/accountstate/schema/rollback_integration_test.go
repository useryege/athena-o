//go:build integration

package schema

import (
	"bytes"
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/testutil/pgtest"
)

// rollbackStallConn forwards real PostgreSQL traffic except the first ROLLBACK,
// whose write blocks on a net.Pipe. Both transports honor network deadlines.
// Closing the fixture always releases the blocked write, including on RED.
type rollbackStallConn struct {
	net.Conn
	stall, peer net.Conn
	block       *atomic.Bool
	entered     chan struct{}
	closed      chan struct{}
	closeOnce   sync.Once
}

func (c *rollbackStallConn) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("rollback")) && c.block.CompareAndSwap(false, true) {
		close(c.entered)
		return c.stall.Write(p)
	}
	return c.Conn.Write(p)
}
func (c *rollbackStallConn) SetDeadline(deadline time.Time) error {
	if err := c.stall.SetDeadline(deadline); err != nil {
		return err
	}
	return c.Conn.SetDeadline(deadline)
}
func (c *rollbackStallConn) SetWriteDeadline(deadline time.Time) error {
	if err := c.stall.SetWriteDeadline(deadline); err != nil {
		return err
	}
	return c.Conn.SetWriteDeadline(deadline)
}
func (c *rollbackStallConn) Close() error {
	c.closeOnce.Do(func() { _ = c.stall.Close(); _ = c.peer.Close(); close(c.closed) })
	return c.Conn.Close()
}

func TestVerifyRollbackHonorsTotalDeadlineAndDiscardsFailedConnection(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	config, err := pgxpool.ParseConfig(db.DSN)
	require.NoError(t, err)
	config.MaxConns = 1
	var block atomic.Bool
	entered := make(chan struct{})
	connections := make(chan *rollbackStallConn, 8)
	config.ConnConfig.DialFunc = func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
		if err != nil {
			return nil, err
		}
		stall, peer := net.Pipe()
		wrapped := &rollbackStallConn{Conn: conn, stall: stall, peer: peer, block: &block, entered: entered, closed: make(chan struct{})}
		connections <- wrapped
		return wrapped, nil
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)
	defer pool.Close()
	var initialPID uint32
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT pg_backend_pid()`).Scan(&initialPID))
	first := <-connections
	defer first.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	done := make(chan error, 1)
	go func() { done <- Verify(ctx, pool) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("verification never reached the blocked ROLLBACK")
	}
	select {
	case err := <-done:
		require.ErrorIs(t, err, ErrVersions)
		require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
		require.Less(t, time.Since(started), 400*time.Millisecond, "cleanup exceeded the caller's total budget")
	case <-time.After(400 * time.Millisecond):
		t.Fatal("Verify remains blocked in ROLLBACK after its 100ms total deadline")
	}
	select {
	case <-first.closed:
	case <-time.After(time.Second):
		t.Fatal("failed rollback connection was not closed")
	}
	// A single-connection pool must replace the failed connection, not hand it
	// back as usable. Reusing this pool for a healthy verification must succeed.
	healthy, cancelHealthy := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelHealthy()
	require.NoError(t, Up(healthy, db.DSN))
	require.NoError(t, Verify(healthy, pool))
	var recoveredPID uint32
	require.NoError(t, pool.QueryRow(healthy, `SELECT pg_backend_pid()`).Scan(&recoveredPID))
	require.NotEqual(t, initialPID, recoveredPID)
}
