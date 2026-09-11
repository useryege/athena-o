package store

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestRuntimeMetricsSQLProjectsOnlySafeFields(t *testing.T) {
	for _, path := range []string{"queries/metrics.sql", "sqlc/metrics.sql.go"} {
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		s := string(raw)
		if regexp.MustCompile(`(?i)select\s+\w+\.\*`).MatchString(s) {
			t.Errorf("%s reads private full rows", path)
		}
		for _, name := range []string{"note_snapshot", "target_display_snapshot", "trade_json", "raw_json", "payload", "d.title", "d.body", "d.link"} {
			if strings.Contains(s, name) {
				t.Errorf("%s requests private column %s", path, name)
			}
		}
		if regexp.MustCompile(`(?i)select[^;]*a\.formation_evidence\s*[,\n]`).MatchString(s) {
			t.Errorf("%s projects full formation evidence", path)
		}
	}
}

// Check the exact aggregate SELECT separately from observation-write queries.
// A safe response alone cannot establish that the administrator avoided raw reads.
func TestFinalityRuntimeSQLProjectsOnlyTimingScalars(t *testing.T) {
	for _, path := range []string{"queries/finality.sql", "sqlc/finality.sql.go"} {
		raw, e := os.ReadFile(path)
		if e != nil {
			t.Fatal(e)
		}
		start := strings.Index(string(raw), "WITH facts AS (")
		if start < 0 {
			t.Fatalf("%s has no aggregate", path)
		}
		sql := string(raw)[start:]
		if end := strings.Index(sql, "ORDER BY name"); end >= 0 {
			sql = sql[:end]
		}
		for _, name := range []string{"raw_json", "wallet", "owner_id", "transaction_hash", "trade_json", "SELECT *", "SELECT finality_timing FROM"} {
			if strings.Contains(sql, name) {
				t.Errorf("%s aggregate reads %s", path, name)
			}
		}
	}
}
