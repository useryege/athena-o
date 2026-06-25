package notification

import (
	"context"
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
		SendInterval: defaultWorkerSendInterval,
		PollInterval: defaultWorkerPollInterval,
		BatchSize:    defaultWorkerBatchSize,
		MaxAttempts:  defaultWorkerMaxAttempts,
		LockTimeout:  defaultWorkerLockTimeout,
		WorkerID:     defaultWorkerID,
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

func (s *Service) runWorker(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}
		claimed, err := s.store.ClaimPendingDeliveries(ctx, notificationstore.ClaimDeliveriesOptions{
			Limit:       s.workerConfig.BatchSize,
			LockedBy:    s.workerConfig.WorkerID,
			LockTimeout: s.workerConfig.LockTimeout,
		})
		if err != nil {
			log.WithError(err).Warn("failed to claim notification deliveries")
			if !sleepWorker(ctx, s.workerConfig.PollInterval) {
				return
			}
			continue
		}
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
			s.processClaimedDelivery(ctx, delivery)
			if !sleepWorker(ctx, s.workerConfig.SendInterval) {
				return
			}
		}
	}
}

func (s *Service) processClaimedDelivery(ctx context.Context, delivery notificationstore.ClaimedDelivery) {
	message := renderNotificationMessage(sendNotificationParams{
		source:       delivery.Source,
		severity:     delivery.Severity,
		title:        delivery.Title,
		body:         delivery.Body,
		link:         delivery.Link,
		telegramChat: delivery.TelegramChat,
		topicLabel:   delivery.TopicLabel,
	})
	providerMessageID, err := s.sender.Send(ctx, SendRequest{TelegramChat: delivery.TelegramChat, MessageThreadID: delivery.MessageThreadID, Text: message.Text})
	if err == nil {
		if markErr := s.store.MarkDeliverySent(ctx, delivery.ID, providerMessageID); markErr != nil {
			log.WithError(markErr).WithField("notification_id", delivery.ID).Warn("failed to mark notification delivery sent")
		}
		return
	}

	if delivery.Attempts >= s.workerConfig.MaxAttempts {
		if markErr := s.store.MarkDeliveryFailed(ctx, delivery.ID, err.Error()); markErr != nil {
			log.WithError(markErr).WithField("notification_id", delivery.ID).Warn("failed to mark notification delivery failed")
		}
		return
	}

	nextAttemptAt := time.Now().UTC().Add(retryDelay(delivery.Attempts, err))
	if retryErr := s.store.ScheduleDeliveryRetry(ctx, delivery.ID, err.Error(), nextAttemptAt); retryErr != nil {
		log.WithError(retryErr).WithField("notification_id", delivery.ID).Warn("failed to schedule notification delivery retry")
		return
	}
	log.WithError(err).WithFields(log.Fields{
		"notification_id": delivery.ID,
		"attempts":        delivery.Attempts,
		"next_attempt_at": nextAttemptAt.Format(time.RFC3339),
	}).Warn("scheduled notification delivery retry")
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
