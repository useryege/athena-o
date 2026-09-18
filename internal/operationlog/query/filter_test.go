package query

import (
	"testing"
	"time"
)

func TestNormalizeFilterDefaultsAndBounds(t *testing.T) {
	now := time.Date(2026, 9, 18, 12, 0, 0, 0, time.UTC)
	f, err := NormalizeFilter(Filter{}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !f.From.Equal(now.Add(-7*24*time.Hour)) || !f.To.Equal(now) {
		t.Fatalf("defaults: %#v", f)
	}
	if f.PageSize != 50 {
		t.Fatalf("page size=%d", f.PageSize)
	}
	if _, err := NormalizeFilter(Filter{From: ptrTime(now)}, now); err == nil {
		t.Fatal("one-sided range accepted")
	}
	if _, err := NormalizeFilter(Filter{From: ptrTime(now.Add(-91 * 24 * time.Hour)), To: ptrTime(now)}, now); err == nil {
		t.Fatal("range over 90 days accepted")
	}
	if _, err := NormalizeFilter(Filter{PageSize: 101}, now); err == nil {
		t.Fatal("page size over 100 accepted")
	}
}
func TestNormalizeFilterDirectoryAndResourcePair(t *testing.T) {
	now := time.Now().UTC()
	if _, err := NormalizeFilter(Filter{ModuleCode: "account", ActionCode: "account.access.update"}, now); err != nil {
		t.Fatal(err)
	}
	if _, err := NormalizeFilter(Filter{ResourceType: "wallet"}, now); err == nil {
		t.Fatal("resource type without id accepted")
	}
	if _, err := NormalizeFilter(Filter{ResourceID: "x"}, now); err == nil {
		t.Fatal("resource id without type accepted")
	}
}
func ptrTime(v time.Time) *time.Time { return &v }
