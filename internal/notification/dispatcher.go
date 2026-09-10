package notification

import (
	"context"
	"errors"
	"github.com/useryege/athena/internal/notification/delivery"
	"sort"
	"sync"
	"time"
)

type Clock interface {
	Now() time.Time
	After(time.Duration) <-chan time.Time
}
type wallClock struct{}

func (wallClock) Now() time.Time                         { return time.Now() }
func (wallClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

type WorkSource interface {
	Ready(context.Context, time.Time) ([]delivery.Candidate, error)
	Dispatch(context.Context, delivery.Candidate, func(time.Time)) error
}
type Dispatcher struct {
	clock       Clock
	sources     []WorkSource
	budget      *Budget
	concurrency int
	lastOwner   string
}

func NewDispatcher(clock Clock, sources []WorkSource, budget *Budget, concurrency int) *Dispatcher {
	if clock == nil || budget == nil || concurrency < 1 {
		panic("invalid dispatcher configuration")
	}
	return &Dispatcher{clock: clock, sources: sources, budget: budget, concurrency: concurrency}
}

type dispatchWork struct {
	candidate delivery.Candidate
	source    WorkSource
}
type dispatchResult struct {
	candidate delivery.Candidate
	err       error
}

func (d *Dispatcher) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	var wg sync.WaitGroup
	defer wg.Wait()
	active := map[delivery.WorkRef]bool{}
	chats := map[int64]bool{}
	completed := make(chan dispatchResult, d.concurrency)
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		now := d.clock.Now()
		var pending []dispatchWork
		for _, source := range d.sources {
			items, err := source.Ready(ctx, now)
			if err != nil {
				cancel()
				return err
			}
			for _, c := range items {
				if !active[c.Ref] {
					pending = append(pending, dispatchWork{c, source})
				}
			}
		}
		now = d.clock.Now()
		for len(active) < d.concurrency {
			futureChats := map[int64]bool{}
			botFutureChats := map[int64]bool{}
			for _, w := range pending {
				c := w.candidate
				if c.Deadline != nil && c.NotBefore.After(now) && c.NotBefore.Before(now.Add(6*time.Second)) {
					futureChats[c.ChatID] = true
					if c.NotBefore.Before(now.Add(time.Second)) {
						botFutureChats[c.ChatID] = true
					}
				}
			}
			eligible := make([]dispatchWork, 0, len(pending))
			for _, w := range pending {
				c := w.candidate
				if chats[c.ChatID] || active[c.Ref] || d.budget.Next(c, now).After(now) {
					continue
				}
				// Keep a same-chat slot free for a future deadline: one HTTP timeout plus chat spacing.
				blocked := c.Deadline == nil && len(active) >= d.concurrency-len(futureChats)

				for _, future := range pending {
					f := future.candidate
					if f.Ref == c.Ref || f.Deadline == nil || !f.NotBefore.After(now) {
						continue
					}
					if (f.ChatID == c.ChatID && f.NotBefore.Before(now.Add(6*time.Second))) || (c.Deadline == nil && f.NotBefore.Before(now.Add(time.Second)) && !d.budget.roomForDeadline(f, len(botFutureChats))) {
						blocked = true
						break
					}
				}
				if !blocked {
					eligible = append(eligible, w)
				}
			}
			if len(eligible) == 0 {
				break
			}
			sort.SliceStable(eligible, func(i, j int) bool {
				a, b := eligible[i].candidate, eligible[j].candidate
				urgentA := a.Deadline != nil && a.Deadline.Before(now.Add(6*time.Second))
				urgentB := b.Deadline != nil && b.Deadline.Before(now.Add(6*time.Second))
				if urgentA != urgentB {
					return urgentA
				}
				if urgentA && !a.Deadline.Equal(*b.Deadline) {
					return a.Deadline.Before(*b.Deadline)
				}
				afterA, afterB := a.OwnerID > d.lastOwner, b.OwnerID > d.lastOwner
				if afterA != afterB {
					return afterA
				}
				if a.OwnerID != b.OwnerID {
					return a.OwnerID < b.OwnerID
				}
				if a.NotBefore != b.NotBefore {
					return a.NotBefore.Before(b.NotBefore)
				}
				return a.Ref.ID < b.Ref.ID
			})
			w := eligible[0]
			c := w.candidate
			reservation, ok := d.budget.Reserve(c, now)
			if !ok {
				break
			}
			active[c.Ref] = true
			chats[c.ChatID] = true
			d.lastOwner = c.OwnerID
			wg.Add(1)
			go func() {
				defer wg.Done()
				defer d.budget.Release(reservation)
				var once sync.Once
				dispatchCtx := context.WithValue(ctx, dispatchSlotKey{}, dispatchSlot{clock: d.clock, until: now.Add(time.Second)})
				err := w.source.Dispatch(dispatchCtx, c, func(time.Time) { once.Do(func() { d.budget.Start(c, d.clock.Now()) }) })
				completed <- dispatchResult{c, err}
			}()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case r := <-completed:
			delete(active, r.candidate.Ref)
			delete(chats, r.candidate.ChatID)
			if r.err != nil && !errors.Is(r.err, context.Canceled) {
				var rate *RateLimitError
				if errors.As(r.err, &rate) {
					d.budget.Tighten(d.clock.Now().Add(rate.RetryAfter))
				} else if !errors.Is(r.err, ErrDispatchDeferred) {
					cancel()
					return r.err
				}
			}
		case <-d.clock.After(50 * time.Millisecond):
		}
	}
}

var ErrDispatchDeferred = errors.New("notification dispatch deferred before HTTP")

type RateLimitError struct{ RetryAfter time.Duration }

func (e *RateLimitError) Error() string { return "notification provider rate limited" }

type dispatchSlotKey struct{}
type dispatchSlot struct {
	clock Clock
	until time.Time
}

func checkDispatchSlot(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if slot, ok := ctx.Value(dispatchSlotKey{}).(dispatchSlot); ok && slot.clock.Now().After(slot.until) {
		return ErrDispatchDeferred
	}
	return nil
}
