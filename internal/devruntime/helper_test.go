package devruntime

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
