package notification

import (
	"context"
	"errors"
	"github.com/useryege/athena/internal/notification/delivery"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestBudgetKeepsDifferentChatsIndependent(t *testing.T) {
	b := NewBudget(20, time.Second, 20, time.Minute)
	at := time.Unix(100, 0)
	a := delivery.Candidate{ChatID: 1}
	b.Start(a, at)
	if got := b.Next(a, at); !got.Equal(at.Add(time.Second)) {
		t.Fatal(got)
	}
	if got := b.Next(delivery.Candidate{ChatID: 2}, at); !got.Equal(at) {
		t.Fatal(got)
	}
}
func TestBudgetReservationsCannotOversellOrDoubleCount(t *testing.T) {
	b := NewBudget(2, time.Second, 20, time.Minute)
	at := time.Unix(100, 0)
	a := delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: 1}, ChatID: 1}
	z := delivery.Candidate{Ref: delivery.WorkRef{Kind: "system", ID: 2}, ChatID: 2}
	first, ok := b.Reserve(a, at)
	if !ok {
		t.Fatal("first reservation rejected")
	}
	if _, ok = b.Reserve(a, at); ok {
		t.Fatal("duplicate reserved")
	}
	second, ok := b.Reserve(z, at)
	if !ok {
		t.Fatal("second rejected")
	}
	if _, ok = b.Reserve(delivery.Candidate{ChatID: 3}, at); ok {
		t.Fatal("oversold bot")
	}
	b.Release(second)
	b.Start(a, at)
	b.Release(first)
	if _, ok = b.Reserve(z, at); !ok {
		t.Fatal("start double counted reservation")
	}
}
func TestBudgetGroupSlidingWindowAndRateLimit(t *testing.T) {
	b := NewBudget(20, time.Second, 20, time.Minute)
	at := time.Unix(100, 0)
	c := delivery.Candidate{ChatID: -1, Group: true}
	for i := 0; i < 20; i++ {
		b.Start(c, at.Add(time.Duration(i)*time.Second))
	}
	if got := b.Next(c, at.Add(20*time.Second)); !got.Equal(at.Add(time.Minute)) {
		t.Fatal(got)
	}
	b.Tighten(at.Add(90 * time.Second))
	if got := b.Next(delivery.Candidate{ChatID: 4}, at.Add(30*time.Second)); !got.Equal(at.Add(90 * time.Second)) {
		t.Fatal(got)
	}
}

func TestBudgetSharesAllSourceKindsAndRetainsActualWindowAfterRelease(t *testing.T) {
	b := NewBudget(20, time.Second, 20, time.Minute)
	now := time.Unix(100, 0)
	for i := int64(0); i < 20; i++ {
		c := delivery.Candidate{Ref: delivery.WorkRef{Kind: []string{"account", "system", "reply"}[i%3], ID: i + 1}, ChatID: i + 1}
		id, ok := b.Reserve(c, now)
		if !ok {
			t.Fatal("budget rejected below limit", i)
		}
		b.Start(c, now)
		b.Release(id)
	}
	c := delivery.Candidate{ChatID: 99}
	if _, ok := b.Reserve(c, now.Add(999*time.Millisecond)); ok {
		t.Fatal("source released an actual start")
	}
	if _, ok := b.Reserve(c, now.Add(time.Second)); !ok {
		t.Fatal("rolling window did not release at boundary")
	}
}

func TestBudgetKeepsAllImminentDeadlineCredits(t *testing.T) {
	b := NewBudget(20, time.Second, 20, time.Minute)
	now := time.Unix(100, 0)
	future := delivery.Candidate{NotBefore: now.Add(500 * time.Millisecond)}
	for i := int64(1); i <= 9; i++ {
		b.Start(delivery.Candidate{ChatID: i}, now)
	}
	if !b.roomForDeadline(future, 10) {
		t.Fatal("blocked ordinary send despite ten spare deadline credits")
	}
	b.Start(delivery.Candidate{ChatID: 10}, now)
	if b.roomForDeadline(future, 10) {
		t.Fatal("ten future deadline chats collapsed into one reserved credit")
	}
}

func TestBudgetTightenInvalidatesUnpermittedReservations(t *testing.T) {
	b := NewBudget(1, time.Second, 20, time.Minute)
	now := time.Unix(100, 0)
	c := delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: 1}, ChatID: 123}
	old, ok := b.Reserve(c, now)
	if !ok {
		t.Fatal("initial reservation failed")
	}
	b.Tighten(now.Add(120 * time.Second))
	if _, ok = b.Reserve(c, now); ok {
		t.Fatal("cooldown allowed new reservation")
	}
	fresh, ok := b.Reserve(c, now.Add(120*time.Second))
	if !ok {
		t.Fatal("invalidated reservation still occupies capacity after cooldown")
	}
	b.Release(old)
	if _, ok = b.Reserve(delivery.Candidate{ChatID: 456}, now.Add(120*time.Second)); ok {
		t.Fatal("release of invalidated reservation freed a newer reservation")
	}
	b.Start(c, now.Add(120*time.Second))
	b.Release(fresh)
	if got := b.Next(delivery.Candidate{ChatID: 456}, now.Add(120*time.Second)); !got.Equal(now.Add(121 * time.Second)) {
		t.Fatal("actual start lost after invalidation", got)
	}
}

func TestBudgetAuthorizationKeepsCommittedReservationThroughTighten(t *testing.T) {
	b := NewBudget(1, time.Second, 20, time.Minute)
	now := time.Unix(100, 0)
	c := delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: 1}, ChatID: 123}
	old, ok := b.Reserve(c, now)
	if !ok {
		t.Fatal("reserve failed")
	}
	b.Tighten(now.Add(time.Second))
	if finish, ok := b.beginAuthorization(old, now.Add(time.Second)); ok {
		finish(false)
		t.Fatal("invalidated reservation regained permission after cooldown")
	}
	fresh, ok := b.Reserve(c, now.Add(time.Second))
	if !ok {
		t.Fatal("fresh reservation denied")
	}
	finish, ok := b.beginAuthorization(fresh, now.Add(time.Second))
	if !ok {
		t.Fatal("fresh reservation cannot authorize")
	}
	tightened := make(chan struct{})
	go func() { b.Tighten(now.Add(2 * time.Second)); close(tightened) }()
	waitBudgetWriter(t, b)
	finish(true)
	<-tightened
	// Permission committed before tightening still owns capacity until the actual HTTP start.
	if _, ok = b.Reserve(delivery.Candidate{ChatID: 456}, now.Add(2*time.Second)); ok {
		t.Fatal("tightening removed already committed reservation")
	}
	b.Start(c, now.Add(2*time.Second))
	b.Release(fresh)
	if got := b.Next(delivery.Candidate{ChatID: 456}, now.Add(2*time.Second)); !got.Equal(now.Add(3 * time.Second)) {
		t.Fatal("actual start after committed permission lost capacity", got)
	}
}

func TestBudgetTightenWriterDoesNotBlockStartedAndAbortReleasesAuthorization(t *testing.T) {
	b := NewBudget(20, time.Second, 20, time.Minute)
	now := time.Unix(100, 0)
	c := delivery.Candidate{Ref: delivery.WorkRef{Kind: "account", ID: 1}, ChatID: 123}
	id, ok := b.Reserve(c, now)
	if !ok {
		t.Fatal("reserve failed")
	}
	finish, ok := b.beginAuthorization(id, now)
	if !ok {
		t.Fatal("authorization guard failed")
	}
	defer func() {
		if finish != nil {
			finish(false)
		}
	}()
	tightened := make(chan struct{})
	go func() { b.Tighten(now.Add(120 * time.Second)); close(tightened) }()
	waitBudgetWriter(t, b)
	started := make(chan struct{})
	go func() {
		b.Start(delivery.Candidate{Ref: delivery.WorkRef{Kind: "system", ID: 2}, ChatID: -777, Group: true}, now)
		close(started)
	}()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("HTTP started callback blocked on permission transaction/writer")
	}
	finish(false)
	finish = nil
	select {
	case <-tightened:
	case <-time.After(3 * time.Second):
		t.Fatal("aborted permission retained authorization lock")
	}
	if finishAgain, ok := b.beginAuthorization(id, now.Add(120*time.Second)); ok {
		finishAgain(false)
		t.Fatal("aborted old reservation survived tightening")
	}
}

// Observe a queued writer only to control the interleaving; assertions above exercise
// actual started accounting and cancellation cleanup rather than an arbitrary sleep.
func waitBudgetWriter(t *testing.T, b *Budget) {
	t.Helper()
	deadline := time.After(3 * time.Second)
	for {
		if !b.authorizationMu.TryRLock() {
			return
		}
		b.authorizationMu.RUnlock()
		select {
		case <-deadline:
			t.Fatal("budget writer was not queued")
		default:
			runtime.Gosched()
		}
	}
}

type observedAdmissionContext struct {
	context.Context
	observed chan struct{}
	once     sync.Once
}

func (c *observedAdmissionContext) Done() <-chan struct{} {
	c.once.Do(func() { close(c.observed) })
	return c.Context.Done()
}

func TestBudgetStartAdmissionCancellationWhileWriterAwaitsCommit(t *testing.T) {
	b := NewBudget(20, time.Second, 20, time.Minute)
	clock := &manualDispatchClock{now: time.Now()}
	id, ok := b.Reserve(delivery.Candidate{ChatID: 123}, clock.Now())
	if !ok {
		t.Fatal("reserve")
	}
	finish, ok := b.beginAuthorization(id, clock.Now())
	if !ok {
		t.Fatal("authorize")
	}
	tightened := make(chan struct{})
	defer func() { finish(false); <-tightened }()
	go func() { b.Tighten(clock.Now().Add(120 * time.Second)); close(tightened) }()
	waitBudgetWriter(t, b)
	baseCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &observedAdmissionContext{Context: baseCtx, observed: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		release, err := b.admitStart(ctx, clock)
		if release != nil {
			release()
		}
		done <- err
	}()
	select {
	case <-ctx.observed:
	case <-time.After(time.Second):
		t.Fatal("admission did not wait for queued writer")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("admission cancellation waited for another database transaction")
	}
}

func TestBudgetStartAdmissionRechecksExtendedCooldownAndOrdersActualStart(t *testing.T) {
	b := NewBudget(20, time.Second, 20, time.Minute)
	clock := &manualDispatchClock{now: time.Now()}
	b.Tighten(clock.Now().Add(120 * time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ready := make(chan func(), 1)
	go func() {
		release, err := b.admitStart(ctx, clock)
		if err != nil {
			t.Error(err)
			return
		}
		ready <- release
	}()
	awaitWait := func() {
		t.Helper()
		for {
			clock.mu.Lock()
			n := len(clock.waiters)
			clock.mu.Unlock()
			if n > 0 {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatal("admission did not wait")
			default:
				runtime.Gosched()
			}
		}
	}
	awaitWait()
	b.Tighten(clock.Now().Add(240 * time.Second))
	clock.advance(120 * time.Second)
	awaitWait()
	select {
	case release := <-ready:
		release()
		t.Fatal("extended cooldown ignored")
	default:
	}
	clock.advance(120 * time.Second)
	var release func()
	select {
	case release = <-ready:
	case <-ctx.Done():
		t.Fatal("admission did not resume")
	}
	tightened := make(chan struct{})
	go func() { b.Tighten(clock.Now().Add(120 * time.Second)); close(tightened) }()
	waitBudgetWriter(t, b)
	// The transport's started callback can account its real event while the writer waits.
	b.Start(delivery.Candidate{ChatID: 123}, clock.Now())
	release()
	select {
	case <-tightened:
	case <-ctx.Done():
		t.Fatal("actual start retained admission lock")
	}
}
