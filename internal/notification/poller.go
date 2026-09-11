package notification

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

const (
	defaultTelegramPollTimeout    = 30 * time.Second
	defaultTelegramPollRetryDelay = time.Second
	maximumTelegramPollRetryDelay = 30 * time.Second
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
				if err := p.store.ApplyBotUpdate(ctx, update); err != nil {
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
			nextUpdateID = update.ID + 1
			retryDelay = defaultTelegramPollRetryDelay
			p.setActive(true)
			p.mu.Lock()
			p.lastUpdateAt = time.Now().UTC()
			p.mu.Unlock()
		}
	}
	p.setActive(false)
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
