package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	notificationsqlc "github.com/useryege/athena/internal/notification/store/sqlc"
)

var (
	ErrAccountNotificationIdempotencyConflict = errors.New("account notification idempotency conflict")
	ErrAccountNotificationBindingChanged      = errors.New("account notification binding changed")
)

const (
	AccountNotificationRecipientConnected   = "connected"
	AccountNotificationRecipientNotBound    = "not_bound"
	AccountNotificationRecipientUnreachable = "unreachable"
)

type CreateAccountNotificationDeliveryRequest struct {
	AccountID      string
	IdempotencyKey string
	PayloadDigest  []byte
	Source         string
	Severity       string
	Title          string
	Body           string
	Link           string
	Channel        string
	Status         string
}

type AccountNotificationDelivery struct {
	ID                int64
	AccountID         string
	IdempotencyKey    string
	PayloadDigest     []byte
	Source            string
	Severity          string
	Title             string
	Body              string
	Link              string
	Channel           string
	Status            string
	TelegramChatID    int64
	BindingRevision   int64
	ProviderMessageID string
	ErrorMessage      string
	CreatedAt         time.Time
	SentAt            time.Time
}

type EnqueueAccountNotificationResult struct {
	RecipientStatus string
	Delivery        *AccountNotificationDelivery
	Created         bool
}

type ClaimedAccountNotificationDelivery struct {
	ID              int64
	AccountID       string
	Source          string
	Severity        string
	Title           string
	Body            string
	Link            string
	Channel         string
	Status          string
	TelegramChatID  int64
	BindingRevision int64
	Attempts        int
}

type AccountNotificationRuntimeCounts struct {
	Pending             int64
	Retry               int64
	Failed              int64
	UnreachableBindings int64
}

func (s *SQLStore) EnqueueAccountNotification(ctx context.Context, req CreateAccountNotificationDeliveryRequest) (EnqueueAccountNotificationResult, error) {
	if err := s.transactional(); err != nil {
		return EnqueueAccountNotificationResult{}, err
	}
	accountUUID, err := uuidValue(req.AccountID)
	if err != nil {
		return EnqueueAccountNotificationResult{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return EnqueueAccountNotificationResult{}, fmt.Errorf("failed to begin account notification enqueue transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockTelegramBindingAccount(ctx, accountUUID); err != nil {
		return EnqueueAccountNotificationResult{}, fmt.Errorf("failed to lock account notification recipient: %w", err)
	}
	existing, err := queries.GetAccountNotificationDeliveryByIdempotency(ctx, notificationsqlc.GetAccountNotificationDeliveryByIdempotencyParams{
		AccountID: accountUUID, Source: req.Source, IdempotencyKey: req.IdempotencyKey,
	})
	if err == nil {
		if !bytes.Equal(existing.PayloadDigest, req.PayloadDigest) {
			return EnqueueAccountNotificationResult{}, ErrAccountNotificationIdempotencyConflict
		}
		return EnqueueAccountNotificationResult{
			RecipientStatus: AccountNotificationRecipientConnected,
			Delivery: accountNotificationDeliveryFromValues(
				existing.ID, existing.AccountID, existing.IdempotencyKey, existing.PayloadDigest,
				existing.Source, existing.Severity, existing.Title, existing.Body, existing.Link,
				existing.Channel, existing.Status, existing.TelegramChatID, existing.BindingRevision,
				existing.ProviderMessageID, existing.ErrorMessage, existing.CreatedAt, existing.SentAt,
			),
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return EnqueueAccountNotificationResult{}, fmt.Errorf("failed to check account notification idempotency: %w", err)
	}
	binding, err := queries.GetTelegramBindingForShare(ctx, accountUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return EnqueueAccountNotificationResult{RecipientStatus: AccountNotificationRecipientNotBound}, nil
		}
		return EnqueueAccountNotificationResult{}, fmt.Errorf("failed to resolve account notification recipient: %w", err)
	}
	if binding.Status != TelegramBindingStatusConnected {
		return EnqueueAccountNotificationResult{RecipientStatus: AccountNotificationRecipientUnreachable}, nil
	}

	params := notificationsqlc.CreateAccountNotificationDeliveryParams{
		AccountID: accountUUID, IdempotencyKey: req.IdempotencyKey,
		PayloadDigest: append([]byte(nil), req.PayloadDigest...), Source: req.Source,
		Severity: req.Severity, Title: nullableText(req.Title), Body: req.Body,
		Link: nullableText(req.Link), Channel: req.Channel, Status: req.Status,
		TelegramChatID: binding.TelegramChatID, BindingRevision: binding.Revision,
	}
	row, err := queries.CreateAccountNotificationDelivery(ctx, params)
	created := err == nil
	var delivery *AccountNotificationDelivery
	if errors.Is(err, pgx.ErrNoRows) {
		existing, getErr := queries.GetAccountNotificationDeliveryByIdempotency(ctx, notificationsqlc.GetAccountNotificationDeliveryByIdempotencyParams{
			AccountID: accountUUID, Source: req.Source, IdempotencyKey: req.IdempotencyKey,
		})
		if getErr != nil {
			return EnqueueAccountNotificationResult{}, fmt.Errorf("failed to resolve account notification idempotency result: %w", getErr)
		}
		if !bytes.Equal(existing.PayloadDigest, req.PayloadDigest) {
			return EnqueueAccountNotificationResult{}, ErrAccountNotificationIdempotencyConflict
		}
		delivery = accountNotificationDeliveryFromValues(
			existing.ID, existing.AccountID, existing.IdempotencyKey, existing.PayloadDigest,
			existing.Source, existing.Severity, existing.Title, existing.Body, existing.Link,
			existing.Channel, existing.Status, existing.TelegramChatID, existing.BindingRevision,
			existing.ProviderMessageID, existing.ErrorMessage, existing.CreatedAt, existing.SentAt,
		)
	} else if err != nil {
		return EnqueueAccountNotificationResult{}, fmt.Errorf("failed to create account notification delivery: %w", err)
	} else {
		delivery = accountNotificationDeliveryFromValues(
			row.ID, row.AccountID, row.IdempotencyKey, row.PayloadDigest, row.Source, row.Severity,
			row.Title, row.Body, row.Link, row.Channel, row.Status, row.TelegramChatID,
			row.BindingRevision, row.ProviderMessageID, row.ErrorMessage, row.CreatedAt, row.SentAt,
		)
	}
	if err := tx.Commit(ctx); err != nil {
		return EnqueueAccountNotificationResult{}, fmt.Errorf("failed to commit account notification enqueue: %w", err)
	}
	return EnqueueAccountNotificationResult{
		RecipientStatus: AccountNotificationRecipientConnected,
		Delivery:        delivery,
		Created:         created,
	}, nil
}

func (s *SQLStore) ClaimPendingAccountNotificationDeliveries(ctx context.Context, opts ClaimDeliveriesOptions) ([]ClaimedAccountNotificationDelivery, error) {
	if err := s.configured(); err != nil {
		return nil, err
	}
	limit, lockTimeout := normalizeClaimOptions(opts)
	rows, err := s.queries.ClaimPendingAccountNotificationDeliveries(ctx, notificationsqlc.ClaimPendingAccountNotificationDeliveriesParams{
		Limit: int32(limit), LockedBy: textValue(opts.LockedBy), LockTimeout: intervalValue(lockTimeout),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to claim account notification deliveries: %w", err)
	}
	items := make([]ClaimedAccountNotificationDelivery, 0, len(rows))
	for _, row := range rows {
		items = append(items, ClaimedAccountNotificationDelivery{
			ID: row.ID, AccountID: uuidString(row.AccountID), Source: row.Source,
			Severity: row.Severity, Title: row.Title, Body: row.Body, Link: row.Link,
			Channel: row.Channel, Status: row.Status, TelegramChatID: row.TelegramChatID,
			BindingRevision: row.BindingRevision, Attempts: int(row.Attempts),
		})
	}
	return items, nil
}

// SendAccountNotificationWithBindingLock closes the gap between queue claiming and delivery.
// Binding mutations take the same account advisory lock, so a successful return guarantees
// the message was sent only while the queued binding revision was still current and connected.
func (s *SQLStore) SendAccountNotificationWithBindingLock(
	ctx context.Context,
	accountID string,
	chatID int64,
	revision int64,
	deliveryID int64,
	send func(context.Context) (string, error),
) (string, error) {
	if err := s.transactional(); err != nil {
		return "", err
	}
	if send == nil {
		return "", errors.New("account notification send callback is required")
	}
	accountUUID, err := uuidValue(accountID)
	if err != nil {
		return "", err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to begin account notification send transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockTelegramBindingAccount(ctx, accountUUID); err != nil {
		return "", fmt.Errorf("failed to lock account notification binding: %w", err)
	}
	binding, err := queries.GetTelegramBindingForShare(ctx, accountUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAccountNotificationBindingChanged
		}
		return "", fmt.Errorf("failed to revalidate account notification binding: %w", err)
	}
	if binding.Status != TelegramBindingStatusConnected ||
		binding.TelegramChatID != chatID || binding.Revision != revision {
		return "", ErrAccountNotificationBindingChanged
	}
	if _, err := queries.GetPendingAccountNotificationDeliveryForUpdate(ctx, notificationsqlc.GetPendingAccountNotificationDeliveryForUpdateParams{
		ID: deliveryID, AccountID: accountUUID, TelegramChatID: chatID, BindingRevision: revision,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrAccountNotificationBindingChanged
		}
		return "", fmt.Errorf("failed to revalidate pending account notification delivery: %w", err)
	}
	providerMessageID, err := send(ctx)
	if err != nil {
		return "", err
	}
	if err := queries.MarkAccountNotificationDeliverySent(ctx, notificationsqlc.MarkAccountNotificationDeliverySentParams{
		ID: deliveryID, ProviderMessageID: textValue(providerMessageID),
	}); err != nil {
		return "", fmt.Errorf("failed to mark account notification delivery sent: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("failed to commit account notification delivery: %w", err)
	}
	return providerMessageID, nil
}

func (s *SQLStore) MarkAccountNotificationDeliveryFailed(ctx context.Context, delivery ClaimedAccountNotificationDelivery, message string) error {
	if err := s.transactional(); err != nil {
		return err
	}
	accountUUID, err := uuidValue(delivery.AccountID)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin account notification failure transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockTelegramBindingAccount(ctx, accountUUID); err != nil {
		return fmt.Errorf("failed to lock account notification failure transition: %w", err)
	}
	if _, err := queries.MarkAccountNotificationDeliveryFailed(ctx, notificationsqlc.MarkAccountNotificationDeliveryFailedParams{
		ID: delivery.ID, AccountID: accountUUID, TelegramChatID: delivery.TelegramChatID,
		BindingRevision: delivery.BindingRevision, ErrorMessage: textValue(message),
	}); err != nil {
		return fmt.Errorf("failed to mark account notification delivery failed: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit account notification failure transition: %w", err)
	}
	return nil
}

func (s *SQLStore) ScheduleAccountNotificationDeliveryRetry(ctx context.Context, delivery ClaimedAccountNotificationDelivery, message string, nextAttemptAt time.Time) error {
	if err := s.transactional(); err != nil {
		return err
	}
	accountUUID, err := uuidValue(delivery.AccountID)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin account notification retry transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockTelegramBindingAccount(ctx, accountUUID); err != nil {
		return fmt.Errorf("failed to lock account notification retry transition: %w", err)
	}
	if _, err := queries.ScheduleAccountNotificationDeliveryRetry(ctx, notificationsqlc.ScheduleAccountNotificationDeliveryRetryParams{
		ID: delivery.ID, AccountID: accountUUID, TelegramChatID: delivery.TelegramChatID,
		BindingRevision: delivery.BindingRevision, ErrorMessage: textValue(message),
		NextAttemptAt: timestamptzValue(nextAttemptAt),
	}); err != nil {
		return fmt.Errorf("failed to schedule account notification delivery retry: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit account notification retry transition: %w", err)
	}
	return nil
}

func (s *SQLStore) GetAccountNotificationRuntimeCounts(ctx context.Context) (AccountNotificationRuntimeCounts, error) {
	if err := s.configured(); err != nil {
		return AccountNotificationRuntimeCounts{}, err
	}
	row, err := s.queries.GetAccountNotificationRuntimeCounts(ctx)
	if err != nil {
		return AccountNotificationRuntimeCounts{}, fmt.Errorf("failed to get account notification runtime counts: %w", err)
	}
	return AccountNotificationRuntimeCounts{
		Pending: row.PendingCount, Retry: row.RetryCount, Failed: row.FailedCount,
		UnreachableBindings: row.UnreachableBindingCount,
	}, nil
}

func accountNotificationDeliveryFromValues(
	id int64,
	accountID pgtype.UUID,
	idempotencyKey string,
	payloadDigest []byte,
	source, severity, title, body, link, channel, status string,
	telegramChatID, bindingRevision int64,
	providerMessageID, errorMessage pgtype.Text,
	createdAt, sentAt pgtype.Timestamptz,
) *AccountNotificationDelivery {
	delivery := &AccountNotificationDelivery{
		ID: id, AccountID: uuidString(accountID), IdempotencyKey: idempotencyKey,
		PayloadDigest: append([]byte(nil), payloadDigest...), Source: source, Severity: severity,
		Title: title, Body: body, Link: link, Channel: channel, Status: status,
		TelegramChatID: telegramChatID, BindingRevision: bindingRevision,
		ProviderMessageID: providerMessageID.String, ErrorMessage: errorMessage.String,
		CreatedAt: createdAt.Time,
	}
	if sentAt.Valid {
		delivery.SentAt = sentAt.Time
	}
	return delivery
}
