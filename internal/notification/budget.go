package notification

import (
	"context"
	"github.com/useryege/athena/internal/notification/delivery"
	"sort"
	"sync"
	"time"
)

type budgetEvent struct {
	chat  int64
	group bool
	at    time.Time
}
type budgetReservation struct {
	candidate   delivery.Candidate
	authorizing bool
	authorized  bool
}
type Budget struct {
	// Authorization holds a read lock only from the post-gate guard through commit.
	// HTTP admission also holds it through the constant-time started handshake only.
	// Tighten takes the write lock; actual start accounting uses mu independently.
	authorizationMu              sync.RWMutex
	mu                           sync.Mutex
	botCount, groupCount         int
	privateInterval, groupWindow time.Duration
	events                       []budgetEvent
	reservations                 map[uint64]budgetReservation
	sequence                     uint64
	blockedUntil                 time.Time
}

func NewBudget(botPerSecond int, privateInterval time.Duration, groupCount int, groupWindow time.Duration) *Budget {
	if botPerSecond < 1 || privateInterval <= 0 || groupCount < 1 || groupWindow <= 0 {
		panic("invalid notification budget")
	}
	return &Budget{botCount: botPerSecond, privateInterval: privateInterval, groupCount: groupCount, groupWindow: groupWindow, reservations: make(map[uint64]budgetReservation)}
}
func (b *Budget) Next(c delivery.Candidate, now time.Time) time.Time {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.next(c, now)
}
func (b *Budget) next(c delivery.Candidate, now time.Time) time.Time {
	next := now
	if c.NotBefore.After(next) {
		next = c.NotBefore
	}
	if b.blockedUntil.After(next) {
		next = b.blockedUntil
	}
	var bot, chat []time.Time
	for _, e := range b.events {
		if e.at.Add(time.Second).After(now) {
			bot = append(bot, e.at)
		}
		if e.chat == c.ChatID {
			if c.Group {
				if e.at.Add(b.groupWindow).After(now) {
					chat = append(chat, e.at)
				}
			} else if e.at.Add(b.privateInterval).After(next) {
				next = e.at.Add(b.privateInterval)
			}
		}
	}
	reservedBot, reservedChat := 0, 0
	for _, r := range b.reservations {
		reservedBot++
		if r.candidate.ChatID == c.ChatID {
			reservedChat++
		}
	}
	// A held reservation has no known start. Recheck on completion/start, never age it out.
	if reservedBot >= b.botCount || reservedChat > 0 {
		return maxTime(next, now.Add(time.Second))
	}
	sort.Slice(bot, func(i, j int) bool { return bot[i].Before(bot[j]) })
	if n := len(bot) + reservedBot - b.botCount; n >= 0 {
		next = maxTime(next, bot[n].Add(time.Second))
	}
	if c.Group {
		sort.Slice(chat, func(i, j int) bool { return chat[i].Before(chat[j]) })
		if n := len(chat) + reservedChat - b.groupCount; n >= 0 {
			next = maxTime(next, chat[n].Add(b.groupWindow))
		}
	}
	return next
}
func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
func (b *Budget) Reserve(c delivery.Candidate, now time.Time) (uint64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.next(c, now).After(now) {
		return 0, false
	}
	b.sequence++
	b.reservations[b.sequence] = budgetReservation{candidate: c}
	return b.sequence, true
}
func (b *Budget) Release(id uint64) { b.mu.Lock(); defer b.mu.Unlock(); delete(b.reservations, id) }
func (b *Budget) Start(c delivery.Candidate, at time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for id, r := range b.reservations {
		if r.candidate.Ref == c.Ref {
			delete(b.reservations, id)
			break
		}
	}
	b.events = append(b.events, budgetEvent{c.ChatID, c.Group, at})
	cutoff := at.Add(-max(time.Second, b.privateInterval, b.groupWindow))
	kept := b.events[:0]
	for _, e := range b.events {
		if e.at.After(cutoff) {
			kept = append(kept, e)
		}
	}
	b.events = kept
}
func (b *Budget) Tighten(until time.Time) {
	b.authorizationMu.Lock()
	defer b.authorizationMu.Unlock()
	b.mu.Lock()
	defer b.mu.Unlock()
	if !until.After(b.blockedUntil) {
		return
	}
	b.blockedUntil = until
	for id, r := range b.reservations {
		if !r.authorized {
			delete(b.reservations, id)
		}
	}
}

// beginAuthorization validates the reservation after account/row locks and orders the
// permission commit against Tighten. The caller must finish after commit/rollback,
// before invoking Sender. Start never waits for this database coordination lock.
func (b *Budget) beginAuthorization(id uint64, now time.Time) (finish func(bool), ok bool) {
	b.authorizationMu.RLock()
	b.mu.Lock()
	r, exists := b.reservations[id]
	if !exists || r.authorizing || r.authorized || b.blockedUntil.After(now) {
		b.mu.Unlock()
		b.authorizationMu.RUnlock()
		return nil, false
	}
	r.authorizing = true
	b.reservations[id] = r
	b.mu.Unlock()
	return func(committed bool) {
		b.mu.Lock()
		if r, exists := b.reservations[id]; exists {
			r.authorizing = false
			r.authorized = committed
			b.reservations[id] = r
		}
		b.mu.Unlock()
		b.authorizationMu.RUnlock()
	}, true
}

// roomForDeadline preserves bot credits for all imminent distinct deadline chats.
func (b *Budget) roomForDeadline(f delivery.Candidate, slots int) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	count := 1
	for _, e := range b.events {
		if e.at.Add(time.Second).After(f.NotBefore) {
			count++
		}
	}
	for range b.reservations {
		count++
	}
	return count+slots <= b.botCount
}

// admitStart waits without any account gate or budget lock. The returned release
// must run immediately after the transport's actual started handshake, before I/O.
// TryRLock keeps cancellation responsive even when a writer awaits another commit.
func (b *Budget) admitStart(ctx context.Context, clock Clock) (func(), error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !b.authorizationMu.TryRLock() {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Millisecond):
			}
			continue
		}
		b.mu.Lock()
		remaining := b.blockedUntil.Sub(clock.Now())
		b.mu.Unlock()
		if err := ctx.Err(); err != nil {
			b.authorizationMu.RUnlock()
			return nil, err
		}
		if remaining <= 0 {
			return b.authorizationMu.RUnlock, nil
		}
		b.authorizationMu.RUnlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-clock.After(remaining):
		}
	}
}
