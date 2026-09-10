package store

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	q "github.com/useryege/athena/internal/notification/store/sqlc"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

const telegramBindingConfirmationText = "ATHENA Telegram notifications are connected. You can return to ATHENA."
const telegramBindingInvalidText = "This ATHENA Telegram binding link is invalid or expired. Create a new attempt in ATHENA."
const telegramBindingIdentityUsedText = "This Telegram identity is already connected and cannot be used."

// ApplyBotUpdate commits consumption, binding mutations, reply and offset together.
// A caller may replay this database operation after any error, including an uncertain commit.
func (s *SQLStore) ApplyBotUpdate(ctx context.Context, update utiltelegram.Update) error {
	if err := s.transactional(); err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	queries := q.New(tx)
	if _, err = queries.ConsumeTelegramUpdate(ctx, update.ID); errors.Is(err, pgx.ErrNoRows) {
		return nil
	} else if err != nil {
		return err
	}
	if update.Message != nil {
		err = applyBotMessageTx(ctx, tx, update.ID, update.Message)
	} else if member := update.MyChatMember; member != nil && validPrivateIdentity(member.ChatType, member.IsBot, member.UserID, member.ChatID) {
		switch member.NewStatus {
		case "left", "kicked":
			err = markTelegramBindingStatusByIdentityTx(ctx, tx, member.UserID, member.ChatID, TelegramBindingStatusUnreachable, "telegram bot blocked")
		case "member":
			err = markTelegramBindingStatusByIdentityTx(ctx, tx, member.UserID, member.ChatID, TelegramBindingStatusConnected, "")
		}
	}
	if err != nil {
		return err
	}
	if _, err = queries.AdvanceTelegramPollingState(ctx, q.AdvanceTelegramPollingStateParams{NextUpdateID: update.ID + 1, LastUpdateAt: timestamptzValue(time.Now().UTC())}); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit telegram update: %w", err)
	}
	return nil
}
func validPrivateIdentity(chatType string, isBot bool, userID, chatID int64) bool {
	return chatType == "private" && !isBot && userID > 0 && userID == chatID
}
func applyBotMessageTx(ctx context.Context, tx pgx.Tx, id int64, message *utiltelegram.Message) error {
	if !validPrivateIdentity(message.ChatType, message.IsBot, message.UserID, message.ChatID) {
		return nil
	}
	fields := strings.Fields(message.Text)
	if len(fields) != 2 || strings.ToLower(fields[0]) != "/start" {
		return nil
	}
	token := fields[1]
	body := telegramBindingInvalidText
	var binding *TelegramBinding
	decoded, decodeErr := base64.RawURLEncoding.DecodeString(token)
	if decodeErr == nil && len(decoded) == 32 {
		digest := sha256.Sum256([]byte(token))
		var err error
		binding, err = completeTelegramBindingAttemptTx(ctx, tx, completeTelegramBindingAttemptRequest{
			TokenDigest: digest[:], TelegramUserID: message.UserID, TelegramChatID: message.ChatID,
			TelegramUsername: strings.TrimSpace(message.Username), TelegramDisplayName: strings.TrimSpace(strings.TrimSpace(message.FirstName) + " " + strings.TrimSpace(message.LastName)),
		})
		switch {
		case err == nil:
			body = telegramBindingConfirmationText
		case errors.Is(err, ErrTelegramBindingAttemptInvalid), errors.Is(err, ErrTelegramBindingAttemptExpired):
		case errors.Is(err, ErrTelegramIdentityInUse):
			body = telegramBindingIdentityUsedText
		default:
			return err
		}
	}
	var owner pgtype.UUID
	var revision pgtype.Int8
	if binding != nil {
		owner, _ = uuidValue(binding.AccountID)
		revision = pgtype.Int8{Int64: binding.Revision, Valid: true}
	}
	payload, err := json.Marshal([]any{message.ChatID, body})
	if err != nil {
		return err
	}
	digest := sha256.Sum256(payload)
	return q.New(tx).CreateTelegramBindingReply(ctx, q.CreateTelegramBindingReplyParams{UpdateID: id, AccountID: owner, BindingRevision: revision, TelegramChatID: message.ChatID, Body: body, PayloadDigest: digest[:]})
}

type ClaimedTelegramBindingReply struct {
	ID              int64
	AccountID       string
	BindingRevision int64
	TelegramChatID  int64
	Body            string
}

func (s *SQLStore) ClaimPendingTelegramBindingReplies(ctx context.Context, opts ClaimDeliveriesOptions) ([]ClaimedTelegramBindingReply, error) {
	if err := s.configured(); err != nil {
		return nil, err
	}
	limit, timeout := normalizeClaimOptions(opts)
	rows, err := s.queries.ClaimPendingTelegramBindingReplies(ctx, q.ClaimPendingTelegramBindingRepliesParams{Limit: int32(limit), LockedBy: textValue(opts.LockedBy), LockTimeout: intervalValue(timeout)})
	if err != nil {
		return nil, err
	}
	items := make([]ClaimedTelegramBindingReply, 0, len(rows))
	for _, r := range rows {
		items = append(items, ClaimedTelegramBindingReply{ID: r.ID, AccountID: uuidString(r.AccountID), BindingRevision: r.BindingRevision.Int64, TelegramChatID: r.TelegramChatID, Body: r.Body})
	}
	return items, nil
}
