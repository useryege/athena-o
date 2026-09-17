package devruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRunLifecycleSharedFailureCancellationAndBusinessIsolation(t *testing.T) {
	for _, scenario := range []string{"build-failure", "cancel-build", "core-failure", "business-failure", "success"} {
		t.Run(scenario, func(t *testing.T) {
			k := testKey(t)
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			binDir := filepath.Join(k.Checkout, "tools")
			os.Mkdir(binDir, 0700)
			// A real compiler command boundary: these fixtures deliberately exit or stay
			// alive; the application executable is this test's real gRPC helper process.
			compiler := "#!/bin/sh\ncp '" + strings.ReplaceAll(binary, "'", "'\\''") + "' \"$3\"\n"
			if scenario == "build-failure" {
				compiler = "#!/bin/sh\necho diagnostic-build-failure >&2\nexit 42\n"
			}
			if scenario == "cancel-build" {
				compiler = "#!/bin/sh\nsleep 60\n"
			}
			if err = os.WriteFile(filepath.Join(binDir, "go"), []byte(compiler), 0700); err != nil {
				t.Fatal(err)
			}
			values := map[string]string{}
			for _, name := range []string{"wallet", "market-radar", "profit-sharing"} {
				l, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				_, port, _ := net.SplitHostPort(l.Addr().String())
				l.Close()
				values[serviceRegistry()[name].PortKey] = port
			}
			config := []string{}
			for key, value := range values {
				config = append(config, key+"="+value)
			}
			os.WriteFile(filepath.Join(k.Checkout, ".env"), []byte(strings.Join(config, "\n")), 0600)
			cmd := exec.Command(binary, "-test.run=TestLifecycleSupervisorProcess", "--", k.Checkout, scenario)
			cmd.Env = []string{RunIDEnv + "=" + NewRunID(), "PATH=" + binDir + ":/usr/bin:/bin", "HOME=" + k.Checkout}
			output := filepath.Join(k.Checkout, "supervisor.log")
			log, err := os.Create(output)
			if err != nil {
				t.Fatal(err)
			}
			defer log.Close()
			cmd.Stdout = log
			cmd.Stderr = log
			if err = cmd.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			t.Cleanup(func() { cmd.Process.Kill(); <-done }) // drained below is replaced with a buffered result
			if scenario == "business-failure" || scenario == "success" {
				deadline := time.Now().Add(8 * time.Second)
				ready := false
				for time.Now().Before(deadline) {
					state, e := LoadState(k.StatePath())
					if e == nil && state.Phase == "running" {
						ready = true
						if !state.CoreUsable || (scenario == "business-failure" && state.SelectedReady) {
							t.Fatal("incorrect core/full state", state)
						}
						break
					}
					time.Sleep(20 * time.Millisecond)
				}
				if !ready {
					data, _ := os.ReadFile(output)
					t.Fatalf("supervisor did not settle: %s", data)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				err = NewManager(k).Stop(ctx)
				cancel()
				if err != nil {
					t.Fatal(err)
				}
			}
			select {
			case err = <-done:
				done <- err
			case <-time.After(10 * time.Second):
				t.Fatal("supervisor did not exit")
			}
			if (scenario == "success") != (err == nil) {
				data, _ := os.ReadFile(output)
				t.Fatalf("unexpected exit: %v %s", err, data)
			}
			state, e := LoadState(k.StatePath())
			if e != nil {
				t.Fatal(e)
			}
			if state.Phase != "stopped" || (scenario != "success" && len(state.Failures) == 0) {
				t.Fatalf("cleanup/history missing: %+v", state)
			}
			for _, p := range state.Processes {
				if _, err = VerifyProcess(p); !os.IsNotExist(err) {
					t.Fatal("owned process survived", p, err)
				}
			}
			for name, path := range state.Logs {
				if data, err := os.ReadFile(path); err == nil {
					t.Logf("preserved-log scenario=%s service=%s contents=%q", scenario, name, data)
				}
			}
			if data, err := os.ReadFile(output); err == nil {
				t.Logf("supervisor-log scenario=%s contents=%q", scenario, data)
			}
			encoded, _ := json.Marshal(state)
			t.Logf("scenario=%s final=%s", scenario, encoded)
		})
	}
}

func TestLifecycleSupervisorProcess(t *testing.T) {
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
	checkout, scenario := args[0], args[1]
	key, err := NewInstanceKey(checkout, "test")
	if err != nil {
		os.Exit(3)
	}
	specs, _ := ResolveServices([]string{"market-radar", "wallet", "profit-sharing"})
	env, err := LoadEnvironment(filepath.Join(checkout, ".env"), nil)
	if err != nil {
		os.Exit(4)
	}
	failed := ""
	if scenario == "core-failure" {
		failed = "wallet"
	}
	if scenario == "business-failure" {
		failed = "market-radar"
	}
	for i := range specs {
		spec := &specs[i]
		spec.Infrastructure = nil
		spec.Schemas = nil
		spec.StartupTimeout = 200 * time.Millisecond
		spec.ShutdownTimeout = time.Second
		spec.Args = []string{"-test.run=TestLifecycleApplicationProcess", "--", spec.Name, env[spec.PortKey], failed}
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	defer cancel()
	if scenario == "cancel-build" {
		var stop context.CancelFunc
		ctx, stop = context.WithTimeout(ctx, 300*time.Millisecond)
		defer stop()
	}
	err = runResolved(ctx, RunOptions{Key: key, Services: []string{"market-radar", "wallet", "profit-sharing"}, DBMode: "managed"}, specs, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
