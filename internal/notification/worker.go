package notification

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	notificationstore "github.com/useryege/athena/internal/notification/store"
)

const (
	defaultWorkerSendInterval = 1100 * time.Millisecond
	defaultWorkerPollInterval = 500 * time.Millisecond
	defaultWorkerBatchSize    = 10
	defaultWorkerMaxAttempts  = 5
	defaultWorkerLockTimeout  = 2 * time.Minute
	defaultWorkerID           = "notification-worker"
)

type WorkerConfig struct {
	SendInterval time.Duration
	PollInterval time.Duration
	BatchSize    int
	MaxAttempts  int
	LockTimeout  time.Duration
	WorkerID     string
	Disabled     bool
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		SendInterval: defaultWorkerSendInterval, PollInterval: defaultWorkerPollInterval,
		BatchSize: defaultWorkerBatchSize, MaxAttempts: defaultWorkerMaxAttempts,
		LockTimeout: defaultWorkerLockTimeout, WorkerID: defaultWorkerID,
	}
}

func normalizeWorkerConfig(config WorkerConfig) WorkerConfig {
	defaults := DefaultWorkerConfig()
	if config.SendInterval <= 0 {
		config.SendInterval = defaults.SendInterval
	}
	if config.PollInterval <= 0 {
		config.PollInterval = defaults.PollInterval
	}
	if config.BatchSize <= 0 {
		config.BatchSize = defaults.BatchSize
	}
	if config.MaxAttempts <= 0 {
		config.MaxAttempts = defaults.MaxAttempts
	}
	if config.LockTimeout <= 0 {
		config.LockTimeout = defaults.LockTimeout
	}
	config.WorkerID = strings.TrimSpace(config.WorkerID)
	if config.WorkerID == "" {
		config.WorkerID = defaults.WorkerID
		if hostname, err := os.Hostname(); err == nil && strings.TrimSpace(hostname) != "" {
			config.WorkerID += "-" + strings.TrimSpace(hostname)
		}
	}
	return config
}

func (s *Service) startWorkerLocked(ctx context.Context) {
	if s.workerConfig.Disabled {
		return
	}
	workerCtx, cancel := context.WithCancel(ctx)
	s.workerCancel = cancel
	s.workerWG.Add(1)
	go func() {
		defer s.workerWG.Done()
		s.runWorker(workerCtx)
	}()
}

func (s *Service) stopWorkerLocked() {
	if s.workerCancel == nil {
		return
	}
	s.workerCancel()
	s.workerWG.Wait()
	s.workerCancel = nil
}

type claimedNotification struct {
	system  *notificationstore.ClaimedSystemNotificationDelivery
	account *notificationstore.ClaimedAccountNotificationDelivery
}

func (s *Service) runWorker(ctx context.Context) {
	preferAccount := false
	for ctx.Err() == nil {
		claimed := s.claimFairNotificationBatch(ctx, preferAccount)
		preferAccount = !preferAccount
		if len(claimed) == 0 {
			if !sleepWorker(ctx, s.workerConfig.PollInterval) {
				return
			}
			continue
		}
		for _, delivery := range claimed {
			if ctx.Err() != nil {
				return
			}
			if delivery.system != nil {
				s.processClaimedSystemNotification(ctx, *delivery.system)
			} else if delivery.account != nil {
				s.processClaimedAccountNotification(ctx, *delivery.account)
			}
			if !sleepWorker(ctx, s.workerConfig.SendInterval) {
				return
			}
		}
	}
}

func (s *Service) claimFairNotificationBatch(ctx context.Context, preferAccount bool) []claimedNotification {
	batchSize := s.workerConfig.BatchSize
	accountLimit := batchSize / 2
	systemLimit := batchSize - accountLimit
	if preferAccount {
		accountLimit, systemLimit = systemLimit, accountLimit
	}
	claimOptions := func(limit int) notificationstore.ClaimDeliveriesOptions {
		return notificationstore.ClaimDeliveriesOptions{
			Limit: limit, LockedBy: s.workerConfig.WorkerID, LockTimeout: s.workerConfig.LockTimeout,
		}
	}
	var systemItems []notificationstore.ClaimedSystemNotificationDelivery
	if systemLimit > 0 {
		items, err := s.store.ClaimPendingSystemNotificationDeliveries(ctx, claimOptions(systemLimit))
		if err != nil {
			log.WithError(err).Warn("failed to claim system notification deliveries")
		} else {
			systemItems = items
		}
	}
	var accountItems []notificationstore.ClaimedAccountNotificationDelivery
	if accountLimit > 0 {
		items, err := s.store.ClaimPendingAccountNotificationDeliveries(ctx, claimOptions(accountLimit))
		if err != nil {
			log.WithError(err).Warn("failed to claim account notification deliveries")
		} else {
			accountItems = items
		}
	}
	return interleaveClaimedNotifications(systemItems, accountItems, preferAccount)
}

func interleaveClaimedNotifications(
	systemItems []notificationstore.ClaimedSystemNotificationDelivery,
	accountItems []notificationstore.ClaimedAccountNotificationDelivery,
	preferAccount bool,
) []claimedNotification {
	items := make([]claimedNotification, 0, len(systemItems)+len(accountItems))
	for systemIndex, accountIndex := 0, 0; systemIndex < len(systemItems) || accountIndex < len(accountItems); {
		if preferAccount && accountIndex < len(accountItems) {
			item := accountItems[accountIndex]
			items = append(items, claimedNotification{account: &item})
			accountIndex++
		}
		if systemIndex < len(systemItems) {
			item := systemItems[systemIndex]
			items = append(items, claimedNotification{system: &item})
			systemIndex++
		}
		if !preferAccount && accountIndex < len(accountItems) {
			item := accountItems[accountIndex]
			items = append(items, claimedNotification{account: &item})
			accountIndex++
		}
	}
	return items
}

func (s *Service) processClaimedSystemNotification(ctx context.Context, delivery notificationstore.ClaimedSystemNotificationDelivery) {
	message := renderNotificationMessage(sendNotificationParams{
		source: delivery.Source, severity: delivery.Severity, title: delivery.Title,
		body: delivery.Body, link: delivery.Link, telegramChat: delivery.TelegramChat,
		topicLabel: delivery.TopicLabel,
	})
	providerMessageID, err := s.sender.Send(ctx, SendRequest{
		SystemTelegramChat: delivery.TelegramChat, MessageThreadID: delivery.MessageThreadID,
		Text: message.Text,
	})
	if err == nil {
		if markErr := s.store.MarkSystemNotificationDeliverySent(ctx, delivery.ID, providerMessageID); markErr != nil {
			log.WithError(markErr).WithField("notification_id", delivery.ID).Warn("failed to mark system notification delivery sent")
		}
		return
	}
	if delivery.Attempts >= s.workerConfig.MaxAttempts {
		if markErr := s.store.MarkSystemNotificationDeliveryFailed(ctx, delivery.ID, err.Error()); markErr != nil {
			log.WithError(markErr).WithField("notification_id", delivery.ID).Warn("failed to mark system notification delivery failed")
		}
		return
	}
	nextAttemptAt := time.Now().UTC().Add(retryDelay(delivery.Attempts, err))
	if retryErr := s.store.ScheduleSystemNotificationDeliveryRetry(ctx, delivery.ID, err.Error(), nextAttemptAt); retryErr != nil {
		log.WithError(retryErr).WithField("notification_id", delivery.ID).Warn("failed to schedule system notification retry")
	}
}

func (s *Service) processClaimedAccountNotification(ctx context.Context, delivery notificationstore.ClaimedAccountNotificationDelivery) {
	message := renderNotificationMessage(sendNotificationParams{
		source: delivery.Source, severity: delivery.Severity, title: delivery.Title,
		body: delivery.Body, link: delivery.Link,
	})
	_, err := s.store.SendAccountNotificationWithBindingLock(
		ctx, delivery.AccountID, delivery.TelegramChatID, delivery.BindingRevision, delivery.ID,
		func(sendCtx context.Context) (string, error) {
			return s.sender.Send(sendCtx, SendRequest{
				TelegramChatID: delivery.TelegramChatID, Text: message.Text,
			})
		},
	)
	if errors.Is(err, notificationstore.ErrAccountNotificationBindingChanged) {
		return
	}
	if err == nil {
		return
	}
	if IsTelegramRecipientUnreachable(err) {
		if bindingErr := s.store.MarkTelegramBindingUnreachable(
			ctx, delivery.AccountID, delivery.TelegramChatID, delivery.BindingRevision, delivery.ID, err.Error(),
		); bindingErr != nil {
			log.WithError(bindingErr).WithField("notification_id", delivery.ID).Warn("failed to mark telegram binding unreachable")
		}
		return
	}
	if delivery.Attempts >= s.workerConfig.MaxAttempts {
		if markErr := s.store.MarkAccountNotificationDeliveryFailed(ctx, delivery, err.Error()); markErr != nil {
			log.WithError(markErr).WithField("notification_id", delivery.ID).Warn("failed to mark account notification delivery failed")
		}
		return
	}
	nextAttemptAt := time.Now().UTC().Add(retryDelay(delivery.Attempts, err))
	if retryErr := s.store.ScheduleAccountNotificationDeliveryRetry(ctx, delivery, err.Error(), nextAttemptAt); retryErr != nil {
		log.WithError(retryErr).WithField("notification_id", delivery.ID).Warn("failed to schedule account notification retry")
	}
}

func retryDelay(attempts int, err error) time.Duration {
	if retryAfter, ok := RetryAfterFromError(err); ok {
		return retryAfter
	}
	if attempts < 1 {
		attempts = 1
	}
	delay := time.Duration(1<<min(attempts, 5)) * time.Second
	if delay > 32*time.Second {
		return 32 * time.Second
	}
	return delay
}

func sleepWorker(ctx context.Context, duration time.Duration) bool {
	if duration <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func (c WorkerConfig) String() string {
	return fmt.Sprintf("send_interval=%s poll_interval=%s batch_size=%d max_attempts=%d lock_timeout=%s worker_id=%s disabled=%t",
		c.SendInterval, c.PollInterval, c.BatchSize, c.MaxAttempts, c.LockTimeout, c.WorkerID, c.Disabled)
}
