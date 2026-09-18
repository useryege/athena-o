package query

import (
	"testing"
	"time"
)

func TestCodecBindsViewerAndRejectsTamperExpiryAndParameters(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	c, err := NewCodec(key, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	viewer := Viewer{AccountID: "00000000-0000-4000-8000-000000000001", Realm: "ADMIN", CredentialKind: "LOGIN_SESSION", SessionBinding: []byte("01234567890123456789012345678901"), AccessRevision: 2}
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	f, err := NormalizeFilter(Filter{From: ptrTime(now.Add(-time.Hour)), To: ptrTime(now), PageSize: 10, ModuleCode: "wallet"}, now)
	if err != nil {
		t.Fatal(err)
	}
	token, err := c.EncodeCursor(f, viewer, CursorPosition{StartedAt: now, OperationID: "00000000-0000-4000-8000-000000000002"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(token) > 4096 {
		t.Fatal("cursor too long")
	}
	if got, err := c.DecodeCursor(token, viewer, now); err != nil || got.Filters.ModuleCode != "wallet" {
		t.Fatalf("decode: %#v %v", got, err)
	}
	other := viewer
	other.SessionBinding = []byte("11234567890123456789012345678901")
	if _, err := c.DecodeCursor(token, other, now); err != ErrCursorInvalid {
		t.Fatalf("session accepted: %v", err)
	}
	altered := token[:len(token)-1] + "A"
	if _, err := c.DecodeCursor(altered, viewer, now); err != ErrCursorInvalid {
		t.Fatalf("tamper: %v", err)
	}
	if _, err := c.DecodeCursor(token, viewer, now.Add(31*time.Minute)); err != ErrCursorExpired {
		t.Fatalf("expiry: %v", err)
	}
	changed := f
	changed.ModuleCode = "account"
	if err := c.ValidateCursorParameters(token, changed, viewer, now); err != ErrCursorInvalid {
		t.Fatalf("parameter mismatch: %v", err)
	}
	snap, err := c.EncodeSnapshot(f, viewer, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.DecodeCursor(snap, viewer, now); err != ErrCursorInvalid {
		t.Fatalf("cursor accepted snapshot: %v", err)
	}
	if _, err := c.DecodeSnapshot(snap, viewer, now); err != nil {
		t.Fatal(err)
	}
}
