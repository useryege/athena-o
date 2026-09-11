//go:build integration

package acceptance

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestCapacityDistinctTargets(t *testing.T) { runCapacity(t, 100) }
func TestCapacitySharedTargets(t *testing.T)   { runCapacity(t, 10) }
func runCapacity(t *testing.T, targets int) {
	h := newHarness(t, 10, targets)
	defer h.Close()
	s := h.Stats()
	if s.Relationships != 100 || s.UniqueTargets != targets {
		t.Fatal(s)
	}
	// Recorded ABI data are preserved; wallet/location/header/receipt state are an
	// explicitly synthetic, self-consistent chain replay, never historical proof.
	h.Advance(1100 * time.Millisecond)
	for i, wallet := range h.wallets {
		h.Push(h.Replay(wallet, []int{0, 4, 8}[i%3]))
	}
	h.wait("100 actual activities and 100 successful HTTP attempts", 30*time.Second, func() bool {
		s = h.Stats()
		if s.Activities != 100 || s.HTTPCalls != 100 {
			return false
		}
		for _, v := range s.Owners {
			if v.Activities != 10 || v.Sent != 10 {
				return false
			}
		}
		return true
	})
	if s.Sources != targets || s.Confirmed != targets || s.HistoricalRangeCalls != 0 || s.SideEffects != 0 {
		t.Fatal(s)
	}
	h.mu.Lock()
	events := append([]sendEvent(nil), h.events...)
	h.mu.Unlock()
	for _, event := range events {
		ownerIndex := int(event.Chat - 1000)
		if !strings.Contains(event.Text, "owner-"+string(rune('0'+ownerIndex))+"-target-") {
			t.Fatalf("wrong owner note/chat snapshot: %d %s", event.Chat, event.Text)
		}
	}
	raw, _ := json.Marshal(s)
	t.Logf("synthetic_replay_capacity %s", raw)
	h.captureRuntime(fmt.Sprintf("tradersync-capacity-%d-targets", targets))
	h.captureTiming(fmt.Sprintf("tradersync-capacity-%d-targets", targets))
}
