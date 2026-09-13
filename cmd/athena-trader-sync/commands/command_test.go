package commands

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/tradersync"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type blockedRuntime struct{ worker chan struct{} }

func (r *blockedRuntime) ShutdownLogFields() tradersync.RuntimeLogFields {
	return tradersync.RuntimeLogFields{Instance: "unmanaged", Run: "direct", RuntimeGeneration: "unknown", CollectorEpoch: "unknown"}
}
func (r *blockedRuntime) Wait() error                    { return nil }
func (r *blockedRuntime) Shutdown(context.Context) error { <-r.worker; return nil }
func TestRuntimeHelperProcess(t *testing.T) {
	if os.Getenv("ATHENA_TEST_BLOCKED_RUNTIME") != "1" {
		return
	}
	r := &blockedRuntime{worker: make(chan struct{})}
	runLifecycle(context.Background(), r, time.Second)
	os.Exit(0)
}
func TestRuntimeWatchdogTerminatesActualProcess(t *testing.T) {
	child := exec.Command(os.Args[0], "-test.run=^TestRuntimeHelperProcess$")
	child.Env = append(os.Environ(), "ATHENA_TEST_BLOCKED_RUNTIME=1", "ATHENA_LOCAL_RUNTIME_INSTANCE=", "ATHENA_LOCAL_RUNTIME_RUN_ID=")
	var logs bytes.Buffer
	child.Stdout = &logs
	child.Stderr = &logs
	require.NoError(t, child.Start())
	started := time.Now()
	err := child.Wait()
	require.Error(t, err)
	require.Less(t, time.Since(started), 5*time.Second)
	line := watchdogLogLine(t, logs.String())
	require.Contains(t, line, `instance="unmanaged"`)
	require.Contains(t, line, `run="direct"`)
	require.Contains(t, line, "runtime_generation=unknown")
	require.Contains(t, line, "collector_epoch=unknown")
	t.Log(logs.String())
}

func watchdogLogLine(t *testing.T, logs string) string {
	t.Helper()
	for _, line := range strings.Split(logs, "\n") {
		if strings.Contains(line, "shutdown deadline exceeded") {
			return line
		}
	}
	t.Fatal("watchdog final line missing")
	return ""
}
