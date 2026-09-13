package tradersync

import (
	"math"
	"strconv"
	"testing"
)

func FuzzCursorRoundTrip(f *testing.F) {
	for _, snapshot := range []int64{0, 1, 9007199254740993, math.MaxInt64, -1} {
		f.Add(snapshot)
	}

	key := []byte("test-only-cursor-key")
	f.Fuzz(func(t *testing.T, snapshot int64) {
		cursor := Cursor{
			Version:      1,
			Kind:         "activity_snapshot",
			PrincipalID:  "owner-a",
			FilterDigest: "all",
			Direction:    "desc",
			PageSize:     50,
			SnapshotID:   strconv.FormatInt(snapshot, 10),
		}
		token, err := EncodeCursor(cursor, key)
		if snapshot < 0 {
			if err == nil {
				t.Fatalf("negative snapshot accepted: %d", snapshot)
			}
			return
		}
		if err != nil {
			t.Fatalf("encode snapshot %d: %v", snapshot, err)
		}
		got, err := DecodeCursor(token, key, "owner-a", "all", "activity_snapshot", 50)
		if err != nil || got != cursor {
			t.Fatalf("round trip snapshot %d: %#v %v", snapshot, got, err)
		}
		if _, err := DecodeCursor(token, key, "owner-b", "all", "activity_snapshot", 50); err == nil {
			t.Fatalf("cross-account cursor accepted for snapshot %d", snapshot)
		}
	})
}
