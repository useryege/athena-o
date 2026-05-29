package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	notificationsqlc "github.com/useryege/athena/internal/notification/store/sqlc"
	"github.com/useryege/athena/internal/postgres"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

//go:embed migrations/*.sql
var migrations embed.FS

type SQLStore struct {
	pool    *pgxpool.Pool
	queries *notificationsqlc.Queries
	legacy  *sql.DB
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

func NewSQLStore(db any) *SQLStore {
	var queries *notificationsqlc.Queries
	switch value := db.(type) {
	case *pgxpool.Pool:
		if value != nil {
			queries = notificationsqlc.New(value)
		}
		return &SQLStore{pool: value, queries: queries}
	case *sql.DB:
		return &SQLStore{legacy: value}
	case nil:
		return &SQLStore{}
	default:
		panic(fmt.Sprintf("unsupported notification postgres store db %T", db))
	}
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
	if s.pool == nil {
		return nil
	}
	s.pool.Close()
	return nil
}

func (s *SQLStore) CreateDelivery(ctx context.Context, req CreateDeliveryRequest) (*v1alpha1.NotificationDeliveryDetail, error) {
	if s.legacy != nil {
		var item v1alpha1.NotificationDeliveryDetail
		var createdAt time.Time
		err := s.legacy.QueryRowContext(ctx, `
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
	if s.queries == nil {
		return nil, fmt.Errorf("notification postgres database is not configured")
	}
	row, err := s.queries.CreateDelivery(ctx, notificationsqlc.CreateDeliveryParams{
		Source:   req.Source,
		Severity: req.Severity,
		Title:    textValue(req.Title),
		Body:     req.Body,
		Link:     textValue(req.Link),
		Channel:  req.Channel,
		Status:   req.Status,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create notification delivery: %w", err)
	}
	item := deliveryDetailFromRow(deliveryRow{
		ID: row.ID, Source: row.Source, Severity: row.Severity, Title: row.Title, Body: row.Body, Link: row.Link, Channel: row.Channel,
		Status: row.Status, ProviderMessageID: row.ProviderMessageID, ErrorMessage: row.ErrorMessage, CreatedAt: row.CreatedAt, SentAt: row.SentAt,
	})
	return item, nil
}

func (s *SQLStore) MarkDeliverySent(ctx context.Context, id int64, providerMessageID string) error {
	if s.legacy != nil {
		_, err := s.legacy.ExecContext(ctx, `
UPDATE notification_deliveries
SET status = 'sent', provider_message_id = $2, error_message = NULL, sent_at = NOW()
WHERE id = $1
`, id, providerMessageID)
		if err != nil {
			return fmt.Errorf("failed to mark notification delivery sent: %w", err)
		}
		return nil
	}
	if s.queries == nil {
		return fmt.Errorf("notification postgres database is not configured")
	}
	if err := s.queries.MarkDeliverySent(ctx, notificationsqlc.MarkDeliverySentParams{ID: id, ProviderMessageID: textValue(providerMessageID)}); err != nil {
		return fmt.Errorf("failed to mark notification delivery sent: %w", err)
	}
	return nil
}

func (s *SQLStore) MarkDeliveryFailed(ctx context.Context, id int64, errorMessage string) error {
	if s.legacy != nil {
		_, err := s.legacy.ExecContext(ctx, `
UPDATE notification_deliveries
SET status = 'failed', error_message = $2
WHERE id = $1
`, id, errorMessage)
		if err != nil {
			return fmt.Errorf("failed to mark notification delivery failed: %w", err)
		}
		return nil
	}
	if s.queries == nil {
		return fmt.Errorf("notification postgres database is not configured")
	}
	if err := s.queries.MarkDeliveryFailed(ctx, notificationsqlc.MarkDeliveryFailedParams{ID: id, ErrorMessage: textValue(errorMessage)}); err != nil {
		return fmt.Errorf("failed to mark notification delivery failed: %w", err)
	}
	return nil
}

func (s *SQLStore) ListDeliveries(ctx context.Context, opts ListDeliveriesOptions) ([]*v1alpha1.NotificationDeliveryItem, int64, error) {
	if s.legacy != nil {
		where, args := deliveryFilterWhere(opts)
		var total int64
		countQuery := "SELECT COUNT(*) FROM notification_deliveries" + where
		if err := s.legacy.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
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
		rows, err := s.legacy.QueryContext(ctx, query, args...)
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
	if s.queries == nil {
		return nil, 0, fmt.Errorf("notification postgres database is not configured")
	}
	params := deliveryFilterParams(opts)
	total, err := s.queries.CountDeliveries(ctx, notificationsqlc.CountDeliveriesParams{
		Status:   params.Status,
		Severity: params.Severity,
		Source:   params.Source,
		Keyword:  params.Keyword,
	})
	if err != nil {
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
	rows, err := s.queries.ListDeliveries(ctx, notificationsqlc.ListDeliveriesParams{
		Limit:    int32(pageSize),
		Offset:   int32((page - 1) * pageSize),
		Status:   params.Status,
		Severity: params.Severity,
		Source:   params.Source,
		Keyword:  params.Keyword,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notification deliveries: %w", err)
	}

	items := []*v1alpha1.NotificationDeliveryItem{}
	for _, row := range rows {
		items = append(items, deliveryItemFromRow(deliveryRow{
			ID: row.ID, Source: row.Source, Severity: row.Severity, Title: row.Title, Body: row.Body, Link: row.Link, Channel: row.Channel,
			Status: row.Status, ProviderMessageID: row.ProviderMessageID, ErrorMessage: row.ErrorMessage, CreatedAt: row.CreatedAt, SentAt: row.SentAt,
		}))
	}
	return items, total, nil
}

func (s *SQLStore) GetDelivery(ctx context.Context, id int64) (*v1alpha1.NotificationDeliveryDetail, error) {
	if s.legacy != nil {
		row := s.legacy.QueryRowContext(ctx, `
SELECT id, source, severity, COALESCE(title, ''), body, COALESCE(link, ''), channel, status, provider_message_id, error_message, created_at, sent_at
FROM notification_deliveries
WHERE id = $1
`, id)
		return scanDeliveryDetail(row)
	}
	if s.queries == nil {
		return nil, fmt.Errorf("notification postgres database is not configured")
	}
	row, err := s.queries.GetDelivery(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to get notification delivery: %w", err)
	}
	return deliveryDetailFromRow(deliveryRow{
		ID: row.ID, Source: row.Source, Severity: row.Severity, Title: row.Title, Body: row.Body, Link: row.Link, Channel: row.Channel,
		Status: row.Status, ProviderMessageID: row.ProviderMessageID, ErrorMessage: row.ErrorMessage, CreatedAt: row.CreatedAt, SentAt: row.SentAt,
	}), nil
}

type deliveryFilter struct {
	Status   pgtype.Text
	Severity pgtype.Text
	Source   pgtype.Text
	Keyword  pgtype.Text
}

func deliveryFilterParams(opts ListDeliveriesOptions) deliveryFilter {
	return deliveryFilter{
		Status:   nullableText(strings.TrimSpace(opts.Status)),
		Severity: nullableText(strings.TrimSpace(opts.Severity)),
		Source:   nullableText(strings.TrimSpace(opts.Source)),
		Keyword:  nullableKeyword(opts.Keyword),
	}
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

type deliveryRow struct {
	ID                int64
	Source            string
	Severity          string
	Title             string
	Body              string
	Link              string
	Channel           string
	Status            string
	ProviderMessageID pgtype.Text
	ErrorMessage      pgtype.Text
	CreatedAt         pgtype.Timestamptz
	SentAt            pgtype.Timestamptz
}

func deliveryItemFromRow(row deliveryRow) *v1alpha1.NotificationDeliveryItem {
	item := &v1alpha1.NotificationDeliveryItem{
		ID:                row.ID,
		Source:            row.Source,
		Severity:          row.Severity,
		Title:             row.Title,
		Body:              row.Body,
		Link:              row.Link,
		Channel:           row.Channel,
		Status:            row.Status,
		ProviderMessageID: row.ProviderMessageID.String,
		ErrorMessage:      row.ErrorMessage.String,
		CreatedAt:         formatTime(row.CreatedAt.Time),
	}
	if row.SentAt.Valid {
		item.SentAt = formatTime(row.SentAt.Time)
	}
	return item
}

func deliveryDetailFromRow(row deliveryRow) *v1alpha1.NotificationDeliveryDetail {
	item := deliveryItemFromRow(row)
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
	}
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
	if sentAt.Valid {
		item.SentAt = formatTime(sentAt.Time)
	}
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

func textValue(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: true}
}

func nullableText(value string) pgtype.Text {
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

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
