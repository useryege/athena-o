package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	q "github.com/useryege/athena/internal/notification/store/sqlc"
	"os"
	"strings"
	"sync"
	"time"
)

const senderLockKey int64 = 1096042561

var ErrSenderActive = errors.New("notification sender is active or requires explicit stopped-process confirmation")

type SenderSession struct {
	mu          sync.Mutex
	conn        *pgxpool.Conn
	incarnation uuid.UUID
	failed      bool
	firstStart  bool
}

func (s *SQLStore) lockSender(ctx context.Context) (*SenderSession, error) {
	pool, ok := s.pool.(interface {
		Acquire(context.Context) (*pgxpool.Conn, error)
	})
	if !ok {
		return nil, fmt.Errorf("sender requires a dedicated PostgreSQL session")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	var locked bool
	if err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, senderLockKey).Scan(&locked); err != nil {
		// A failed acknowledgement cannot prove that the server did not take the lock.
		raw := conn.Hijack()
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = raw.Close(closeCtx)
		return nil, err
	}
	if !locked {
		conn.Release()
		return nil, ErrSenderActive
	}
	return &SenderSession{conn: conn}, nil
}
func (s *SQLStore) AcquireSender(ctx context.Context, id uuid.UUID) (*SenderSession, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("sender incarnation required")
	}
	session, err := s.lockSender(ctx)
	if err != nil {
		return nil, err
	}
	queries := q.New(session.conn)
	old, err := queries.ListUnstoppedSenders(ctx)
	if err != nil || len(old) > 0 {
		session.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", ErrSenderActive, uuidString(old[0].Incarnation))
	}
	history, err := queries.HasSenderHistory(ctx)
	if err != nil {
		session.Close()
		return nil, err
	}
	session.firstStart = history.Valid && !history.Bool
	host, _ := os.Hostname()
	identity := host + ":" + fmt.Sprint(os.Getpid())
	if stat, readErr := os.ReadFile("/proc/self/stat"); readErr == nil {
		identity += " " + strings.TrimSpace(string(stat))
	}
	err = queries.RegisterSender(ctx, q.RegisterSenderParams{Incarnation: uuidPG(id), Hostname: host, ProcessID: int32(os.Getpid()), ProcessIdentity: identity})
	if err != nil {
		session.Close()
		return nil, err
	}
	session.incarnation = id
	return session, nil
}

// Check validates the dedicated session still owns the singleton lock. A failure is sticky.
func (s *SenderSession) Check(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.failed || s.conn == nil {
		return ErrSenderActive
	}
	var held bool
	err := s.conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid=pg_backend_pid() AND locktype='advisory' AND classid=0 AND objid=$1::oid AND objsubid=1 AND granted)`, senderLockKey).Scan(&held)
	if err != nil || !held {
		s.failed = true
		if err != nil {
			return err
		}
		return ErrSenderActive
	}
	return nil
}

// Finish is called only after the poller and every sender/result writer have joined.
func (s *SenderSession) Finish(ctx context.Context) error {
	if err := s.Check(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := q.New(s.conn).StopSenderInstance(ctx, q.StopSenderInstanceParams{Incarnation: uuidPG(s.incarnation), StopConfirmation: textValue("graceful")})
	return err
}

// Close destroys the session instead of returning a possibly locked connection to the pool.
func (s *SenderSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.conn != nil {
		conn := s.conn.Hijack()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = conn.Close(ctx)
		s.conn = nil
	}
}

// ConfirmStoppedSender records an operator assertion, never an inference from a lease or PID.
// The caller MUST have independently confirmed the registered process has exited.
func (s *SQLStore) ConfirmStoppedSender(ctx context.Context, id uuid.UUID) error {
	session, err := s.lockSender(ctx)
	if err != nil {
		return err
	}
	defer session.Close()
	if _, err = s.queries.GetSenderInstance(ctx, uuidPG(id)); err != nil {
		return err
	}
	if _, err = s.queries.StopSenderInstance(ctx, q.StopSenderInstanceParams{Incarnation: uuidPG(id), StopConfirmation: textValue("operator")}); err != nil {
		return err
	}
	return s.RecoverSender(ctx, id)
}

func (s *SenderSession) FirstStart() bool { return s.firstStart }
