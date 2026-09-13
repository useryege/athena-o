package devruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func initialState(t *testing.T, k InstanceKey) State {
	t.Helper()
	s := State{Version: 1, Key: k, RunID: "test-run", Phase: "running", DBMode: "managed", Processes: map[string]ProcessIdentity{}}
	if e := SaveState(k.StatePath(), s); e != nil {
		t.Fatal(e)
	}
	return s
}
func TestStopPreservesOtherInstanceAndStateOnSignalFailure(t *testing.T) {
	for _, permission := range []bool{false, true} {
		t.Run(fmt.Sprint(permission), func(t *testing.T) {
			k := testKey(t)
			s := initialState(t, k)
			c, p := temporaryProcess(t, "exec sleep 60")
			other, _ := temporaryProcess(t, "exec sleep 60")
			time.Sleep(20 * time.Millisecond)
			p, _ = ReadProcess(c.Process.Pid)
			s.Processes["service"] = p
			s.StopBudgets = map[string]int64{"service": int64(100 * time.Millisecond)}
			if e := SaveState(k.StatePath(), s); e != nil {
				t.Fatal(e)
			}
			m := NewManager(k)
			calls := 0
			if permission {
				m.Signal = func(_ ProcessIdentity, _ syscall.Signal) error { calls++; return syscall.EPERM }
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			e := m.Stop(ctx)
			final, _ := LoadState(k.StatePath())
			if permission {
				if e == nil || calls == 0 || len(final.Failures) == 0 || final.Phase == "stopped" {
					t.Fatalf("lost signal failure: %v %+v", e, final)
				}
			} else {
				if e != nil || final.Phase != "stopped" {
					t.Fatalf("stop: %v %+v", e, final)
				}
				if e = m.Stop(ctx); e != nil {
					t.Fatal(e)
				}
			}
			if e := other.Process.Signal(syscall.Signal(0)); e != nil {
				t.Fatal("other process was killed")
			}
		})
	}
}
func TestStopRejectsRecycledPID(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	c, p := temporaryProcess(t, "exec sleep 60")
	time.Sleep(20 * time.Millisecond)
	p, _ = ReadProcess(c.Process.Pid)
	p.StartTicks++
	s.Processes["stale"] = p
	_ = SaveState(k.StatePath(), s)
	m := NewManager(k)
	calls := 0
	m.Signal = func(ProcessIdentity, syscall.Signal) error { calls++; return nil }
	if e := m.Stop(context.Background()); e == nil {
		t.Fatal("ignored recycled PID")
	}
	if calls != 0 {
		t.Fatalf("signalled recycled PID %d", calls)
	}
	if e := c.Process.Signal(syscall.Signal(0)); e != nil {
		t.Fatal(e)
	}
}
func TestResetOnlyDeletesPreciselyOwnedResources(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	s.Phase = "stopped"
	s.Resources = []ResourceRef{{Kind: "volume", ID: "own-data", Name: "own-data", Namespace: k.Namespace, RunID: "older-run", Owned: true}, {Kind: "volume", ID: "external", Name: "external", Owned: false}}
	_ = SaveState(k.StatePath(), s)
	m := NewManager(k)
	var calls [][]string
	m.Docker.Exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "docker" {
			t.Fatal(name)
		}
		calls = append(calls, append([]string(nil), args...))
		if reflect.DeepEqual(args, []string{"volume", "inspect", "--format", "{{json .}}", "own-data"}) {
			return json.Marshal(map[string]any{"Name": "own-data", "Labels": ResourceLabels(k, "volume", "older-run", "own-data")})
		}
		if reflect.DeepEqual(args, []string{"volume", "rm", "own-data"}) {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected command %q", args)
	}
	if e := m.Reset(context.Background()); e != nil {
		t.Fatal(e)
	}
	if len(calls) != 2 {
		t.Fatal(calls)
	}
	final, _ := LoadState(k.StatePath())
	if len(final.Resources) != 1 || final.Resources[0].ID != "external" {
		t.Fatalf("wrong resources %+v", final.Resources)
	}
}
func TestDockerCleanupFailurePersistsEvidence(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	s.Resources = []ResourceRef{{Kind: "container", ID: "owned-id", Name: "db", Namespace: k.Namespace, RunID: s.RunID, Owned: true}}
	_ = SaveState(k.StatePath(), s)
	m := NewManager(k)
	m.Docker.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[1] == "inspect" {
			return json.Marshal(map[string]any{"Id": "owned-id", "Name": "/db", "Config": map[string]any{"Labels": ResourceLabels(k, "container", s.RunID, "db")}})
		}
		return nil, errors.New("cleanup denied")
	}
	if e := m.Stop(context.Background()); e == nil {
		t.Fatal("ignored docker failure")
	}
	final, _ := LoadState(k.StatePath())
	if final.Phase == "stopped" || len(final.Failures) == 0 || len(final.Resources) != 1 {
		t.Fatalf("lost failure %+v", final)
	}
}
func TestCreateIntentRecoveryAndNoCommandBeforeIntent(t *testing.T) {
	k := testKey(t)
	initialState(t, k)
	m := NewManager(k)
	created := false
	var calls [][]string
	m.Docker.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		s, e := LoadState(k.StatePath())
		if e != nil || len(s.Intents) != 1 {
			t.Fatalf("command before durable intent: %+v %v", s, e)
		}
		if args[1] == "create" {
			created = true
			return nil, errors.New("transport failed after daemon created volume")
		}
		if args[1] == "ls" {
			return []byte("volume-a\n"), nil
		}
		if args[1] == "inspect" {
			return json.Marshal(map[string]any{"Name": "volume-a", "Labels": ResourceLabels(k, "volume", "test-run", "volume-a")})
		}
		return nil, fmt.Errorf("unexpected args %v", args)
	}
	if _, e := m.CreateResource(context.Background(), "volume", "volume-a", nil); e == nil || !created {
		t.Fatal("creation failure lost")
	}
	if e := m.Recover(context.Background()); e != nil {
		t.Fatal(e)
	}
	s, _ := LoadState(k.StatePath())
	if len(s.Intents) != 0 || len(s.Resources) != 1 || s.Resources[0].ID != "volume-a" {
		t.Fatalf("recovery %+v", s)
	}

	wantRecovery := []string{"volume", "ls", "--quiet", "--filter", "label=athena.instance=test", "--filter", "label=athena.kind=volume", "--filter", "label=athena.namespace=" + k.Namespace, "--filter", "label=athena.resource=volume-a", "--filter", "label=athena.run=test-run"}
	if !reflect.DeepEqual(calls[1], wantRecovery) {
		t.Fatalf("wrong recovery argv: %v", calls[1])
	}
	wantCreate := []string{"volume", "create", "--label", "athena.instance=test", "--label", "athena.kind=volume", "--label", "athena.namespace=" + k.Namespace, "--label", "athena.resource=volume-a", "--label", "athena.run=test-run", "volume-a"}
	if !reflect.DeepEqual(calls[0], wantCreate) {
		t.Fatalf("wrong create argv: %v", calls[0])
	}

	// An unsafe state path prevents any external mutation.
	bad := testKey(t)
	_ = os.MkdirAll(bad.Dir(), 0700)
	_ = os.WriteFile(bad.StatePath(), []byte(`{"Version":999}`), 0600)
	m = NewManager(bad)
	m.Docker.Exec = func(context.Context, string, ...string) ([]byte, error) {
		t.Fatal("docker invoked without valid state intent")
		return nil, nil
	}
	if _, e := m.CreateResource(context.Background(), "volume", "b", nil); e == nil {
		t.Fatal("accepted invalid state")
	}
}
func TestResetRejectsActiveAndExternalInstances(t *testing.T) {
	for _, mode := range []string{"managed", "external"} {
		k := testKey(t)
		s := initialState(t, k)
		s.DBMode = mode
		if mode == "external" {
			s.Phase = "stopped"
		}
		_ = SaveState(k.StatePath(), s)
		m := NewManager(k)
		m.Docker.Exec = func(context.Context, string, ...string) ([]byte, error) {
			t.Fatal("reset touched forbidden resources")
			return nil, nil
		}
		if e := m.Reset(context.Background()); e == nil {
			t.Fatal("reset accepted forbidden state")
		}
	}
}

func TestStopRetainsInfrastructureWhenProcessCleanupFails(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	c, p := temporaryProcess(t, "exec sleep 60")
	time.Sleep(20 * time.Millisecond)
	p, _ = ReadProcess(c.Process.Pid)
	s.Processes["service"] = p
	s.Resources = []ResourceRef{{Kind: "container", ID: "db", Name: "db", Namespace: k.Namespace, RunID: s.RunID, Owned: true}}
	_ = SaveState(k.StatePath(), s)
	m := NewManager(k)
	m.Signal = func(ProcessIdentity, syscall.Signal) error { return syscall.EPERM }
	calls := 0
	m.Docker.Exec = func(context.Context, string, ...string) ([]byte, error) {
		calls++
		return nil, errors.New("must not stop infrastructure while users remain")
	}
	if e := m.Stop(context.Background()); e == nil {
		t.Fatal("lost process failure")
	}
	if calls != 0 {
		t.Fatalf("touched dependency with active process: %d calls", calls)
	}
}
func TestStopAcquiresLockWhileCreateIsSlow(t *testing.T) {
	k := testKey(t)
	initialState(t, k)
	m := NewManager(k)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	m.Docker.Exec = func(ctx context.Context, _ string, args ...string) ([]byte, error) {
		if args[1] == "create" {
			close(entered)
			<-release
			return nil, errors.New("creation canceled")
		}
		return nil, nil
	}
	go func() { _, e := m.CreateResource(context.Background(), "volume", "slow", nil); done <- e }()
	<-entered
	stopDone := make(chan error, 1)
	go func() { stopDone <- m.Stop(context.Background()) }()
	select {
	case e := <-stopDone:
		if e == nil {
			t.Fatal("unresolved create was treated as stopped")
		}
	case <-time.After(time.Second):
		t.Fatal("stop blocked on slow create")
	}
	close(release)
	<-done
	s, _ := m.Status()
	if len(s.Intents) != 1 || s.Phase == "stopped" {
		t.Fatalf("lost in-flight intent %+v", s)
	}
}

func TestRecoveryDisambiguatesMultipleResourcesOfSameKind(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	s.Intents = []CreationIntent{{Kind: "container", Name: "db", Namespace: k.Namespace, RunID: s.RunID}, {Kind: "container", Name: "redis", Namespace: k.Namespace, RunID: s.RunID}}
	_ = SaveState(k.StatePath(), s)
	m := NewManager(k)
	m.Docker.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[1] == "ls" {
			for _, arg := range args {
				if arg == "label=athena.resource=db" {
					return []byte("db-id"), nil
				}
				if arg == "label=athena.resource=redis" {
					return []byte("redis-id"), nil
				}
			}
			return []byte("db-id\nredis-id"), nil
		}
		id := args[len(args)-1]
		name := strings.TrimSuffix(id, "-id")
		labels := ResourceLabels(k, "container", s.RunID)
		labels["athena.resource"] = name
		return json.Marshal(map[string]any{"Id": id, "Name": "/" + name, "Config": map[string]any{"Labels": labels}})
	}
	if e := m.Recover(context.Background()); e != nil {
		t.Fatal(e)
	}
	s, _ = m.Status()
	if len(s.Resources) != 2 || len(s.Intents) != 0 {
		t.Fatalf("wrong recovered set %+v", s)
	}
}

func TestSupervisorHelper(t *testing.T) {
	checkout := os.Getenv("DEVRUNTIME_TEST_CHECKOUT")
	if checkout == "" {
		return
	}
	k, e := NewInstanceKey(checkout, "test")
	if e != nil {
		fmt.Fprintln(os.Stderr, "helper:", e)
		t.Fatal(e)
	}
	m := NewManager(k)
	if _, e = m.Begin([]string{"sleep"}, "managed"); e != nil {
		fmt.Fprintln(os.Stderr, "helper:", e)
		t.Fatal(e)
	}
	if _, e = m.Begin([]string{"sleep"}, "managed"); e == nil {
		t.Fatal("duplicate live supervisor accepted")
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()
	cmd := exec.Command("/bin/sleep", "60")
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	p, e := m.Spawn("sleep", cmd, 100*time.Millisecond)
	if e != nil {
		fmt.Fprintln(os.Stderr, "helper:", e)
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		e := cmd.Wait()
		code := 0
		if e != nil {
			code = cmd.ProcessState.ExitCode()
		}
		_ = m.RecordExit("sleep", code)
		done <- e
	}()
	if e = json.NewEncoder(os.Stdout).Encode(p); e != nil {
		fmt.Fprintln(os.Stderr, "helper:", e)
		t.Fatal(e)
	}
	<-ctx.Done()
	stop, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e = m.Stop(stop); e != nil {
		fmt.Fprintf(os.Stderr, "helper stop failed: %v\n", e)
		fmt.Fprintln(os.Stderr, "helper:", e)
		t.Fatal(e)
	}
	<-done
}
func TestForegroundSupervisorBeginSpawnRemoteStop(t *testing.T) {
	k := testKey(t)
	initial := initialState(t, k)
	initial.Phase = "stopped"
	initial.InitializationMode = "managed-account-state"
	initial.ConfigFingerprints = map[string]string{"database": "persistent-config"}
	if e := SaveState(k.StatePath(), initial); e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestSupervisorHelper$")
	cmd.Env = append(os.Environ(), "DEVRUNTIME_TEST_CHECKOUT="+k.Checkout, RunIDEnv+"="+NewRunID())
	out, e := cmd.StdoutPipe()
	if e != nil {
		t.Fatal(e)
	}
	cmd.Stderr = os.Stderr
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	var child ProcessIdentity
	if e = json.NewDecoder(out).Decode(&child); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = SignalProcess(child, syscall.SIGKILL) })
	s, e := LoadState(k.StatePath())
	if e != nil {
		t.Fatal(e)
	}
	if s.InitializationMode != "managed-account-state" || s.ConfigFingerprints["database"] != "persistent-config" {
		t.Fatal("restart lost persistent resource configuration")
	}
	if s.Supervisor.PID != cmd.Process.Pid || s.Processes["sleep"] != child || child.PGID != child.PID {
		t.Fatalf("wrong supervisor/child identity %+v", s)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if e = Stop(ctx, k); e != nil {
		t.Fatal(e)
	}
	if e = cmd.Wait(); e != nil {
		t.Fatal(e)
	}
	s, e = LoadState(k.StatePath())
	if e != nil || s.Phase != "stopped" {
		t.Fatalf("remote stop %v %+v", e, s)
	}
	if _, ok := s.ExitCodes["sleep"]; !ok {
		t.Fatal("supervisor did not persist child exit result")
	}
}

func TestStopCannotAdoptNewRunDuringSupervisorExit(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	c, p := temporaryProcess(t, "exec sleep 60")
	time.Sleep(20 * time.Millisecond)
	p, _ = ReadProcess(c.Process.Pid)
	s.Supervisor = p
	_ = SaveState(k.StatePath(), s)
	m := NewManager(k)
	m.Signal = func(p ProcessIdentity, sig syscall.Signal) error {
		e := SignalProcess(p, sig)
		_ = m.Update(func(s *State) error {
			s.RunID = "new-run"
			s.Phase = "running"
			s.Supervisor = ProcessIdentity{}
			return nil
		})
		return e
	}
	if e := m.Stop(context.Background()); e == nil {
		t.Fatal("stop adopted replacement run")
	}
	final, _ := m.Status()
	if final.RunID != "new-run" || final.Phase != "running" {
		t.Fatalf("stop mutated replacement run %+v", final)
	}
}

func TestStopIsolationAcrossNamesAndCheckouts(t *testing.T) {
	for _, separateCheckout := range []bool{false, true} {
		t.Run(fmt.Sprint(separateCheckout), func(t *testing.T) {
			root := t.TempDir()
			a, _ := NewInstanceKey(root, "a")
			if separateCheckout {
				root = t.TempDir()
			}
			b, _ := NewInstanceKey(root, "b")
			sa := initialState(t, a)
			sb := initialState(t, b)
			ca, pa := temporaryProcess(t, "exec sleep 60")
			cb, pb := temporaryProcess(t, "exec sleep 60")
			time.Sleep(20 * time.Millisecond)
			pa, _ = ReadProcess(ca.Process.Pid)
			pb, _ = ReadProcess(cb.Process.Pid)
			sa.Processes["service"] = pa
			sb.Processes["service"] = pb
			_ = SaveState(a.StatePath(), sa)
			_ = SaveState(b.StatePath(), sb)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if e := Stop(ctx, a); e != nil {
				t.Fatal(e)
			}
			if _, e := ReadProcess(pb.PID); e != nil {
				t.Fatal("stopped another instance")
			}
			after, e := LoadState(b.StatePath())
			if e != nil || !reflect.DeepEqual(after, sb) {
				t.Fatalf("changed other instance state %v %+v", e, after)
			}
		})
	}
}
func TestResetRejectsIncorrectOwnershipLabels(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	s.Phase = "stopped"
	s.Resources = []ResourceRef{{Kind: "volume", ID: "data", Name: "data", Namespace: k.Namespace, RunID: s.RunID, Owned: true}}
	_ = SaveState(k.StatePath(), s)
	m := NewManager(k)
	var calls [][]string
	m.Docker.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		labels := ResourceLabels(k, "volume", s.RunID, "data")
		labels["athena.namespace"] = "someone-else"
		return json.Marshal(map[string]any{"Name": "data", "Labels": labels})
	}
	if e := m.Reset(context.Background()); e == nil {
		t.Fatal("reset accepted another owner's labels")
	}
	want := [][]string{{"volume", "inspect", "--format", "{{json .}}", "data"}}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("attempted foreign deletion %v", calls)
	}
	final, _ := m.Status()
	if len(final.Resources) != 1 || len(final.Failures) == 0 {
		t.Fatal("lost refused resource evidence")
	}
}
func TestIdentityFailureDoesNotEraseOtherProcessRecords(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	ca, pa := temporaryProcess(t, "exec sleep 60")
	cb, pb := temporaryProcess(t, "exec sleep 60")
	time.Sleep(20 * time.Millisecond)
	pa, _ = ReadProcess(ca.Process.Pid)
	pb, _ = ReadProcess(cb.Process.Pid)
	pa.StartTicks++
	s.Processes["stale"] = pa
	s.Processes["owned"] = pb
	_ = SaveState(k.StatePath(), s)
	if e := Stop(context.Background(), k); e == nil {
		t.Fatal("ignored stale identity")
	}
	final, _ := LoadState(k.StatePath())
	if len(final.Processes) != 2 {
		t.Fatal("lost other process identity")
	}
}

func TestStopEscalatesThroughVerifiedPidfdAfterTotalBudget(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	c, p := temporaryProcess(t, "trap '' TERM; exec sleep 60")
	time.Sleep(20 * time.Millisecond)
	p, _ = ReadProcess(c.Process.Pid)
	s.Processes["service"] = p
	s.StopBudgets = map[string]int64{"service": int64(100 * time.Millisecond)}
	_ = SaveState(k.StatePath(), s)
	m := NewManager(k)
	var signals []syscall.Signal
	m.Signal = func(target ProcessIdentity, sig syscall.Signal) error {
		if target.PID != p.PID || target.PID <= 0 || target != p {
			t.Fatalf("unverified signal target %+v", target)
		}
		signals = append(signals, sig)
		return SignalProcess(target, sig)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	start := time.Now()
	if e := m.Stop(ctx); e != nil {
		t.Fatal(e)
	}
	if elapsed := time.Since(start); elapsed < 100*time.Millisecond || elapsed >= time.Second {
		t.Fatalf("wrong total budget %s", elapsed)
	}
	if !reflect.DeepEqual(signals, []syscall.Signal{syscall.SIGTERM, syscall.SIGKILL}) {
		t.Fatalf("wrong signals %v", signals)
	}
	if e := c.Wait(); e == nil {
		t.Fatal("fixture was not killed")
	}
	if c.ProcessState.Sys().(syscall.WaitStatus).Signal() != syscall.SIGKILL {
		t.Fatal("did not enforce deadline with SIGKILL")
	}
}
func TestCreateCannotOverrideOwnershipLabels(t *testing.T) {
	for _, arg := range []string{"--label", "--label=athena.namespace=foreign", "-lathena.namespace=foreign", "--label-file"} {
		k := testKey(t)
		initialState(t, k)
		m := NewManager(k)
		calls := 0
		m.Docker.Exec = func(context.Context, string, ...string) ([]byte, error) {
			calls++
			return nil, errors.New("unexpected docker mutation")
		}
		if _, e := m.CreateResource(context.Background(), "container", "db", []string{arg, "postgres:18"}); e == nil {
			t.Fatal("accepted override")
		}
		s, _ := m.Status()
		if calls != 0 || len(s.Intents) != 0 {
			t.Fatalf("override passed preflight: %s", arg)
		}
	}
}
