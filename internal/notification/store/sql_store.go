package store

import (
	"context"
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	notificationsqlc "github.com/useryege/athena/internal/notification/store/sqlc"
	"github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrations() embed.FS {
	return migrations
}

type SQLStore struct {
	pool    *pgxpool.Pool
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
			DSNEnv:       "ATHENA_NOTIFICATION_POSTGRES_DSN",
			Database:     "notification",
			Migrations:   migrations,
			MigrationDir: "migrations",
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
