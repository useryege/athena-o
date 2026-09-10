package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	accountstatemigrations "github.com/useryege/athena/internal/accountstate/store/migrations"
	notificationsqlc "github.com/useryege/athena/internal/notification/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

type sqlPool interface {
	notificationsqlc.DBTX
	Begin(context.Context) (pgx.Tx, error)
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	Close()
}

type SQLStore struct {
	pool    sqlPool
	queries notificationsqlc.Querier
}

func NewSQLStore(pool *pgxpool.Pool) *SQLStore {
	if pool == nil {
		return &SQLStore{}
	}
	return &SQLStore{pool: pool, queries: notificationsqlc.New(pool)}
}

func NewSQLStoreWithQuerier(querier notificationsqlc.Querier) *SQLStore {
	return &SQLStore{queries: querier}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		pool, err := postgres.ConnectAndMigrate(ctx, postgres.Options{
			Module:       "notification",
			DSNEnv:       "ATHENA_SERVER_POSTGRES_DSN",
			Database:     "athena",
			Migrations:   accountstatemigrations.FS,
			MigrationDir: accountstatemigrations.Dir,
		})
		if err != nil {
			return nil, err
		}
		log.Info("notification postgres migrations are up to date")
		return NewSQLStore(pool), nil
	}
}

func (s *SQLStore) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}

func (s *SQLStore) configured() error {
	if s.queries == nil {
		return fmt.Errorf("notification postgres database is not configured")
	}
	return nil
}

func (s *SQLStore) transactional() error {
	if s.pool == nil || s.queries == nil {
		return fmt.Errorf("notification postgres database is not configured")
	}
	return nil
}

func uuidValue(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid uuid: %w", err)
	}
	return pgtype.UUID{Bytes: [16]byte(parsed), Valid: true}, nil
}

func uuidString(value pgtype.UUID) string {
	if !value.Valid {
		return ""
	}
	return uuid.UUID(value.Bytes).String()
}

func textValue(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: true}
}

func nullableText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return textValue(value)
}

func nullableKeyword(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return textValue("%" + value + "%")
}

func intervalValue(value time.Duration) pgtype.Interval {
	return pgtype.Interval{Microseconds: value.Microseconds(), Valid: true}
}

func timestamptzValue(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}

// BorrowPool returns the existing physical pool without transferring ownership.
// Only this SQLStore's composition owner closes it, after Service.Stop joins all work.
func (s *SQLStore) BorrowPool() (*pgxpool.Pool, error) {
	p, ok := s.pool.(*pgxpool.Pool)
	if !ok || p == nil {
		return nil, fmt.Errorf("physical notification pool required")
	}
	return p, nil
}
