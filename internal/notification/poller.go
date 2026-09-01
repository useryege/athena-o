package notification

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-telegram/bot/models"
	log "github.com/sirupsen/logrus"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

const (
	defaultTelegramPollTimeout      = 30 * time.Second
	defaultTelegramPollRetryDelay   = time.Second
	maximumTelegramPollRetryDelay   = 30 * time.Second
	telegramBindingConfirmationText = "ATHENA Telegram notifications are connected. You can return to ATHENA."
)

type TelegramPollerStatus struct {
	Active       bool
	LastPollAt   time.Time
	LastUpdateAt time.Time
}

type TelegramPoller struct {
	store  *notificationstore.SQLStore
	client utiltelegram.Client

	mu           sync.Mutex
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	started      bool
	active       bool
	lastPollAt   time.Time
	lastUpdateAt time.Time
}

func NewTelegramPoller(store *notificationstore.SQLStore, client utiltelegram.Client) *TelegramPoller {
	return &TelegramPoller{store: store, client: client}
}

func (p *TelegramPoller) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.started {
		return nil
	}
	if p.store == nil {
		return errors.New("telegram poller store is required")
	}
	if p.client == nil {
		return errors.New("telegram poller client is required")
	}
	webhook, err := p.client.GetWebhookInfo(ctx)
	if err != nil {
		return err
	}
	if webhook != nil && strings.TrimSpace(webhook.URL) != "" {
		return errors.New("telegram webhook is configured; long polling requires no webhook")
	}
	state, err := p.store.GetTelegramPollingState(ctx)
	if err != nil {
		return err
	}
	p.lastPollAt = state.LastPollAt
	p.lastUpdateAt = state.LastUpdateAt
	pollerCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.started = true
	p.wg.Add(1)
	go func(nextUpdateID int64) {
		defer p.wg.Done()
		p.run(pollerCtx, nextUpdateID)
	}(state.NextUpdateID)
	return nil
}

func (p *TelegramPoller) Stop() {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return
	}
	cancel := p.cancel
	p.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	p.wg.Wait()
	p.mu.Lock()
	p.cancel = nil
	p.started = false
	p.active = false
	p.mu.Unlock()
}

func (p *TelegramPoller) Status() TelegramPollerStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	return TelegramPollerStatus{
		Active: p.active, LastPollAt: p.lastPollAt, LastUpdateAt: p.lastUpdateAt,
	}
}

func (p *TelegramPoller) run(ctx context.Context, nextUpdateID int64) {
	retryDelay := defaultTelegramPollRetryDelay
	for ctx.Err() == nil {
		p.setActive(true)
		updates, err := p.client.PollUpdates(ctx, utiltelegram.PollUpdatesRequest{
			Offset: nextUpdateID, Timeout: defaultTelegramPollTimeout,
		})
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			p.setActive(false)
			log.WithError(err).Warn("telegram update poll failed")
			if !sleepWorker(ctx, retryDelay) {
				break
			}
			retryDelay = min(retryDelay*2, maximumTelegramPollRetryDelay)
			continue
		}
		polledAt := time.Now().UTC()
		state, err := p.store.RecordTelegramPoll(ctx, polledAt)
		if err != nil {
			p.setActive(false)
			log.WithError(err).Warn("failed to persist telegram poll freshness")
			if !sleepWorker(ctx, retryDelay) {
				break
			}
			retryDelay = min(retryDelay*2, maximumTelegramPollRetryDelay)
			continue
		}
		p.recordPollingState(state)
		retryDelay = defaultTelegramPollRetryDelay

		for _, update := range updates {
			if update.ID < nextUpdateID {
				continue
			}
			for {
				if ctx.Err() != nil {
					return
				}
				if err := p.handleUpdate(ctx, update); err != nil {
					p.setActive(false)
					log.WithError(err).Warn("failed to process telegram update")
					if !sleepWorker(ctx, retryDelay) {
						return
					}
					retryDelay = min(retryDelay*2, maximumTelegramPollRetryDelay)
					continue
				}
				break
			}
			for {
				if ctx.Err() != nil {
					return
				}
				processedAt := time.Now().UTC()
				state, err := p.store.AdvanceTelegramPollingState(ctx, update.ID+1, processedAt)
				if err != nil {
					p.setActive(false)
					log.WithError(err).Warn("failed to persist telegram update offset")
					if !sleepWorker(ctx, retryDelay) {
						return
					}
					retryDelay = min(retryDelay*2, maximumTelegramPollRetryDelay)
					continue
				}
				p.recordPollingState(state)
				nextUpdateID = update.ID + 1
				retryDelay = defaultTelegramPollRetryDelay
				p.setActive(true)
				break
			}
		}
	}
	p.setActive(false)
}

func (p *TelegramPoller) handleUpdate(ctx context.Context, update utiltelegram.Update) error {
	if update.Message != nil {
		return p.handleMessage(ctx, update.Message)
	}
	if update.MyChatMember != nil {
		return p.handleMyChatMember(ctx, update.MyChatMember)
	}
	return nil
}

func (p *TelegramPoller) handleMessage(ctx context.Context, message *utiltelegram.Message) error {
	if message == nil || message.ChatType != string(models.ChatTypePrivate) || message.IsBot ||
		message.UserID <= 0 || message.ChatID <= 0 || message.UserID != message.ChatID {
		return nil
	}
	token, ok := telegramStartToken(message.Text)
	if !ok {
		return nil
	}
	if !validTelegramBindingToken(token) {
		p.sendBindingReply(ctx, message.ChatID, "This ATHENA Telegram binding link is invalid or expired. Create a new attempt in ATHENA.")
		return nil
	}
	displayName := strings.TrimSpace(strings.TrimSpace(message.FirstName) + " " + strings.TrimSpace(message.LastName))
	_, err := p.store.CompleteTelegramBindingAttempt(ctx, notificationstore.CompleteTelegramBindingAttemptRequest{
		TokenDigest: telegramBindingTokenDigest(token), TelegramUserID: message.UserID,
		TelegramChatID: message.ChatID, TelegramUsername: strings.TrimSpace(message.Username),
		TelegramDisplayName: displayName,
	})
	switch {
	case err == nil:
		p.sendBindingReply(ctx, message.ChatID, telegramBindingConfirmationText)
		return nil
	case errors.Is(err, notificationstore.ErrTelegramBindingAttemptInvalid),
		errors.Is(err, notificationstore.ErrTelegramBindingAttemptExpired):
		p.sendBindingReply(ctx, message.ChatID, "This ATHENA Telegram binding link is invalid or expired. Create a new attempt in ATHENA.")
		return nil
	case errors.Is(err, notificationstore.ErrTelegramIdentityInUse):
		p.sendBindingReply(ctx, message.ChatID, "This Telegram identity is already connected and cannot be used.")
		return nil
	default:
		return err
	}
}

func (p *TelegramPoller) handleMyChatMember(ctx context.Context, member *utiltelegram.MyChatMemberUpdate) error {
	if member == nil || member.ChatType != string(models.ChatTypePrivate) || member.IsBot ||
		member.UserID <= 0 || member.ChatID <= 0 || member.UserID != member.ChatID {
		return nil
	}
	switch member.NewStatus {
	case string(models.ChatMemberTypeLeft), string(models.ChatMemberTypeBanned):
		return p.store.MarkTelegramBindingStatusByIdentity(
			ctx, member.UserID, member.ChatID, notificationstore.TelegramBindingStatusUnreachable, "telegram bot blocked",
		)
	case string(models.ChatMemberTypeMember):
		return p.store.MarkTelegramBindingStatusByIdentity(
			ctx, member.UserID, member.ChatID, notificationstore.TelegramBindingStatusConnected, "",
		)
	default:
		return nil
	}
}

func (p *TelegramPoller) sendBindingReply(ctx context.Context, chatID int64, text string) {
	_, err := p.client.SendMessage(ctx, utiltelegram.SendMessageRequest{
		ChatID: fmt.Sprintf("%d", chatID), Text: text,
	})
	if err != nil {
		log.WithError(err).Warn("failed to send telegram binding reply")
	}
}

func (p *TelegramPoller) setActive(active bool) {
	p.mu.Lock()
	p.active = active
	p.mu.Unlock()
}

func (p *TelegramPoller) recordPollingState(state notificationstore.TelegramPollingState) {
	p.mu.Lock()
	p.lastPollAt = state.LastPollAt
	p.lastUpdateAt = state.LastUpdateAt
	p.mu.Unlock()
}

func telegramStartToken(text string) (string, bool) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) != 2 {
		return "", false
	}
	if strings.ToLower(fields[0]) != "/start" {
		return "", false
	}
	return fields[1], true
}

func validTelegramBindingToken(value string) bool {
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(value))
	return err == nil && len(decoded) == 32
}
