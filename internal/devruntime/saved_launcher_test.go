package devruntime

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSavedLauncherWorksWithoutCompilerAndRejectsTampering(t *testing.T) {
	checkout, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	launcher := filepath.Join(checkout, "hack/run-local-runtime.sh")
	for _, fault := range []string{"", "digest", "symlink", "checkout"} {
		t.Run(fault, func(t *testing.T) {
			key := testKey(t)
			state := initialState(t, key)
			source := filepath.Join(key.Checkout, "fixture-runner")
			if err := os.WriteFile(source, []byte("#!/bin/sh\nprintf 'saved-runner:%s\\n' \"$1\"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			saved, err := ImmutableExecutable(key, source)
			if err != nil {
				t.Fatal(err)
			}
			state.Supervisor = ProcessIdentity{PID: os.Getpid(), BootID: "previous-boot", Exe: saved}
			if fault == "checkout" {
				state.Supervisor.Exe = "/tmp/foreign"
			}
			if err := SaveState(key.StatePath(), state); err != nil {
				t.Fatal(err)
			}
			if fault == "digest" {
				if err := os.WriteFile(saved, []byte("#!/bin/sh\necho WRONG\n"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			if fault == "symlink" {
				os.Remove(saved)
				if err := os.Symlink(source, saved); err != nil {
					t.Fatal(err)
				}
			}
			// No Go compiler on PATH, and invalid source in this independent checkout.
			os.WriteFile(filepath.Join(key.Checkout, "go.mod"), []byte("broken source"), 0600)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			for _, action := range []string{"status", "stop"} {
				cmd := exec.CommandContext(ctx, "/bin/bash", launcher, action, "--instance", key.Name)
				cmd.Dir = key.Checkout
				cmd.Env = []string{"PATH=/usr/bin:/bin"}
				output, err := cmd.CombinedOutput()
				if fault == "" {
					if err != nil || !strings.Contains(string(output), "saved-runner:"+action) {
						t.Fatalf("saved runner requires compiler: %v %s", err, output)
					}
				} else if err == nil || strings.Contains(string(output), "WRONG") {
					t.Fatalf("tampered runner executed: %v %s", err, output)
				}
			}
		})
	}
}

func TestSavedLauncherActualStatusAndStopWithBrokenCheckout(t *testing.T) {
	checkout, _ := filepath.Abs("../..")
	key := testKey(t)
	source := filepath.Join(t.TempDir(), "athena-local-runtime")
	build := exec.Command("go", "build", "-o", source, "./cmd/athena-local-runtime")
	build.Dir = checkout
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build fixture runner: %v %s", err, output)
	}
	saved, err := ImmutableExecutable(key, source)
	if err != nil {
		t.Fatal(err)
	}
	command, p := temporaryProcess(t, "exec sleep 60")
	time.Sleep(20 * time.Millisecond)
	p, _ = ReadProcess(command.Process.Pid)
	state := initialState(t, key)
	state.Services = []string{"etherscan-manager"}
	state.Processes["etherscan-manager"] = p
	state.Supervisor = ProcessIdentity{PID: os.Getpid(), BootID: "previous-boot", Exe: saved}
	state.StopBudgets = map[string]int64{"etherscan-manager": int64(time.Second)}
	if err = SaveState(key.StatePath(), state); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(key.Checkout, "go.mod"), []byte("invalid checkout source"), 0600)
	for _, action := range []string{"status", "stop"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		cmd := exec.CommandContext(ctx, "/bin/bash", filepath.Join(checkout, "hack/run-local-runtime.sh"), action, "--instance", key.Name)
		cmd.Dir = key.Checkout
		cmd.Env = []string{"PATH=/usr/bin:/bin"}
		output, err := cmd.CombinedOutput()
		cancel()
		if err != nil {
			t.Fatalf("%s: %v %s", action, err, output)
		}
		if action == "status" {
			var got State
			if err = json.Unmarshal(output, &got); err != nil || got.Health["etherscan-manager"] != "unavailable" {
				t.Fatalf("invalid actual status: %v %s", err, output)
			}
		}
	}
	if _, err = VerifyProcess(p); !os.IsNotExist(err) {
		t.Fatal("saved runner did not stop owned process", err)
	}
	state, err = LoadState(key.StatePath())
	if err != nil || state.Phase != "stopped" {
		t.Fatal(state, err)
	}
}
