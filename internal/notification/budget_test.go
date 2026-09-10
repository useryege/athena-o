package notification

import (
	"github.com/useryege/athena/internal/notification/delivery"
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
