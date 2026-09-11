package notification

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
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
	// Publish the new entry before releasing Start's lifecycle lock. A restart
	// cannot expose its previous completed recovery while this worker is scheduled.
	s.recovery.begin(wallClock{})
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
	s.recovery.ensureStarted(wallClock{})
	budget := NewBudget(20, time.Second, 20, time.Minute)
	if err := restoreBudget(ctx, s.store, budget, time.Now()); err != nil {
		if ctx.Err() != nil {
			s.recovery.finish("cancelled", "cancelled")
		} else {
			s.recovery.finish("failed", "attempt_history_read_failed")
		}
		return err
	}
	firstStart := s.senderSession == nil || s.senderSession.FirstStart()
	retries, err := recoverStartupBudget(ctx, s.store, budget, wallClock{}, firstStart, &s.recovery)
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
	return s.executePermit(ctx, permit, onStarted, nil)
}

// executePermit is the only Sender/result path, shared by ordinary work and
// summary first parts. Its input is the committed payload, never re-rendered text.
func (s *Service) executePermit(ctx context.Context, permit delivery.Permit, onStarted func(time.Time), summary *summaryAttempt) (delivery.Outcome, error) {
	payload, err := delivery.DecodePayload(permit.Payload)
	if err != nil {
		return delivery.Outcome{}, err
	}
	request := SendRequest{TelegramChatID: permit.ChatID, MessageThreadID: payload.MessageThreadID, Text: payload.Text, Format: payload.Format}
	started := make(chan time.Time, 1)
	type senderResult struct {
		outcome delivery.Outcome
		timing  delivery.ResultTiming
	}
	result := make(chan senderResult, 1)
	// The transport starts its HTTP timeout after cancellable local budget admission.
	sendCtx, cancel := context.WithCancel(ctx)
	if summary != nil {
		sendCtx = utiltelegram.WithSendAdmission(sendCtx, summary.admit)
	} else if slot, ok := ctx.Value(dispatchSlotKey{}).(dispatchSlot); ok && slot.budget != nil {
		sendCtx = utiltelegram.WithSendAdmission(sendCtx, func(ctx context.Context) (func(), error) { return slot.budget.admitStart(ctx, slot.clock) })
	}
	defer cancel()
	go func() {
		var originalStart atomic.Pointer[time.Time]
		outcome := s.sender.Send(sendCtx, request, func(at time.Time) {
			originalStart.CompareAndSwap(nil, &at)
			if onStarted != nil {
				onStarted(at)
			}
			select {
			case started <- at:
			default:
			}
		})
		returned := time.Now()
		timing := delivery.ResultTiming{SenderReturnedAt: &returned}
		if at := originalStart.Load(); at != nil && *at != at.Round(0) {
			elapsed := returned.Sub(*at).Nanoseconds()
			if elapsed >= 0 {
				timing.SenderElapsedNS = &elapsed
			}
		}
		result <- senderResult{outcome: outcome, timing: timing}
	}()
	var outcome delivery.Outcome
	var startedAt time.Time
	for {
		select {
		case at := <-started:
			startedAt = at
			recordCtx, recordCancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
			if summary != nil {
				err = summary.recordStart(recordCtx, at)
			} else {
				err = s.store.RecordStarted(recordCtx, permit, at)
			}
			recordCancel()
			if err != nil {
				log.WithError(err).WithField("attempt_id", permit.AttemptID).Warn("failed to record notification HTTP start")
			}
		case receipt := <-result:
			outcome = receipt.outcome
			outcome.Timing = receipt.timing
			// Both channels may be ready; preserve the start even when the result wins the select.
			if startedAt.IsZero() {
				select {
				case startedAt = <-started:
					if summary != nil {
						recordCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
						_ = summary.recordStart(recordCtx, startedAt)
						cancel()
					}
				default:
				}
			}
			goto received
		}
	}
received:
	if summary != nil {
		summary.releaseGate()
	}
	resultAt := time.Now().UTC()
	if retries, ok := ctx.Value(retryBudgetKey{}).(*retryBudget); ok {
		retries.track(permit.AttemptID, outcome.RetryAfter)
	}
	recordCtx, recordCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer recordCancel()
	if !startedAt.IsZero() && summary == nil {
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
	log.WithError(err).WithField("attempt_id", permit.AttemptID).WithField("observed_outcome", outcome.Kind).WithField("sender_returned_at", outcome.Timing.SenderReturnedAt).WithField("result_persisted", false).Warn("notification result remains unconfirmed; send will not be repeated")
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
