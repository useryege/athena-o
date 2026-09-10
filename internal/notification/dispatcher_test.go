package notification

import (
	"context"
	"fmt"
	"github.com/useryege/athena/internal/notification/delivery"
	"sync"
	"testing"
	"time"
)

type manualDispatchClock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []chan time.Time
}

func (c *manualDispatchClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *manualDispatchClock) After(time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	c.waiters = append(c.waiters, ch)
	return ch
}
func (c *manualDispatchClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	for _, ch := range c.waiters {
		ch <- c.now
	}
	c.waiters = nil
}

type blockedWorkSource struct {
	mu      sync.Mutex
	items   []delivery.Candidate
	started chan delivery.Candidate
	done    chan struct{}
	clock   Clock
}

func (s *blockedWorkSource) Ready(context.Context, time.Time) ([]delivery.Candidate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]delivery.Candidate(nil), s.items...), nil
}
func (s *blockedWorkSource) Dispatch(ctx context.Context, c delivery.Candidate, started func(time.Time)) error {
	s.mu.Lock()
	for i, v := range s.items {
		if v.Ref == c.Ref {
			s.items = append(s.items[:i], s.items[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
	started(s.clock.Now())
	s.started <- c
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.done:
		return nil
	}
}
func awaitCandidate(t *testing.T, ch <-chan delivery.Candidate) delivery.Candidate {
	t.Helper()
	select {
	case c := <-ch:
		return c
	case <-time.After(3 * time.Second):
		t.Fatal("dispatcher did not progress")
		return delivery.Candidate{}
	}
}
func TestDispatcherStartsDifferentOwnersConcurrentlyAndBoundsWorkers(t *testing.T) {
	c := &manualDispatchClock{now: time.Unix(100, 0)}
	s := &blockedWorkSource{started: make(chan delivery.Candidate, 30), done: make(chan struct{}), clock: c}
	for i := 0; i < 12; i++ {
		s.items = append(s.items, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: int64(i + 1)}, OwnerID: "busy", ChatID: int64(i + 1)})
	}
	for i := 0; i < 10; i++ {
		s.items = append(s.items, delivery.Candidate{Ref: delivery.WorkRef{Kind: "reply", ID: int64(i + 1)}, OwnerID: fmt.Sprint(i), ChatID: int64(i + 100)})
	}
	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	go func() {
		finished <- NewDispatcher(c, []WorkSource{s}, NewBudget(20, time.Second, 20, time.Minute), 12).Run(ctx)
	}()
	owners := map[string]bool{}
	for i := 0; i < 12; i++ {
		owners[awaitCandidate(t, s.started).OwnerID] = true
	}
	if len(owners) != 11 {
		t.Fatalf("unfair owners: %v", owners)
	}
	select {
	case <-s.started:
		t.Fatal("exceeded concurrency")
	default:
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown leaked work")
	}
}
func TestDispatcherSerializesSameChatUntilCompletionAndBudget(t *testing.T) {
	c := &manualDispatchClock{now: time.Unix(100, 0)}
	s := &blockedWorkSource{started: make(chan delivery.Candidate, 4), done: make(chan struct{}, 4), clock: c}
	for i := int64(1); i <= 2; i++ {
		s.items = append(s.items, delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: i}, ChatID: 1, OwnerID: "one"})
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	finished := make(chan error, 1)
	go func() {
		finished <- NewDispatcher(c, []WorkSource{s}, NewBudget(20, time.Second, 20, time.Minute), 12).Run(ctx)
	}()
	awaitCandidate(t, s.started)
	c.advance(2 * time.Second)
	select {
	case <-s.started:
		t.Fatal("same chat overlapped")
	default:
	}
	s.done <- struct{}{}
	awaitCandidate(t, s.started)
	cancel()
	<-finished
}

func TestDispatcherKeepsFutureDeadlineChatAndWorkerSlot(t *testing.T) {
	clock := &manualDispatchClock{now: time.Unix(100, 0)}
	deadline := clock.Now().Add(4 * time.Second)
	s := &blockedWorkSource{started: make(chan delivery.Candidate, 10), done: make(chan struct{}, 10), clock: clock}
	s.items = []delivery.Candidate{
		{Ref: delivery.WorkRef{Kind: "account", ID: 1}, OwnerID: "a", ChatID: 1},
		{Ref: delivery.WorkRef{Kind: "summary", ID: 2}, OwnerID: "a", ChatID: 1, NotBefore: deadline, Deadline: &deadline},
		{Ref: delivery.WorkRef{Kind: "system", ID: 3}, OwnerID: "b", ChatID: 2},
		{Ref: delivery.WorkRef{Kind: "system", ID: 4}, OwnerID: "c", ChatID: 3},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- NewDispatcher(clock, []WorkSource{s}, NewBudget(20, time.Second, 20, time.Minute), 2).Run(ctx)
	}()
	defer func() { cancel(); <-done }()
	first := awaitCandidate(t, s.started)
	if first.ChatID == 1 {
		t.Fatal("occupied future deadline chat")
	}
	clock.advance(4 * time.Second)
	next := awaitCandidate(t, s.started)
	if next.Ref.Kind != "summary" {
		t.Fatalf("future deadline worker was occupied: %+v", next)
	}
}
