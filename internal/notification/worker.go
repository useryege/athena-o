package notification

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/notification/delivery"
	notificationstore "github.com/useryege/athena/internal/notification/store"
)

const (
	defaultWorkerSendInterval = 1100 * time.Millisecond
	defaultWorkerPollInterval = 500 * time.Millisecond
	defaultWorkerBatchSize    = 10
	defaultWorkerLockTimeout  = 2 * time.Minute
	defaultWorkerID           = "notification-worker"
)

type WorkerConfig struct {
	SendInterval time.Duration
	PollInterval time.Duration
	BatchSize    int
	LockTimeout  time.Duration
	WorkerID     string
	Disabled     bool
}

func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		SendInterval: defaultWorkerSendInterval, PollInterval: defaultWorkerPollInterval,
		BatchSize:   defaultWorkerBatchSize,
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
	s.senderIncarnation = uuid.New()
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

func (s *Service) processClaimedSystemNotification(ctx context.Context, item notificationstore.ClaimedSystemNotificationDelivery) {
	message := renderNotificationMessage(sendNotificationParams{source: item.Source, severity: item.Severity, title: item.Title, body: item.Body, link: item.Link, telegramChat: item.TelegramChat, topicLabel: item.TopicLabel})
	s.sendPermittedNotification(ctx, delivery.WorkRef{Kind: "system", ID: item.ID}, SendRequest{SystemTelegramChat: item.TelegramChat, MessageThreadID: item.MessageThreadID, Text: message.Text})
}

func (s *Service) processClaimedAccountNotification(ctx context.Context, item notificationstore.ClaimedAccountNotificationDelivery) {
	message := renderNotificationMessage(sendNotificationParams{source: item.Source, severity: item.Severity, title: item.Title, body: item.Body, link: item.Link})
	outcome := s.sendPermittedNotification(ctx, delivery.WorkRef{Kind: "account", ID: item.ID}, SendRequest{TelegramChatID: item.TelegramChatID, Text: message.Text})
	if outcome.Code == "recipient_unreachable" {
		if err := s.store.MarkTelegramBindingUnreachable(ctx, item.AccountID, item.TelegramChatID, item.BindingRevision, outcome.Code); err != nil {
			log.WithError(err).Warn("failed to mark telegram binding unreachable")
		}
	}
}

func (s *Service) sendPermittedNotification(ctx context.Context, ref delivery.WorkRef, request SendRequest) delivery.Outcome {
	permit, err := s.store.Authorize(ctx, ref, s.senderIncarnation)
	if err != nil {
		log.WithError(err).WithField("notification_id", ref.ID).Debug("notification send was not authorized")
		return delivery.Outcome{Kind: "failed", Code: "not_authorized"}
	}
	started := make(chan time.Time, 1)
	result := make(chan delivery.Outcome, 1)
	sendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	go func() {
		result <- s.sender.Send(sendCtx, request, func(at time.Time) {
			select {
			case started <- at:
			default:
			}
		})
	}()
	var outcome delivery.Outcome
	var startedAt time.Time
	for {
		select {
		case at := <-started:
			startedAt = at
			recordCtx, recordCancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
			err = s.store.RecordStarted(recordCtx, permit, at)
			recordCancel()
			if err != nil {
				log.WithError(err).WithField("attempt_id", permit.AttemptID).Warn("failed to record notification HTTP start")
			}
		case outcome = <-result:
			// Both channels may be ready; preserve the start even when the result wins the select.
			if startedAt.IsZero() {
				select {
				case startedAt = <-started:
				default:
				}
			}
			goto received
		}
	}
received:
	resultAt := time.Now().UTC()
	recordCtx, recordCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer recordCancel()
	if !startedAt.IsZero() {
		if err = s.store.RecordStarted(recordCtx, permit, startedAt); err != nil {
			log.WithError(err).Warn("failed to persist notification start evidence")
		}
	}
	for {
		err = s.store.RecordOutcome(recordCtx, permit, outcome, resultAt)
		if err == nil {
			return outcome
		}
		if err == notificationstore.ErrStalePermit || !sleepWorker(recordCtx, 100*time.Millisecond) {
			break
		}
	}
	log.WithError(err).WithField("attempt_id", permit.AttemptID).Warn("notification result remains unconfirmed; send will not be repeated")
	return outcome
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
	return fmt.Sprintf("send_interval=%s poll_interval=%s batch_size=%d lock_timeout=%s worker_id=%s disabled=%t",
		c.SendInterval, c.PollInterval, c.BatchSize, c.LockTimeout, c.WorkerID, c.Disabled)
}
