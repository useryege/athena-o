package devruntime

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

func TestUnexpectedZeroExitPersistsAfterStop(t *testing.T) {
	m := helperFixture(t)
	if err := m.Update(func(s *State) error {
		s.Services = []string{"wallet"}
		s.Phase = "running"
		s.Health = map[string]string{"wallet": "ready"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("/bin/sleep", "0.1")
	cmd.Env = []string{}
	if _, err := m.Spawn("wallet", cmd, time.Second); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}
	if err := m.RecordExit("wallet", 0); err != nil {
		t.Fatal(err)
	}
	s, _ := m.Status()
	if s.Health["wallet"] == "ready" || len(s.Failures) == 0 {
		t.Fatalf("unexpected successful exit retained readiness/lost event: %+v", s)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	s, _ = m.Status()
	if len(s.Failures) == 0 || !strings.Contains(strings.Join(s.Failures, " "), "wallet") {
		t.Fatal("stop erased original exit", s)
	}
}

func TestStopPreservesStartupFailure(t *testing.T) {
	m := helperFixture(t)
	_ = m.Update(func(s *State) error { s.Failures = []string{"market-radar initial readiness failed"}; return nil })
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	s, _ := m.Status()
	if len(s.Failures) != 1 || s.Failures[0] != "market-radar initial readiness failed" {
		t.Fatal("stop erased original failure", s.Failures)
	}
}

func TestApplicationStagesIsolateBusinessFailure(t *testing.T) {
	for _, failed := range []string{"market-radar", "wallet", ""} {
		t.Run(failed, func(t *testing.T) {
			m := helperFixture(t)
			specs, _ := ResolveServices([]string{"market-radar", "wallet", "profit-sharing"})
			env := map[string]string{}
			paths := map[string]string{}
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			for i := range specs {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				_, port, _ := net.SplitHostPort(listener.Addr().String())
				listener.Close()
				spec := &specs[i]
				env[spec.PortKey] = port
				env["TEST_APPLICATION_NAME"] = spec.Name // each command gets its name as argument
				spec.Args = []string{"-test.run=TestLifecycleApplicationProcess", "--", spec.Name, port, failed}
				spec.StartupTimeout = 300 * time.Millisecond
				spec.ShutdownTimeout = time.Second
				paths[spec.Name] = binary
			}
			_ = m.Update(func(s *State) error {
				s.Services = []string{"wallet", "market-radar", "profit-sharing"}
				s.Logs = map[string]string{}
				s.Endpoints = map[string]string{}
				return nil
			})
			var children sync.WaitGroup
			err = m.startApplications(context.Background(), specs, paths, env, &children)
			state, _ := m.Status()
			if failed == "wallet" {
				if err == nil || len(state.Processes) != 1 {
					t.Fatalf("core failure did not abort: %v %+v", err, state)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !state.CoreUsable || state.Startup["profit-sharing"].Status != "ready" {
					t.Fatalf("business failure blocked core/later app: %+v", state)
				}
				if failed != "" {
					if state.SelectedReady || len(state.Failures) == 0 {
						t.Fatal("failure marked ready", state)
					}
					if _, err := VerifyProcess(state.Processes[failed]); !os.IsNotExist(err) {
						t.Fatal("failed process survived", err)
					}
				} else if !state.SelectedReady || state.FullStackReady {
					t.Fatal("selected/full readiness conflated", state)
				}
			}
			if err := m.Stop(context.Background()); err != nil {
				t.Fatal(err)
			}
			children.Wait()
			for _, p := range state.Processes {
				if _, err := VerifyProcess(p); !os.IsNotExist(err) {
					t.Fatal("process survived stop", err)
				}
			}
			state, _ = m.Status()
			if failed != "" && len(state.Failures) == 0 {
				t.Fatal("failure lost")
			}
		})
	}
}

func TestLifecycleApplicationProcess(t *testing.T) {
	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 {
		return
	}
	args := os.Args[separator+1:]
	name, port, failed := args[0], args[1], args[2]
	if failed == "exit-"+name {
		time.Sleep(50 * time.Millisecond)
		os.Exit(0)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		os.Exit(3)
	}
	server := grpc.NewServer()
	healthServer := health.NewServer()
	healthpb.RegisterHealthServer(server, healthServer)
	status := healthpb.HealthCheckResponse_SERVING
	if name == failed {
		status = healthpb.HealthCheckResponse_NOT_SERVING
	}
	healthServer.SetServingStatus("", status)
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()
	go func() { <-ctx.Done(); server.Stop() }()
	if err := server.Serve(listener); err != nil {
		os.Exit(4)
	}
	os.Exit(0)
}

func TestExpectedStopExitDoesNotCreateRuntimeFailure(t *testing.T) {
	m := helperFixture(t)
	_ = m.Update(func(s *State) error { s.Services = []string{"wallet"}; s.Phase = "stopping"; return nil })
	if err := m.RecordExit("wallet", -1); err != nil {
		t.Fatal(err)
	}
	state, _ := m.Status()
	if len(state.Failures) != 0 || len(state.Exits) != 1 || !state.Exits[0].Expected {
		t.Fatal(state)
	}
}

func TestStopLayersWaitAndSameLayerRunsConcurrently(t *testing.T) {
	m := helperFixture(t)
	marker := filepath.Join(m.Key.Checkout, "stops")
	for _, name := range []string{"api-server", "market-radar", "profit-sharing", "wallet"} {
		script := `trap 'echo ` + name + `-term >> "$1"; sleep 0.15; echo ` + name + `-done >> "$1"; exit 0' TERM; while :; do sleep 0.03; done`
		cmd := exec.Command("/bin/sh", "-c", script, "fixture", marker)
		cmd.Env = []string{}
		if _, err := m.Spawn(name, cmd, time.Second); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	}
	time.Sleep(30 * time.Millisecond)
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	events := strings.Fields(string(data))
	index := func(want string) int {
		for i, v := range events {
			if v == want {
				return i
			}
		}
		t.Fatalf("missing %s: %s", want, data)
		return -1
	}
	if index("api-server-done") > index("market-radar-term") || index("api-server-done") > index("profit-sharing-term") || index("wallet-term") < index("market-radar-done") || index("wallet-term") < index("profit-sharing-done") {
		t.Fatal("dependencies stopped before users", string(data))
	}
	if index("market-radar-term") > index("profit-sharing-done") || index("profit-sharing-term") > index("market-radar-done") {
		t.Fatal("same layer stopped serially", string(data))
	}
}

func TestOldBootProcessesCannotSignalCurrentPID(t *testing.T) {
	m := helperFixture(t)
	cmd, p := temporaryProcess(t, "exec sleep 60")
	time.Sleep(20 * time.Millisecond)
	p, _ = ReadProcess(cmd.Process.Pid)
	p.BootID = "previous-host-boot"
	_ = m.Update(func(s *State) error { s.Processes["wallet"] = p; s.Supervisor = p; return nil })
	calls := 0
	m.Signal = func(ProcessIdentity, syscall.Signal) error { calls++; return nil }
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal("old boot should allow recorded container recovery", err)
	}
	if calls != 0 {
		t.Fatal("attempted signal to previous boot PID")
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal("foreign process stopped", err)
	}
}

func TestStatusPersistsCurrentProbeWithoutErasingFailure(t *testing.T) {
	m := helperFixture(t)
	_ = m.Update(func(s *State) error {
		s.Services = []string{"wallet"}
		s.Phase = "running"
		s.FullStackReady = true
		s.Failures = []string{"original failure"}
		return nil
	})
	if err := m.SaveSecret("environment.json", []byte(`{"ATHENA_WALLET_POSTGRES_DSN":"postgres://user@127.0.0.1:1/wallet?connect_timeout=1"}`)); err != nil {
		t.Fatal(err)
	}
	state, err := Status(context.Background(), m.Key)
	if err != nil {
		t.Fatal(err)
	}
	if state.FullStackReady || state.Health["wallet"] == "ready" || state.Health["wallet-postgres"] != "unavailable" {
		t.Fatal("status lost current unavailable/owner probe", state)
	}
	stored, _ := m.Status()
	if len(stored.Probes) == 0 || len(stored.Failures) != 1 {
		t.Fatal("status failed to persist current probe/history", stored)
	}
}

func TestFailedBusinessExitDuringStartupContinuesOtherApps(t *testing.T) {
	m := helperFixture(t)
	specs, _ := ResolveServices([]string{"market-radar", "wallet", "profit-sharing"})
	env := map[string]string{}
	paths := map[string]string{}
	binary, _ := os.Executable()
	for i := range specs {
		spec := &specs[i]
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		_, port, _ := net.SplitHostPort(l.Addr().String())
		l.Close()
		env[spec.PortKey] = port
		spec.Args = []string{"-test.run=TestLifecycleApplicationProcess", "--", spec.Name, port, "exit-market-radar"}
		spec.StartupTimeout = 500 * time.Millisecond
		spec.ShutdownTimeout = time.Second
		paths[spec.Name] = binary
	}
	_ = m.Update(func(s *State) error {
		s.Services = []string{"wallet", "market-radar", "profit-sharing"}
		s.Logs = map[string]string{}
		s.Endpoints = map[string]string{}
		return nil
	})
	var children sync.WaitGroup
	err := m.startApplications(context.Background(), specs, paths, env, &children)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := m.Status()
	if state.Startup["market-radar"].Status != "failed" || state.Startup["profit-sharing"].Status != "ready" {
		t.Fatal(state)
	}
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	children.Wait()
}

func TestStopEscalationHonorsSharedDeadline(t *testing.T) {
	m := helperFixture(t)
	for _, name := range []string{"market-radar", "profit-sharing", "managed-oo"} {
		cmd := exec.Command("/bin/sh", "-c", `trap '' TERM; while :; do :; done`)
		cmd.Env = []string{}
		if _, err := m.Spawn(name, cmd, 30*time.Second); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })
	}
	time.Sleep(30 * time.Millisecond)
	ctx, cancel := context.WithTimeout(context.Background(), 1300*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := m.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) > 1300*time.Millisecond {
		t.Fatal("stop budget multiplied by service count")
	}
	state, _ := m.Status()
	for _, p := range state.Processes {
		if _, err := VerifyProcess(p); !os.IsNotExist(err) {
			t.Fatal("helper survived escalation", p, err)
		}
	}
}

func TestStopReusesSupervisorCleanupDeadline(t *testing.T) {
	m := helperFixture(t)
	cmd, p := temporaryProcess(t, "trap '' TERM; exec sleep 60")
	time.Sleep(20 * time.Millisecond)
	p, _ = ReadProcess(cmd.Process.Pid)
	_ = m.Update(func(s *State) error {
		s.Processes["wallet"] = p
		s.Supervisor.PID = os.Getpid()
		s.StopBudgets = map[string]int64{"wallet": int64(time.Second)}
		s.StopDeadline = time.Now().Add(250 * time.Millisecond)
		return nil
	})
	started := time.Now()
	if err := m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if time.Since(started) > 500*time.Millisecond {
		t.Fatal("cleanup deadline restarted instead of sharing helper/supervisor budget")
	}
}

func TestBeginAfterPreviousBootKeepsRecycledProcessAlive(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	s.Phase = "stopped"
	cmd, p := temporaryProcess(t, "exec sleep 60")
	time.Sleep(20 * time.Millisecond)
	p, _ = ReadProcess(cmd.Process.Pid)
	p.BootID = "previous-host-boot"
	s.Processes["wallet"] = p
	if err := SaveState(k.StatePath(), s); err != nil {
		t.Fatal(err)
	}
	attempt := exec.Command(os.Args[0], "-test.run=^TestBeginAttemptHelper$")
	attempt.Env = append(os.Environ(), "DEVRUNTIME_BEGIN_ATTEMPT="+k.Checkout, RunIDEnv+"="+NewRunID())
	output, err := attempt.CombinedOutput()
	if err != nil || !strings.Contains(string(output), "started\n") {
		t.Fatalf("could not begin after old boot recovery: %v %s", err, output)
	}
	if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal(err)
	}
	if err := NewManager(k).Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestFailedBusinessCleanupOwnsChildrenWithoutStoppingCore(t *testing.T) {
	m := helperFixture(t)
	core := exec.Command("/bin/sleep", "60")
	core.Env = []string{}
	coreID, err := m.Spawn("wallet", core, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { core.Process.Kill(); core.Wait() }()
	failed := exec.Command("/bin/bash", "-c", `trap ':' TERM; sleep 60 & child=$!; while kill -0 "$child" 2>/dev/null; do wait "$child"; done`)
	failed.Env = []string{"PATH=/usr/bin:/bin"}
	identity, err := m.Spawn("market-radar", failed, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { failed.Process.Kill(); failed.Wait() }()
	var members map[string]ProcessIdentity
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		members, err = discoverMembers(map[string]ProcessIdentity{"market-radar": identity}, nil, true)
		if err == nil && len(members) == 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if err != nil || len(members) != 2 {
		t.Fatalf("missing real child: %v %+v", err, members)
	}
	if err = m.stopService(context.Background(), "market-radar"); err != nil {
		t.Fatal(err)
	}
	for _, p := range members {
		if _, err := VerifyProcess(p); !os.IsNotExist(err) {
			t.Fatal("business descendant survived", err)
		}
	}
	if _, err := VerifyProcess(coreID); err != nil {
		t.Fatal("core was stopped", err)
	}
	if err = m.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestBusinessCleanupReportsLateUnverifiedChild(t *testing.T) {
	m := helperFixture(t)
	marker := filepath.Join(m.Key.Checkout, "armed")
	cmd := exec.Command("/bin/bash", "-c", `trap 'sleep 0.4 & exit 0' TERM; echo ready > "$1"; while :; do :; done`, "fixture", marker)
	cmd.Env = []string{"PATH=/usr/bin:/bin"}
	if _, err := m.Spawn("market-radar", cmd, time.Second); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			break
		}
		time.Sleep(time.Millisecond)
	}
	err := m.stopService(context.Background(), "market-radar")
	// This deliberately unrecordable child self-exits; never relax ancestry
	// checks or send a signal based only on its inherited group/run marker.
	time.Sleep(450 * time.Millisecond)
	if err == nil {
		t.Fatal("business cleanup claimed success with a late unverified child")
	}
}
