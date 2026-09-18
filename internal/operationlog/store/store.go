// Package store implements the durable operation-log inbox and serialized publisher.
package store

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/useryege/athena/internal/operationlog/event"
	"github.com/useryege/athena/internal/operationlog/ingest"
	"strings"
	"time"
)

const ProjectInterval = 500 * time.Millisecond
const ProjectTimeout = 2 * time.Second

type Store struct{ pool *pgxpool.Pool }

var _ ingest.Sink = (*Store)(nil)

// Projection describes a committed batch. Busy means another publisher held the lock.
type Projection struct {
	PublishedSeq           int64
	Processed, Quarantined int
	Busy                   bool
}

// NormalizedView is a persisted projection, not an ingress envelope. Its
// observation may be COMPLETE; Decode remains strict for producer events.
type NormalizedView struct {
	event.Event
	FirstReceivedAt time.Time `json:"firstReceivedAt"`
	LastReceivedAt  time.Time `json:"lastReceivedAt"`
}

// Open owns a service pool. Startup schema verification is explicit in schema.Verify.
func Open(ctx context.Context, dsn string) (*Store, error) { return open(ctx, dsn, 8, true) }

// OpenProducer constructs a dedicated, lazy pool capped at four connections.
// A missing schema or unavailable database is reported by each bounded Append,
// so a long-lived producer can reconnect after recovery without restarting API.
func OpenProducer(ctx context.Context, dsn string) (*Store, error) { return open(ctx, dsn, 4, false) }
func open(ctx context.Context, dsn string, max int32, ping bool) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("operation-log postgres DSN required")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = max
	cfg.MinConns = 0
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if ping {
		bounded, cancel := context.WithTimeout(ctx, ProjectTimeout)
		defer cancel()
		if err = pool.Ping(bounded); err != nil {
			pool.Close()
			return nil, err
		}
	}
	return &Store{pool: pool}, nil
}
func (s *Store) Close() { s.pool.Close() }

// Read gives query adapters one short repeatable-read, read-only snapshot. It
// must not escape the callback; no connection is held between HTTP requests.
func (s *Store) Read(ctx context.Context, f func(pgx.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, ProjectTimeout)
	defer cancel()
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = f(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func id(v string) pgtype.UUID {
	u, err := uuid.Parse(v)
	return pgtype.UUID{Bytes: u, Valid: err == nil}
}
func optionalID(v *string) pgtype.UUID {
	if v == nil {
		return pgtype.UUID{}
	}
	return id(*v)
}
func textValue(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *v, Valid: true}
}
func timestamp(v time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: v, Valid: !v.IsZero()}
}
func optionalTime(v *time.Time) pgtype.Timestamptz {
	if v == nil {
		return pgtype.Timestamptz{}
	}
	return timestamp(*v)
}
func intValue(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}
