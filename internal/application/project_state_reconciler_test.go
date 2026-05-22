package application

import (
	"testing"
	"time"
)

func TestProjectStateReconcilerJobIntervals(t *testing.T) {
	reconciler := &projectStateReconcilerImpl{}
	jobs := reconciler.reconcilerJobs()

	intervals := make(map[string]time.Duration, len(jobs))
	for _, job := range jobs {
		intervals[job.name] = job.interval
	}

	assertJobInterval(t, intervals, "sourcecode_refresh_active", 10*time.Second)
	assertJobInterval(t, intervals, "sourcecode_refresh_archived", archivedProjectRefreshInterval)
	assertJobInterval(t, intervals, "runtime_code_hash_refresh_active", sourceCodeRefreshInterval)
	assertJobInterval(t, intervals, "creator_other_projects_refresh_active", sourceCodeRefreshInterval)
}

func assertJobInterval(t *testing.T, intervals map[string]time.Duration, name string, want time.Duration) {
	t.Helper()

	got, ok := intervals[name]
	if !ok {
		t.Fatalf("job %q not found", name)
	}
	if got != want {
		t.Fatalf("job %q interval = %s, want %s", name, got, want)
	}
}
