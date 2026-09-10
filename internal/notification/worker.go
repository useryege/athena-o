package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/useryege/athena/internal/notification/delivery"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	utiltelegram "github.com/useryege/athena/util/telegram"
)

type WorkerConfig struct {
	Concurrency int
	Disabled    bool
}

func DefaultWorkerConfig() WorkerConfig { return WorkerConfig{Concurrency: 12} }
func normalizeWorkerConfig(c WorkerConfig) WorkerConfig {
	if c.Concurrency <= 0 {
		c.Concurrency = 12
	}
	return c
}
func (s *Service) startWorkerLocked(ctx context.Context) {
	if s.workerConfig.Disabled {
		return
	}
	s.workerWG.Add(2)
	go func() {
		defer s.workerWG.Done()
		if err := s.runWorker(ctx); err != nil && !errors.Is(err, context.Canceled) {
			s.failRuntime(err)
		}
	}()
	go func() {
		defer s.workerWG.Done()
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				checkCtx, cancel := context.WithTimeout(ctx, time.Second)
				err := s.senderSession.Check(checkCtx)
				cancel()
				if err != nil {
					if ctx.Err() == nil {
						s.failRuntime(err)
					}
					return
				}
			}
		}
	}()
}
func (s *Service) failRuntime(err error) {
	if s.runtimeFailed.CompareAndSwap(false, true) {
		s.workerCancel()
		if s.onFatal != nil {
			s.onFatal(err)
		}
		s.runtimeErrors <- err
	}
}

func (s *Service) runWorker(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	budget := NewBudget(20, time.Second, 20, time.Minute)
	if err := restoreBudget(ctx, s.store, budget, time.Now()); err != nil {
		return err
	}
	firstStart := s.senderSession == nil || s.senderSession.FirstStart()
	retries, err := recoverStartupBudget(ctx, s.store, budget, wallClock{}, firstStart)
	if err != nil {
		return err
	}
	ctx = context.WithValue(ctx, retryBudgetKey{}, retries)
	retryDone := make(chan error, 1)
	go func() { err := retries.run(ctx); retryDone <- err; cancel() }()
	dispatchErr := NewDispatcher(wallClock{}, s.workSources(), budget, s.workerConfig.Concurrency).Run(ctx)
	cancel()
	retryErr := <-retryDone
	if retryErr != nil && !errors.Is(retryErr, context.Canceled) {
		return retryErr
	}
	return dispatchErr
}

func (s *Service) sendPermittedNotification(ctx context.Context, candidate delivery.Candidate, onStarted func(time.Time)) (delivery.Outcome, error) {
	var finishAuthorization func(bool)
	permit, err := s.store.Authorize(ctx, candidate, s.senderIncarnation, func() error {
		if s.senderSession != nil {
			if err := s.senderSession.Check(ctx); err != nil {
				return err
			}
		}
		var err error
		finishAuthorization, err = beginDispatchAuthorization(ctx)
		return err
	})
	if finishAuthorization != nil {
		finishAuthorization(err == nil)
	}
	if err != nil {
		log.WithError(err).WithField("notification_id", candidate.Ref.ID).Debug("notification send was not authorized")
		return delivery.Outcome{}, err
	}
	payload, err := delivery.DecodePayload(permit.Payload)
	if err != nil {
		return delivery.Outcome{}, err
	}
	request := SendRequest{TelegramChatID: permit.ChatID, MessageThreadID: payload.MessageThreadID, Text: payload.Text, Format: payload.Format}
	started := make(chan time.Time, 1)
	result := make(chan delivery.Outcome, 1)
	// The transport starts its HTTP timeout after cancellable local budget admission.
	sendCtx, cancel := context.WithCancel(ctx)
	if slot, ok := ctx.Value(dispatchSlotKey{}).(dispatchSlot); ok && slot.budget != nil {
		sendCtx = utiltelegram.WithSendAdmission(sendCtx, func(ctx context.Context) (func(), error) { return slot.budget.admitStart(ctx, slot.clock) })
	}
	defer cancel()
	go func() {
		result <- s.sender.Send(sendCtx, request, func(at time.Time) {
			if onStarted != nil {
				onStarted(at)
			}
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
	if retries, ok := ctx.Value(retryBudgetKey{}).(*retryBudget); ok {
		retries.track(permit.AttemptID, outcome.RetryAfter)
	}
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
			return outcome, nil
		}
		if err == notificationstore.ErrStalePermit || !sleepWorker(recordCtx, 100*time.Millisecond) {
			break
		}
	}
	log.WithError(err).WithField("attempt_id", permit.AttemptID).Warn("notification result remains unconfirmed; send will not be repeated")
	return outcome, nil
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
	return fmt.Sprintf("concurrency=%d disabled=%t", c.Concurrency, c.Disabled)
}
