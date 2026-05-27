package store

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"github.com/useryege/athena/util/env"
)

const (
	postgresPingAttempts = 5
	postgresPingInterval = time.Second
)

type SQLStore struct {
	db *sql.DB
}

type CreateDeliveryRequest struct {
	Source   string
	Severity string
	Title    string
	Body     string
	Link     string
	Channel  string
	Status   string
}

type ListDeliveriesOptions struct {
	Page     int
	PageSize int
	Status   string
	Severity string
	Source   string
	Keyword  string
}

func NewSQLStore(db *sql.DB) *SQLStore {
	return &SQLStore{db: db}
}

func NewSQLStoreSource() func(context.Context) (*SQLStore, error) {
	return func(ctx context.Context) (*SQLStore, error) {
		log.Info("connecting to notification postgres database")
		db, err := sql.Open("pgx", postgresDSN())
		if err != nil {
			return nil, fmt.Errorf("failed to open notification postgres database: %w", err)
		}

		var pingErr error
		for attempt := 1; attempt <= postgresPingAttempts; attempt++ {
			pingCtx, cancel := context.WithTimeout(ctx, postgresPingInterval)
			pingErr = db.PingContext(pingCtx)
			cancel()
			if pingErr == nil {
				log.Info("successfully connected to notification postgres database")
				return NewSQLStore(db), nil
			}

			log.Warnf("failed to ping notification postgres database, attempt %d/%d: %v", attempt, postgresPingAttempts, pingErr)
			if attempt < postgresPingAttempts {
				select {
				case <-ctx.Done():
					_ = db.Close()
					return nil, fmt.Errorf("notification postgres database ping interrupted: %w", ctx.Err())
				case <-time.After(postgresPingInterval):
				}
			}
		}

		_ = db.Close()
		return nil, fmt.Errorf("failed to ping notification postgres database after %d attempts: %w", postgresPingAttempts, pingErr)
	}
}

func postgresDSN() string {
	if dsn := env.StringFromEnv("ATHENA_NOTIFICATION_POSTGRES_DSN", ""); dsn != "" {
		return dsn
	}
	return defaultPostgresDSN()
}

func defaultPostgresDSN() string {
	postgresUser := env.StringFromEnv("POSTGRES_USER", "athena")
	postgresPassword := env.StringFromEnv("POSTGRES_PASSWORD", "")

	postgresURL := url.URL{
		Scheme: "postgres",
		User:   url.User(postgresUser),
		Host:   net.JoinHostPort("127.0.0.1", env.StringFromEnv("ATHENA_POSTGRES_PORT", "5432")),
		Path:   "notification",
	}
	if postgresPassword != "" {
		postgresURL.User = url.UserPassword(postgresUser, postgresPassword)
	}

	query := postgresURL.Query()
	query.Set("sslmode", "disable")
	postgresURL.RawQuery = query.Encode()

	return postgresURL.String()
}

func (s *SQLStore) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *SQLStore) CreateDelivery(ctx context.Context, req CreateDeliveryRequest) (*v1alpha1.NotificationDeliveryDetail, error) {
	if s.db == nil {
		return nil, fmt.Errorf("notification postgres database is not configured")
	}
	var item v1alpha1.NotificationDeliveryDetail
	var createdAt time.Time
	err := s.db.QueryRowContext(ctx, `
INSERT INTO notification_deliveries (source, severity, title, body, link, channel, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, source, severity, title, body, link, channel, status, created_at
`, req.Source, req.Severity, req.Title, req.Body, req.Link, req.Channel, req.Status).Scan(
		&item.ID,
		&item.Source,
		&item.Severity,
		&item.Title,
		&item.Body,
		&item.Link,
		&item.Channel,
		&item.Status,
		&createdAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification delivery: %w", err)
	}
	item.CreatedAt = formatTime(createdAt)
	return &item, nil
}

func (s *SQLStore) MarkDeliverySent(ctx context.Context, id int64, providerMessageID string) error {
	if s.db == nil {
		return fmt.Errorf("notification postgres database is not configured")
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE notification_deliveries
SET status = 'sent', provider_message_id = $2, error_message = NULL, sent_at = NOW()
WHERE id = $1
`, id, providerMessageID)
	if err != nil {
		return fmt.Errorf("failed to mark notification delivery sent: %w", err)
	}
	return nil
}

func (s *SQLStore) MarkDeliveryFailed(ctx context.Context, id int64, errorMessage string) error {
	if s.db == nil {
		return fmt.Errorf("notification postgres database is not configured")
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE notification_deliveries
SET status = 'failed', error_message = $2
WHERE id = $1
`, id, errorMessage)
	if err != nil {
		return fmt.Errorf("failed to mark notification delivery failed: %w", err)
	}
	return nil
}

func (s *SQLStore) ListDeliveries(ctx context.Context, opts ListDeliveriesOptions) ([]*v1alpha1.NotificationDeliveryItem, int64, error) {
	if s.db == nil {
		return nil, 0, fmt.Errorf("notification postgres database is not configured")
	}
	where, args := deliveryFilterWhere(opts)
	var total int64
	countQuery := "SELECT COUNT(*) FROM notification_deliveries" + where
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count notification deliveries: %w", err)
	}

	page := opts.Page
	if page < 1 {
		page = 1
	}
	pageSize := opts.PageSize
	if pageSize < 1 {
		pageSize = 20
	}
	args = append(args, pageSize, (page-1)*pageSize)
	query := `
SELECT id, source, severity, COALESCE(title, ''), body, COALESCE(link, ''), channel, status, provider_message_id, error_message, created_at, sent_at
FROM notification_deliveries` + where + `
ORDER BY created_at DESC, id DESC
LIMIT $` + fmt.Sprint(len(args)-1) + ` OFFSET $` + fmt.Sprint(len(args))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notification deliveries: %w", err)
	}
	defer rows.Close()

	items := []*v1alpha1.NotificationDeliveryItem{}
	for rows.Next() {
		item, err := scanDeliveryItem(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to read notification deliveries: %w", err)
	}
	return items, total, nil
}

func (s *SQLStore) GetDelivery(ctx context.Context, id int64) (*v1alpha1.NotificationDeliveryDetail, error) {
	if s.db == nil {
		return nil, fmt.Errorf("notification postgres database is not configured")
	}
	row := s.db.QueryRowContext(ctx, `
SELECT id, source, severity, COALESCE(title, ''), body, COALESCE(link, ''), channel, status, provider_message_id, error_message, created_at, sent_at
FROM notification_deliveries
WHERE id = $1
`, id)
	item, err := scanDeliveryDetail(row)
	if err != nil {
		return nil, err
	}
	return item, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func deliveryFilterWhere(opts ListDeliveriesOptions) (string, []any) {
	clauses := []string{}
	args := []any{}
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}
	if status := strings.TrimSpace(opts.Status); status != "" {
		add("status = $%d", status)
	}
	if severity := strings.TrimSpace(opts.Severity); severity != "" {
		add("severity = $%d", severity)
	}
	if source := strings.TrimSpace(opts.Source); source != "" {
		add("source = $%d", source)
	}
	if keyword := strings.TrimSpace(opts.Keyword); keyword != "" {
		args = append(args, "%"+keyword+"%")
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, "(title ILIKE "+placeholder+" OR body ILIKE "+placeholder+" OR error_message ILIKE "+placeholder+" OR provider_message_id ILIKE "+placeholder+")")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func scanDeliveryItem(row rowScanner) (*v1alpha1.NotificationDeliveryItem, error) {
	var item v1alpha1.NotificationDeliveryItem
	var providerMessageID sql.NullString
	var errorMessage sql.NullString
	var createdAt time.Time
	var sentAt sql.NullTime
	if err := row.Scan(
		&item.ID,
		&item.Source,
		&item.Severity,
		&item.Title,
		&item.Body,
		&item.Link,
		&item.Channel,
		&item.Status,
		&providerMessageID,
		&errorMessage,
		&createdAt,
		&sentAt,
	); err != nil {
		return nil, err
	}
	item.ProviderMessageID = providerMessageID.String
	item.ErrorMessage = errorMessage.String
	item.CreatedAt = formatTime(createdAt)
	item.SentAt = formatNullTime(sentAt)
	return &item, nil
}

func scanDeliveryDetail(row rowScanner) (*v1alpha1.NotificationDeliveryDetail, error) {
	item, err := scanDeliveryItem(row)
	if err != nil {
		return nil, err
	}
	return &v1alpha1.NotificationDeliveryDetail{
		ID:                item.ID,
		Source:            item.Source,
		Severity:          item.Severity,
		Title:             item.Title,
		Body:              item.Body,
		Link:              item.Link,
		Channel:           item.Channel,
		Status:            item.Status,
		ProviderMessageID: item.ProviderMessageID,
		ErrorMessage:      item.ErrorMessage,
		CreatedAt:         item.CreatedAt,
		SentAt:            item.SentAt,
	}, nil
}

func formatNullTime(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return formatTime(value.Time)
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
