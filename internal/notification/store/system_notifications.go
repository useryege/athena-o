package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/useryege/athena/internal/notification/delivery"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	notificationsqlc "github.com/useryege/athena/internal/notification/store/sqlc"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type CreateSystemNotificationDeliveryRequest struct {
	Payload      []byte
	Source       string
	Severity     string
	Title        string
	Body         string
	Link         string
	Channel      string
	Status       string
	TelegramChat string
	TopicLabel   string
}

type ListSystemNotificationDeliveriesOptions struct {
	Page         int
	PageSize     int
	Status       string
	Severity     string
	Source       string
	TelegramChat string
	TopicLabel   string
	Keyword      string
}

type ClaimDeliveriesOptions struct {
	Limit       int
	LockedBy    string
	LockTimeout time.Duration
}

type ClaimedSystemNotificationDelivery struct {
	ID              int64
	Source          string
	Severity        string
	Title           string
	Body            string
	Link            string
	Channel         string
	Status          string
	TelegramChat    string
	TopicLabel      string
	MessageThreadID int
	Attempts        int
}

type SystemNotificationDeliveryCounts struct {
	Sending int64
	Unknown int64
	Pending int64
	Retry   int64
	Failed  int64
}

func (s *SQLStore) CreateSystemNotificationDelivery(ctx context.Context, req CreateSystemNotificationDeliveryRequest) (*v1alpha1.SystemNotificationDeliveryDetail, error) {
	if _, err := delivery.DecodePayload(req.Payload); err != nil {
		return nil, err
	}
	if err := s.configured(); err != nil {
		return nil, err
	}
	row, err := s.queries.CreateSystemNotificationDelivery(ctx, notificationsqlc.CreateSystemNotificationDeliveryParams{
		Source: req.Source, Severity: req.Severity, Title: nullableText(req.Title), Body: req.Body,
		Link: nullableText(req.Link), Channel: req.Channel, Status: req.Status,
		TelegramChat: req.TelegramChat, TopicLabel: req.TopicLabel, Payload: req.Payload, PayloadDigest: delivery.PayloadDigest(req.Payload),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create system notification delivery: %w", err)
	}
	return systemDeliveryDetailFromRow(systemDeliveryRow{
		ID: row.ID, Source: row.Source, Severity: row.Severity, Title: row.Title, Body: row.Body,
		Link: row.Link, Channel: row.Channel, Status: row.Status, TelegramChat: row.TelegramChat,
		TopicLabel: row.TopicLabel, ProviderMessageID: row.ProviderMessageID, ErrorMessage: row.ErrorMessage,
		CreatedAt: row.CreatedAt, SentAt: row.SentAt, AuthorizedAt: row.AuthorizedAt, StartedAt: row.StartedAt, ResultAt: row.ResultAt,
	}), nil
}

func (s *SQLStore) EnsureSystemNotificationTopic(ctx context.Context, telegramChat string, label string, create func(context.Context) (int, error)) (int, error) {
	if err := s.transactional(); err != nil {
		return 0, err
	}
	params := notificationsqlc.GetSystemNotificationTopicParams{TelegramChat: telegramChat, Label: label}
	topic, err := s.queries.GetSystemNotificationTopic(ctx, params)
	if err == nil {
		return int(topic.MessageThreadID), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("failed to get system notification topic: %w", err)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to begin system notification topic transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockSystemNotificationTopic(ctx, notificationsqlc.LockSystemNotificationTopicParams{
		TelegramChat: textValue(telegramChat), Label: textValue(label),
	}); err != nil {
		return 0, fmt.Errorf("failed to lock system notification topic: %w", err)
	}
	topic, err = queries.GetSystemNotificationTopic(ctx, params)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("failed to commit system notification topic lookup: %w", err)
		}
		return int(topic.MessageThreadID), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("failed to get system notification topic: %w", err)
	}

	messageThreadID, err := create(ctx)
	if err != nil {
		return 0, err
	}
	if messageThreadID <= 0 {
		return 0, fmt.Errorf("system notification topic returned invalid message thread id")
	}
	if _, err := queries.CreateSystemNotificationTopic(ctx, notificationsqlc.CreateSystemNotificationTopicParams{
		TelegramChat: telegramChat, Label: label, MessageThreadID: int32(messageThreadID),
	}); err != nil {
		return 0, fmt.Errorf("failed to create system notification topic record: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("failed to commit system notification topic: %w", err)
	}
	return messageThreadID, nil
}

func (s *SQLStore) ClaimPendingSystemNotificationDeliveries(ctx context.Context, opts ClaimDeliveriesOptions) ([]ClaimedSystemNotificationDelivery, error) {
	if err := s.configured(); err != nil {
		return nil, err
	}
	limit, lockTimeout := normalizeClaimOptions(opts)
	rows, err := s.queries.ClaimPendingSystemNotificationDeliveries(ctx, notificationsqlc.ClaimPendingSystemNotificationDeliveriesParams{
		Limit: int32(limit), LockedBy: textValue(opts.LockedBy), LockTimeout: intervalValue(lockTimeout),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to claim system notification deliveries: %w", err)
	}
	items := make([]ClaimedSystemNotificationDelivery, 0, len(rows))
	for _, row := range rows {
		items = append(items, ClaimedSystemNotificationDelivery{
			ID: row.ID, Source: row.Source, Severity: row.Severity, Title: row.Title,
			Body: row.Body, Link: row.Link, Channel: row.Channel, Status: row.Status,
			TelegramChat: row.TelegramChat, TopicLabel: row.TopicLabel,
			MessageThreadID: int(row.MessageThreadID), Attempts: int(row.Attempts),
		})
	}
	return items, nil
}

func (s *SQLStore) ListSystemNotificationDeliveries(ctx context.Context, opts ListSystemNotificationDeliveriesOptions) ([]*v1alpha1.SystemNotificationDeliveryItem, int64, error) {
	if err := s.configured(); err != nil {
		return nil, 0, err
	}
	filter := systemDeliveryFilterParams(opts)
	total, err := s.queries.CountSystemNotificationDeliveries(ctx, notificationsqlc.CountSystemNotificationDeliveriesParams{
		Status: filter.Status, Severity: filter.Severity, Source: filter.Source,
		TelegramChat: filter.TelegramChat, TopicLabel: filter.TopicLabel, Keyword: filter.Keyword,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count system notification deliveries: %w", err)
	}
	page, pageSize := opts.Page, opts.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	rows, err := s.queries.ListSystemNotificationDeliveries(ctx, notificationsqlc.ListSystemNotificationDeliveriesParams{
		Limit: int32(pageSize), Offset: int32((page - 1) * pageSize), Status: filter.Status,
		Severity: filter.Severity, Source: filter.Source, TelegramChat: filter.TelegramChat,
		TopicLabel: filter.TopicLabel, Keyword: filter.Keyword,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list system notification deliveries: %w", err)
	}
	items := make([]*v1alpha1.SystemNotificationDeliveryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, systemDeliveryItemFromRow(systemDeliveryRow{
			ID: row.ID, Source: row.Source, Severity: row.Severity, Title: row.Title, Body: row.Body,
			Link: row.Link, Channel: row.Channel, Status: row.Status, TelegramChat: row.TelegramChat,
			TopicLabel: row.TopicLabel, ProviderMessageID: row.ProviderMessageID, ErrorMessage: row.ErrorMessage,
			CreatedAt: row.CreatedAt, SentAt: row.SentAt, AuthorizedAt: row.AuthorizedAt, StartedAt: row.StartedAt, ResultAt: row.ResultAt,
		}))
	}
	return items, total, nil
}

func (s *SQLStore) GetSystemNotificationDelivery(ctx context.Context, id int64) (*v1alpha1.SystemNotificationDeliveryDetail, error) {
	if err := s.configured(); err != nil {
		return nil, err
	}
	row, err := s.queries.GetSystemNotificationDelivery(ctx, id)
	if err != nil {
		return nil, err
	}
	return systemDeliveryDetailFromRow(systemDeliveryRow{
		ID: row.ID, Source: row.Source, Severity: row.Severity, Title: row.Title, Body: row.Body,
		Link: row.Link, Channel: row.Channel, Status: row.Status, TelegramChat: row.TelegramChat,
		TopicLabel: row.TopicLabel, ProviderMessageID: row.ProviderMessageID, ErrorMessage: row.ErrorMessage,
		CreatedAt: row.CreatedAt, SentAt: row.SentAt, AuthorizedAt: row.AuthorizedAt, StartedAt: row.StartedAt, ResultAt: row.ResultAt,
	}), nil
}

func (s *SQLStore) GetSystemNotificationDeliveryCounts(ctx context.Context) (SystemNotificationDeliveryCounts, error) {
	if err := s.configured(); err != nil {
		return SystemNotificationDeliveryCounts{}, err
	}
	row, err := s.queries.GetSystemNotificationDeliveryCounts(ctx)
	if err != nil {
		return SystemNotificationDeliveryCounts{}, fmt.Errorf("failed to get system notification delivery counts: %w", err)
	}
	return SystemNotificationDeliveryCounts{Sending: row.SendingCount, Unknown: row.UnknownCount, Pending: row.PendingCount, Retry: row.RetryCount, Failed: row.FailedCount}, nil
}

func normalizeClaimOptions(opts ClaimDeliveriesOptions) (int, time.Duration) {
	limit := opts.Limit
	if limit < 1 {
		limit = 1
	}
	lockTimeout := opts.LockTimeout
	if lockTimeout <= 0 {
		lockTimeout = time.Minute
	}
	return limit, lockTimeout
}

type systemDeliveryFilter struct {
	Status, Severity, Source, TelegramChat, TopicLabel, Keyword pgtype.Text
}

func systemDeliveryFilterParams(opts ListSystemNotificationDeliveriesOptions) systemDeliveryFilter {
	return systemDeliveryFilter{
		Status: nullableText(opts.Status), Severity: nullableText(opts.Severity),
		Source: nullableText(opts.Source), TelegramChat: nullableText(opts.TelegramChat),
		TopicLabel: nullableText(opts.TopicLabel), Keyword: nullableKeyword(opts.Keyword),
	}
}

type systemDeliveryRow struct {
	ID                int64
	Source            string
	Severity          string
	Title             string
	Body              string
	Link              string
	Channel           string
	Status            string
	TelegramChat      string
	TopicLabel        string
	ProviderMessageID pgtype.Text
	ErrorMessage      pgtype.Text
	CreatedAt         pgtype.Timestamptz
	SentAt            pgtype.Timestamptz
	AuthorizedAt      pgtype.Timestamptz
	StartedAt         pgtype.Timestamptz
	ResultAt          pgtype.Timestamptz
}

func systemDeliveryItemFromRow(row systemDeliveryRow) *v1alpha1.SystemNotificationDeliveryItem {
	item := &v1alpha1.SystemNotificationDeliveryItem{
		ID: row.ID, Source: row.Source, Severity: row.Severity, Title: row.Title, Body: row.Body,
		Link: row.Link, Channel: row.Channel, Status: row.Status, TelegramChat: row.TelegramChat,
		TopicLabel: row.TopicLabel, ProviderMessageID: row.ProviderMessageID.String,
		ErrorMessage: row.ErrorMessage.String, CreatedAt: formatTime(row.CreatedAt.Time),
	}
	if row.SentAt.Valid {
		item.SentAt = formatTime(row.SentAt.Time)
	}
	if row.AuthorizedAt.Valid {
		item.AuthorizedAt = formatTime(row.AuthorizedAt.Time)
	}
	if row.StartedAt.Valid {
		item.StartedAt = formatTime(row.StartedAt.Time)
	}
	if row.ResultAt.Valid {
		item.ResultAt = formatTime(row.ResultAt.Time)
	}
	return item
}

func systemDeliveryDetailFromRow(row systemDeliveryRow) *v1alpha1.SystemNotificationDeliveryDetail {
	item := systemDeliveryItemFromRow(row)
	return &v1alpha1.SystemNotificationDeliveryDetail{
		ID: item.ID, Source: item.Source, Severity: item.Severity, Title: item.Title, Body: item.Body,
		Link: item.Link, Channel: item.Channel, Status: item.Status, TelegramChat: item.TelegramChat,
		TopicLabel: item.TopicLabel, ProviderMessageID: item.ProviderMessageID,
		ErrorMessage: item.ErrorMessage, CreatedAt: item.CreatedAt, SentAt: item.SentAt, AuthorizedAt: item.AuthorizedAt, StartedAt: item.StartedAt, ResultAt: item.ResultAt,
	}
}
