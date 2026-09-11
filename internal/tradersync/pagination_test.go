package tradersync

import (
	"strings"
	"testing"
)

func TestActivityCursorBindsOwnerAndPageSize(t *testing.T) {
	key := []byte("test-only-cursor-key")
	c := Cursor{Version: 1, Kind: "activity_refresh", PrincipalID: "owner-a", FilterDigest: "all", Direction: "desc", PageSize: 50, SnapshotID: "9007199254740993", UpperID: "9007199254740993", LowerID: "9007199254740900"}
	token, err := EncodeCursor(c, key)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeCursor(token, key, "owner-a", "all", "activity_refresh", 50)
	if err != nil || got != c {
		t.Fatalf("round trip: %#v %v", got, err)
	}
	for _, p := range []struct {
		owner, filter, kind string
		size                int32
		key                 []byte
	}{
		{"owner-b", "all", "activity_refresh", 50, key}, {"owner-a", "other", "activity_refresh", 50, key}, {"owner-a", "all", "activity_next", 50, key}, {"owner-a", "all", "activity_refresh", 100, key}, {"owner-a", "all", "activity_refresh", 50, []byte("other")},
	} {
		if _, err := DecodeCursor(token, p.key, p.owner, p.filter, p.kind, p.size); err == nil {
			t.Errorf("accepted outside context: %#v", p)
		}
	}
	parts := strings.Split(token, ".")
	for _, bad := range []string{"", token + "x", parts[0] + "x." + parts[1], parts[0], token + ".extra"} {
		if _, e := DecodeCursor(bad, key, "owner-a", "all", "activity_refresh", 50); e == nil {
			t.Errorf("accepted malformed/tampered %q", bad)
		}
	}
}
func TestCursorRejectsInvalidShape(t *testing.T) {
	key := []byte("test-only-cursor-key")
	base := Cursor{Version: 1, Kind: "activity_refresh", PrincipalID: "owner-a", FilterDigest: "all", Direction: "desc", PageSize: 50, SnapshotID: "100", UpperID: "100", LowerID: "90"}
	mutations := []func(*Cursor){func(c *Cursor) { c.Version = 2 }, func(c *Cursor) { c.Kind = "other" }, func(c *Cursor) { c.Direction = "asc" }, func(c *Cursor) { c.PageSize = 101 }, func(c *Cursor) { c.PrincipalID = "" }, func(c *Cursor) { c.SnapshotID = "9223372036854775808" }, func(c *Cursor) { c.LowerID = "101" }, func(c *Cursor) { c.UpperID = "101" }, func(c *Cursor) { c.SnapshotID = "0100" }}
	for i, mutate := range mutations {
		c := base
		mutate(&c)
		if _, e := EncodeCursor(c, key); e == nil {
			t.Errorf("invalid case %d accepted: %#v", i, c)
		}
	}
	empty := base
	empty.Empty = true
	empty.UpperID = ""
	empty.LowerID = ""
	token, e := EncodeCursor(empty, key)
	if e != nil {
		t.Fatal(e)
	}
	got, e := DecodeCursor(token, key, "owner-a", "all", "activity_refresh", 50)
	if e != nil || !got.Empty {
		t.Fatal(got, e)
	}
}
func TestPageSizeBounds(t *testing.T) {
	for _, tc := range []struct {
		in, want int32
		bad      bool
	}{{0, 50, false}, {1, 1, false}, {100, 100, false}, {-1, 0, true}, {101, 0, true}} {
		got, e := normalizePageSize(tc.in)
		if (e != nil) != tc.bad || got != tc.want {
			t.Errorf("size %d got %d %v", tc.in, got, e)
		}
	}
}
