package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

func TestProjectStateReconcilerReconcileOnceRunsJobsInOrderAndReturnsError(t *testing.T) {
	wantErr := errors.New("stop")
	var got []string
	reconciler := &projectStateReconcilerImpl{
		jobSem: make(chan struct{}, 1),
	}
	jobs := []reconcilerJob{
		{name: "first", run: func(context.Context) error {
			got = append(got, "first")
			return nil
		}},
		{name: "second", run: func(context.Context) error {
			got = append(got, "second")
			return wantErr
		}},
		{name: "third", run: func(context.Context) error {
			got = append(got, "third")
			return nil
		}},
	}

	err := reconciler.reconcileOnceJobs(context.Background(), jobs)
	if !errors.Is(err, wantErr) {
		t.Fatalf("ReconcileOnce error = %v, want %v", err, wantErr)
	}
	if len(got) != 2 || got[0] != "first" || got[1] != "second" {
		t.Fatalf("job order = %v, want [first second]", got)
	}
}

func TestProjectStateReconcilerReconcileOnceSuppressesPolicyTriggers(t *testing.T) {
	triggerCh := make(chan common.Address, 1)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	reconciler := &projectStateReconcilerImpl{
		policyTriggerCh: triggerCh,
		jobSem:          make(chan struct{}, 1),
	}

	err := reconciler.reconcileOnceJobs(context.Background(), []reconcilerJob{{
		name: "trigger",
		run: func(context.Context) error {
			reconciler.triggerPolicyEvaluation(contract, "test")
			return nil
		},
	}})
	if err != nil {
		t.Fatalf("ReconcileOnce: %v", err)
	}
	select {
	case got := <-triggerCh:
		t.Fatalf("unexpected policy trigger %s", got.Hex())
	default:
	}
}
