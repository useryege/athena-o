package commands

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"os/exec"
	"testing"
	"time"
)

type blockedRuntime struct{ worker chan struct{} }

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
	child.Env = append(os.Environ(), "ATHENA_TEST_BLOCKED_RUNTIME=1")
	var logs bytes.Buffer
	child.Stdout = &logs
	child.Stderr = &logs
	require.NoError(t, child.Start())
	started := time.Now()
	err := child.Wait()
	require.Error(t, err)
	require.Less(t, time.Since(started), 5*time.Second)
	require.Contains(t, logs.String(), "shutdown deadline exceeded")
	t.Log(logs.String())
}
