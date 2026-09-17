package devruntime

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func helperFixture(t *testing.T) *Manager {
	t.Helper()
	k := testKey(t)
	s := initialState(t, k)
	s.Phase = "starting"
	if err := SaveState(k.StatePath(), s); err != nil {
		t.Fatal(err)
	}
	m := NewManager(k)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.Stop(ctx); err != nil {
			t.Error(err)
		}
	})
	return m
}

func TestHelperPersistsCompilerDiagnosticsAndExit(t *testing.T) {
	m := helperFixture(t)
	source := filepath.Join(m.Key.Checkout, "broken.go")
	if err := os.WriteFile(source, []byte("package main\nfunc main() { missingTask9Symbol() }\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", "-o", filepath.Join(m.Key.Checkout, "broken"), source)
	cmd.Env = EnvironmentFor(environmentMap(os.Environ()), []string{"PATH", "HOME", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN"})
	err := m.RunHelper(context.Background(), "compiler-error", cmd, time.Second)
	if err == nil {
		t.Fatal("broken build succeeded")
	}
	s, e := m.Status()
	if e != nil {
		t.Fatal(e)
	}
	if len(s.Processes) != 1 || len(s.ExitCodes) != 1 || len(s.Logs) != 1 {
		t.Fatal("helper evidence missing", s.Processes, s.ExitCodes, s.Logs)
	}
	for name, log := range s.Logs {
		data, e := os.ReadFile(log)
		if e != nil {
			t.Fatal(e)
		}
		info, e := os.Stat(log)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(string(data), "undefined: missingTask9Symbol") || !strings.Contains(err.Error(), log) || info.Mode().Perm() != 0600 || s.ExitCodes[name] == 0 {
			t.Fatalf("missing compiler diagnostics: %v; %s", err, data)
		}
		if _, e = VerifyProcess(s.Processes[name]); !os.IsNotExist(e) {
			t.Fatal("helper was not reaped", e)
		}
	}
}

func TestHelperCancellationReapsTargetAndDurableRoot(t *testing.T) {
	m := helperFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.Command("/bin/sleep", "60")
	cmd.Env = []string{}
	done := make(chan error, 1)
	go func() { done <- m.RunHelper(ctx, "cancel", cmd, time.Second) }()
	var members map[string]ProcessIdentity
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		s, e := m.Status()
		if e != nil {
			t.Fatal(e)
		}
		members, e = DiscoverMembers(s.Processes)
		if e == nil && len(members) == 2 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(members) != 2 {
		t.Fatal("target did not acquire durable ancestry", members)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("helper cancellation did not complete")
	}
	for _, p := range members {
		if _, e := VerifyProcess(p); !os.IsNotExist(e) {
			t.Fatal("helper survived cancellation", e)
		}
	}
	s, e := m.Status()
	if e != nil || s.Phase != "stopped" {
		t.Fatal(s.Phase, e)
	}
}

func TestHelperReapUsesExistingStopDeadlineAfterSignalFailure(t *testing.T) {
	m := helperFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	command := exec.Command("/bin/sleep", "60")
	command.Env = []string{}
	done := make(chan error, 1)
	var members map[string]ProcessIdentity
	var rejectSignals atomic.Bool
	rejectSignals.Store(true)
	t.Cleanup(func() {
		// Restore the real ownership-checked stopper, with a fresh recovery budget.
		rejectSignals.Store(false)
		_ = m.Update(func(s *State) error { s.Supervisor = ProcessIdentity{}; s.StopDeadline = time.Time{}; return nil })
		cleanup, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		if err := m.Stop(cleanup); err != nil {
			t.Error("fixture cleanup", err)
		}
		for _, p := range members {
			if _, err := VerifyProcess(p); !os.IsNotExist(err) {
				t.Errorf("fixture process survived: %+v %v", p, err)
			}
		}
		deadline := time.Now().Add(time.Second)
		for time.Now().Before(deadline) {
			state, err := m.Status()
			if err == nil && len(state.ExitCodes) == 1 {
				t.Logf("cleanup verified: %d identities exited and helper Wait recorded", len(members))
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Error("fixture helper was not reaped and recorded")
	})
	// Establish the failure boundary before starting the goroutine; no mutable
	// callback is exchanged while Stop is running.
	m.Signal = func(p ProcessIdentity, signal syscall.Signal) error {
		if rejectSignals.Load() {
			return syscall.EPERM
		}
		return SignalProcess(p, signal)
	}
	go func() { done <- m.RunHelper(ctx, "prior-stop-deadline", command, time.Second) }()
	discoveryDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(discoveryDeadline) {
		state, err := m.Status()
		if err != nil {
			t.Fatal(err)
		}
		members, err = DiscoverMembers(state.Processes)
		if err == nil && len(members) == 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if len(members) != 2 {
		t.Fatal("real helper and target were not registered", members)
	}
	sharedDeadline := time.Now().Add(200 * time.Millisecond)
	if err := m.Update(func(s *State) error { s.Supervisor.PID = os.Getpid(); s.StopDeadline = sharedDeadline; return nil }); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) || !errors.Is(err, syscall.EPERM) || !strings.Contains(err.Error(), "within cleanup budget") {
			t.Errorf("lost cancellation, stop failure or budget result: %v", err)
		}
		if time.Now().Before(sharedDeadline) {
			t.Error("did not wait for the shared deadline")
		}
		for _, p := range members {
			if _, err := VerifyProcess(p); err != nil {
				t.Errorf("helper must remain alive at failed-stop boundary: %+v %v", p, err)
			}
		}
	case <-time.After(time.Until(sharedDeadline) + 300*time.Millisecond):
		t.Error("RunHelper reap wait outlived the existing StopDeadline")
	}
}
