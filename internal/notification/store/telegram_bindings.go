package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	notificationsqlc "github.com/useryege/athena/internal/notification/store/sqlc"
)

var (
	ErrTelegramBindingAttemptInvalid = errors.New("telegram binding attempt is invalid")
	ErrTelegramBindingAttemptExpired = errors.New("telegram binding attempt expired")
	ErrTelegramIdentityInUse         = errors.New("telegram identity is already bound")
)

const (
	TelegramBindingStatusConnected   = "connected"
	TelegramBindingStatusUnreachable = "unreachable"

	TelegramBindingAttemptStatusPending = "pending"
	TelegramBindingAttemptStatusFailed  = "failed"

	TelegramBindingFailureExpired              = "expired"
	TelegramBindingFailureTelegramIdentityUsed = "telegram_identity_in_use"
)

type TelegramBinding struct {
	AccountID           string
	TelegramUserID      int64
	TelegramChatID      int64
	TelegramUsername    string
	TelegramDisplayName string
	Status              string
	Revision            int64
	BoundAt             time.Time
	UpdatedAt           time.Time
	LastError           string
}

type TelegramBindingAttempt struct {
	ID            string
	AccountID     string
	TokenDigest   []byte
	Status        string
	FailureReason string
	ExpiresAt     time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateTelegramBindingAttemptRequest struct {
	ID          string
	AccountID   string
	TokenDigest []byte
	ExpiresAt   time.Time
}

type CompleteTelegramBindingAttemptRequest struct {
	TokenDigest         []byte
	TelegramUserID      int64
	TelegramChatID      int64
	TelegramUsername    string
	TelegramDisplayName string
}

type TelegramPollingState struct {
	NextUpdateID int64
	LastPollAt   time.Time
	LastUpdateAt time.Time
	UpdatedAt    time.Time
}

func (s *SQLStore) GetTelegramBinding(ctx context.Context, accountID string) (*TelegramBinding, error) {
	if err := s.configured(); err != nil {
		return nil, err
	}
	accountUUID, err := uuidValue(accountID)
	if err != nil {
		return nil, err
	}
	row, err := s.queries.GetTelegramBinding(ctx, accountUUID)
	if err != nil {
		return nil, err
	}
	return telegramBindingFromValues(
		row.AccountID, row.TelegramUserID, row.TelegramChatID, row.TelegramUsername,
		row.TelegramDisplayName, row.Status, row.Revision, row.BoundAt, row.UpdatedAt, row.LastError,
	), nil
}

func (s *SQLStore) GetTelegramBindingAttempt(ctx context.Context, accountID string) (*TelegramBindingAttempt, error) {
	if err := s.configured(); err != nil {
		return nil, err
	}
	accountUUID, err := uuidValue(accountID)
	if err != nil {
		return nil, err
	}
	row, err := s.queries.GetTelegramBindingAttempt(ctx, accountUUID)
	if err != nil {
		return nil, err
	}
	attempt := telegramBindingAttemptFromValues(
		row.ID, row.AccountID, row.TokenDigest, row.Status, row.FailureReason,
		row.ExpiresAt, row.CreatedAt, row.UpdatedAt,
	)
	return projectExpiredTelegramBindingAttempt(attempt, time.Now().UTC()), nil
}

func (s *SQLStore) CreateTelegramBindingAttempt(ctx context.Context, req CreateTelegramBindingAttemptRequest) (*TelegramBindingAttempt, error) {
	if err := s.transactional(); err != nil {
		return nil, err
	}
	accountUUID, err := uuidValue(req.AccountID)
	if err != nil {
		return nil, err
	}
	attemptUUID, err := uuidValue(req.ID)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin telegram binding attempt transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockTelegramBindingAccount(ctx, accountUUID); err != nil {
		return nil, fmt.Errorf("failed to lock telegram binding account: %w", err)
	}
	row, err := queries.UpsertTelegramBindingAttempt(ctx, notificationsqlc.UpsertTelegramBindingAttemptParams{
		ID: attemptUUID, AccountID: accountUUID, TokenDigest: append([]byte(nil), req.TokenDigest...),
		ExpiresAt: timestamptzValue(req.ExpiresAt),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to persist telegram binding attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit telegram binding attempt: %w", err)
	}
	return telegramBindingAttemptFromValues(
		row.ID, row.AccountID, row.TokenDigest, row.Status, row.FailureReason,
		row.ExpiresAt, row.CreatedAt, row.UpdatedAt,
	), nil
}

func (s *SQLStore) DeleteTelegramBindingAttempt(ctx context.Context, accountID string) (bool, error) {
	if err := s.transactional(); err != nil {
		return false, err
	}
	accountUUID, err := uuidValue(accountID)
	if err != nil {
		return false, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin telegram binding attempt delete transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockTelegramBindingAccount(ctx, accountUUID); err != nil {
		return false, fmt.Errorf("failed to lock telegram binding account: %w", err)
	}
	rows, err := queries.DeleteTelegramBindingAttemptByAccount(ctx, accountUUID)
	if err != nil {
		return false, fmt.Errorf("failed to delete telegram binding attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("failed to commit telegram binding attempt delete: %w", err)
	}
	return rows > 0, nil
}

func (s *SQLStore) DeleteTelegramBinding(ctx context.Context, accountID string) (bool, error) {
	if err := s.transactional(); err != nil {
		return false, err
	}
	accountUUID, err := uuidValue(accountID)
	if err != nil {
		return false, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to begin telegram binding delete transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockTelegramBindingAccount(ctx, accountUUID); err != nil {
		return false, fmt.Errorf("failed to lock telegram binding account: %w", err)
	}
	deleted := true
	if _, err := queries.DeleteTelegramBinding(ctx, accountUUID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			deleted = false
		} else {
			return false, fmt.Errorf("failed to delete telegram binding: %w", err)
		}
	}
	if _, err := queries.DeleteTelegramBindingAttemptByAccount(ctx, accountUUID); err != nil {
		return false, fmt.Errorf("failed to delete stale telegram binding attempt: %w", err)
	}
	if _, err := queries.CancelPendingAccountNotificationDeliveries(ctx, accountUUID); err != nil {
		return false, fmt.Errorf("failed to cancel pending account notification deliveries: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("failed to commit telegram binding delete: %w", err)
	}
	return deleted, nil
}

func (s *SQLStore) CompleteTelegramBindingAttempt(ctx context.Context, req CompleteTelegramBindingAttemptRequest) (*TelegramBinding, error) {
	if err := s.transactional(); err != nil {
		return nil, err
	}
	attemptAccountID, err := s.queries.GetTelegramBindingAttemptAccountByToken(ctx, req.TokenDigest)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTelegramBindingAttemptInvalid
		}
		return nil, fmt.Errorf("failed to resolve telegram binding attempt account: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin telegram binding completion transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockTelegramBindingAccount(ctx, attemptAccountID); err != nil {
		return nil, fmt.Errorf("failed to lock telegram binding account: %w", err)
	}
	if err := queries.LockTelegramBindingIdentity(ctx, notificationsqlc.LockTelegramBindingIdentityParams{
		TelegramUserID: req.TelegramUserID,
		TelegramChatID: req.TelegramChatID,
	}); err != nil {
		return nil, fmt.Errorf("failed to lock telegram binding identity: %w", err)
	}
	attemptRow, err := queries.GetTelegramBindingAttemptByTokenForUpdate(ctx, req.TokenDigest)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrTelegramBindingAttemptInvalid
		}
		return nil, fmt.Errorf("failed to get telegram binding attempt: %w", err)
	}
	if uuidString(attemptRow.AccountID) != uuidString(attemptAccountID) {
		return nil, ErrTelegramBindingAttemptInvalid
	}
	attempt := telegramBindingAttemptFromValues(
		attemptRow.ID, attemptRow.AccountID, attemptRow.TokenDigest, attemptRow.Status,
		attemptRow.FailureReason, attemptRow.ExpiresAt, attemptRow.CreatedAt, attemptRow.UpdatedAt,
	)
	if attempt.Status != TelegramBindingAttemptStatusPending {
		return nil, ErrTelegramBindingAttemptInvalid
	}
	if !attempt.ExpiresAt.After(time.Now().UTC()) {
		if _, err := queries.FailTelegramBindingAttempt(ctx, notificationsqlc.FailTelegramBindingAttemptParams{
			ID: attemptRow.ID, FailureReason: textValue(TelegramBindingFailureExpired),
		}); err != nil {
			return nil, fmt.Errorf("failed to expire telegram binding attempt: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit expired telegram binding attempt: %w", err)
		}
		return nil, ErrTelegramBindingAttemptExpired
	}
	oldBinding, oldBindingErr := queries.GetTelegramBinding(ctx, attemptRow.AccountID)
	if oldBindingErr != nil && !errors.Is(oldBindingErr, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to check telegram binding account: %w", oldBindingErr)
	}
	identityBindings, err := queries.ListTelegramBindingsByIdentityForShare(ctx, notificationsqlc.ListTelegramBindingsByIdentityForShareParams{
		TelegramUserID: req.TelegramUserID, TelegramChatID: req.TelegramChatID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to check telegram identity: %w", err)
	}
	for _, identityBinding := range identityBindings {
		if uuidString(identityBinding.AccountID) == uuidString(attemptRow.AccountID) {
			continue
		}
		if err := failTelegramBindingAttempt(ctx, queries, attemptRow.ID, TelegramBindingFailureTelegramIdentityUsed); err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("failed to commit used telegram identity attempt: %w", err)
		}
		return nil, ErrTelegramIdentityInUse
	}
	revision, err := queries.NextTelegramBindingRevision(ctx, attemptRow.AccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to allocate telegram binding revision: %w", err)
	}
	if oldBindingErr == nil {
		if _, err := queries.CancelPendingAccountNotificationDeliveriesForBinding(ctx, notificationsqlc.CancelPendingAccountNotificationDeliveriesForBindingParams{
			AccountID: oldBinding.AccountID, TelegramChatID: oldBinding.TelegramChatID,
			BindingRevision: oldBinding.Revision,
		}); err != nil {
			return nil, fmt.Errorf("failed to cancel superseded account notification deliveries: %w", err)
		}
	}
	bindingRow, err := queries.ReplaceTelegramBinding(ctx, notificationsqlc.ReplaceTelegramBindingParams{
		AccountID: attemptRow.AccountID, TelegramUserID: req.TelegramUserID,
		TelegramChatID: req.TelegramChatID, TelegramUsername: nullableText(req.TelegramUsername),
		TelegramDisplayName: strings.TrimSpace(req.TelegramDisplayName), Revision: revision,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram binding: %w", err)
	}
	if _, err := queries.DeleteTelegramBindingAttemptByID(ctx, attemptRow.ID); err != nil {
		return nil, fmt.Errorf("failed to consume telegram binding attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit telegram binding: %w", err)
	}
	return telegramBindingFromValues(
		bindingRow.AccountID, bindingRow.TelegramUserID, bindingRow.TelegramChatID,
		bindingRow.TelegramUsername, bindingRow.TelegramDisplayName, bindingRow.Status,
		bindingRow.Revision, bindingRow.BoundAt, bindingRow.UpdatedAt, bindingRow.LastError,
	), nil
}

func (s *SQLStore) MarkTelegramBindingUnreachable(
	ctx context.Context,
	accountID string,
	chatID int64,
	revision int64,
	reason string,
) error {
	if err := s.transactional(); err != nil {
		return err
	}
	accountUUID, err := uuidValue(accountID)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin telegram binding unreachable transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockTelegramBindingAccount(ctx, accountUUID); err != nil {
		return fmt.Errorf("failed to lock telegram binding account: %w", err)
	}
	_, err = queries.MarkTelegramBindingUnreachable(ctx, notificationsqlc.MarkTelegramBindingUnreachableParams{
		AccountID: accountUUID, TelegramChatID: chatID, Revision: revision, LastError: textValue(reason),
	})
	if err != nil {
		return fmt.Errorf("failed to mark telegram binding unreachable: %w", err)
	}
	if _, err := queries.CancelPendingAccountNotificationDeliveriesForBinding(ctx, notificationsqlc.CancelPendingAccountNotificationDeliveriesForBindingParams{
		AccountID: accountUUID, TelegramChatID: chatID, BindingRevision: revision,
	}); err != nil {
		return fmt.Errorf("failed to cancel unreachable account notification deliveries: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit telegram binding unreachable state: %w", err)
	}
	return nil
}

func (s *SQLStore) MarkTelegramBindingStatusByIdentity(ctx context.Context, userID, chatID int64, status, reason string) error {
	if err := s.transactional(); err != nil {
		return err
	}
	identityParams := notificationsqlc.GetTelegramBindingByIdentityParams{
		TelegramUserID: userID, TelegramChatID: chatID,
	}
	observed, err := s.queries.GetTelegramBindingByIdentity(ctx, identityParams)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to resolve telegram binding identity: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin telegram binding status transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := notificationsqlc.New(tx)
	if err := queries.LockTelegramBindingAccount(ctx, observed.AccountID); err != nil {
		return fmt.Errorf("failed to lock telegram binding account: %w", err)
	}
	current, err := queries.GetTelegramBindingForShare(ctx, observed.AccountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to revalidate telegram binding identity: %w", err)
	}
	if current.TelegramUserID != userID || current.TelegramChatID != chatID {
		return nil
	}
	switch status {
	case TelegramBindingStatusConnected:
		if current.Status != TelegramBindingStatusConnected {
			if _, err := queries.MarkTelegramBindingConnectedByIdentity(ctx, notificationsqlc.MarkTelegramBindingConnectedByIdentityParams{
				TelegramUserID: userID, TelegramChatID: chatID,
			}); err != nil {
				return fmt.Errorf("failed to mark telegram binding connected: %w", err)
			}
		}
	case TelegramBindingStatusUnreachable:
		if current.Status != TelegramBindingStatusUnreachable {
			if _, err := queries.MarkTelegramBindingUnreachableByIdentity(ctx, notificationsqlc.MarkTelegramBindingUnreachableByIdentityParams{
				TelegramUserID: userID, TelegramChatID: chatID, LastError: textValue(reason),
			}); err != nil {
				return fmt.Errorf("failed to mark telegram binding unreachable: %w", err)
			}
		}
		if _, err := queries.CancelPendingAccountNotificationDeliveriesForBinding(ctx, notificationsqlc.CancelPendingAccountNotificationDeliveriesForBindingParams{
			AccountID: current.AccountID, TelegramChatID: current.TelegramChatID, BindingRevision: current.Revision,
		}); err != nil {
			return fmt.Errorf("failed to cancel unreachable account notification deliveries: %w", err)
		}
	default:
		return fmt.Errorf("unsupported telegram binding status %q", status)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit telegram binding status: %w", err)
	}
	return nil
}

func (s *SQLStore) GetTelegramPollingState(ctx context.Context) (TelegramPollingState, error) {
	if err := s.configured(); err != nil {
		return TelegramPollingState{}, err
	}
	row, err := s.queries.GetTelegramPollingState(ctx)
	if err != nil {
		return TelegramPollingState{}, fmt.Errorf("failed to get telegram polling state: %w", err)
	}
	state := TelegramPollingState{NextUpdateID: row.NextUpdateID, UpdatedAt: row.UpdatedAt.Time}
	if row.LastPollAt.Valid {
		state.LastPollAt = row.LastPollAt.Time
	}
	if row.LastUpdateAt.Valid {
		state.LastUpdateAt = row.LastUpdateAt.Time
	}
	return state, nil
}

func (s *SQLStore) AdvanceTelegramPollingState(ctx context.Context, nextUpdateID int64, lastUpdateAt time.Time) (TelegramPollingState, error) {
	if err := s.configured(); err != nil {
		return TelegramPollingState{}, err
	}
	row, err := s.queries.AdvanceTelegramPollingState(ctx, notificationsqlc.AdvanceTelegramPollingStateParams{
		NextUpdateID: nextUpdateID, LastUpdateAt: timestamptzValue(lastUpdateAt),
	})
	if err != nil {
		return TelegramPollingState{}, fmt.Errorf("failed to advance telegram polling state: %w", err)
	}
	state := TelegramPollingState{NextUpdateID: row.NextUpdateID, UpdatedAt: row.UpdatedAt.Time}
	if row.LastPollAt.Valid {
		state.LastPollAt = row.LastPollAt.Time
	}
	if row.LastUpdateAt.Valid {
		state.LastUpdateAt = row.LastUpdateAt.Time
	}
	return state, nil
}

func (s *SQLStore) RecordTelegramPoll(ctx context.Context, polledAt time.Time) (TelegramPollingState, error) {
	if err := s.configured(); err != nil {
		return TelegramPollingState{}, err
	}
	row, err := s.queries.RecordTelegramPoll(ctx, timestamptzValue(polledAt))
	if err != nil {
		return TelegramPollingState{}, fmt.Errorf("failed to record telegram poll: %w", err)
	}
	state := TelegramPollingState{NextUpdateID: row.NextUpdateID, UpdatedAt: row.UpdatedAt.Time}
	if row.LastPollAt.Valid {
		state.LastPollAt = row.LastPollAt.Time
	}
	if row.LastUpdateAt.Valid {
		state.LastUpdateAt = row.LastUpdateAt.Time
	}
	return state, nil
}

func failTelegramBindingAttempt(ctx context.Context, queries *notificationsqlc.Queries, id pgtype.UUID, reason string) error {
	if _, err := queries.FailTelegramBindingAttempt(ctx, notificationsqlc.FailTelegramBindingAttemptParams{
		ID: id, FailureReason: textValue(reason),
	}); err != nil {
		return fmt.Errorf("failed to mark telegram binding attempt failed: %w", err)
	}
	return nil
}

func projectExpiredTelegramBindingAttempt(attempt *TelegramBindingAttempt, now time.Time) *TelegramBindingAttempt {
	if attempt != nil && attempt.Status == TelegramBindingAttemptStatusPending && !attempt.ExpiresAt.After(now) {
		copyAttempt := *attempt
		copyAttempt.Status = TelegramBindingAttemptStatusFailed
		copyAttempt.FailureReason = TelegramBindingFailureExpired
		return &copyAttempt
	}
	return attempt
}

func telegramBindingFromValues(
	accountID pgtype.UUID,
	telegramUserID, telegramChatID int64,
	telegramUsername, telegramDisplayName, status string,
	revision int64,
	boundAt, updatedAt pgtype.Timestamptz,
	lastError string,
) *TelegramBinding {
	return &TelegramBinding{
		AccountID: uuidString(accountID), TelegramUserID: telegramUserID, TelegramChatID: telegramChatID,
		TelegramUsername: telegramUsername, TelegramDisplayName: telegramDisplayName, Status: status,
		Revision: revision, BoundAt: boundAt.Time, UpdatedAt: updatedAt.Time, LastError: lastError,
	}
}

func telegramBindingAttemptFromValues(
	id, accountID pgtype.UUID,
	tokenDigest []byte,
	status string,
	failureReason pgtype.Text,
	expiresAt, createdAt, updatedAt pgtype.Timestamptz,
) *TelegramBindingAttempt {
	return &TelegramBindingAttempt{
		ID: uuidString(id), AccountID: uuidString(accountID), TokenDigest: append([]byte(nil), tokenDigest...),
		Status: status, FailureReason: failureReason.String, ExpiresAt: expiresAt.Time,
		CreatedAt: createdAt.Time, UpdatedAt: updatedAt.Time,
	}
}
