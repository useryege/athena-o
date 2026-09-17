package devruntime

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestImmutableExecutableSurvivesAnotherBuild(t *testing.T) {
	k, _ := NewInstanceKey(t.TempDir(), "binary")
	source := filepath.Join(k.Checkout, "source")
	data, e := os.ReadFile("/bin/sleep")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(source, data, 0700); e != nil {
		t.Fatal(e)
	}
	first, e := ImmutableExecutable(k, source)
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(first, "10")
	runID := NewRunID()
	cmd.Env = []string{RunIDEnv + "=" + runID}
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	// /proc/environ can briefly be empty while exec finishes. Establish
	// the baseline only after the executable and run marker are observable.
	var p ProcessIdentity
	deadline := time.Now().Add(2 * time.Second)
	for {
		p, e = ReadProcess(cmd.Process.Pid)
		if e == nil && p.Exe == first && p.RunID == runID && p.StartTicks > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("child did not finish exec: %+v %v", p, e)
		}
		time.Sleep(time.Millisecond)
	}
	if e = os.WriteFile(source, []byte("replaced"), 0700); e != nil {
		t.Fatal(e)
	}
	if _, e = ImmutableExecutable(k, source); e != nil {
		t.Fatal(e)
	}
	now, e := ReadProcess(cmd.Process.Pid)
	if e != nil || !SameProcess(p, now) {
		t.Fatalf("second build invalidated first process identity: before=%+v after=%+v error=%v", p, now, e)
	}
}
func TestPortConflictLeavesListenerAlive(t *testing.T) {
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	s, _ := ResolveServices([]string{"trader-sync"})
	if e = preflightPorts(s, map[string]string{"ATHENA_TRADER_SYNC_LISTEN_ADDRESS": l.Addr().String()}); e == nil {
		t.Fatal("conflict accepted")
	}
	c, e := net.Dial("tcp", l.Addr().String())
	if e != nil {
		t.Fatal("existing listener affected", e)
	}
	c.Close()
}
func TestUnknownRunFailsBeforeState(t *testing.T) {
	k, _ := NewInstanceKey(t.TempDir(), "bad")
	if e := Run(context.Background(), RunOptions{Key: k, Services: []string{"shell;bad"}, DBMode: "managed"}); e == nil {
		t.Fatal("unknown accepted")
	}
	if _, e := os.Stat(k.StatePath()); !os.IsNotExist(e) {
		t.Fatal("resources allocated before validation")
	}
}
func TestServiceDefaultAddressesAreExplicit(t *testing.T) {
	if got := serviceAddress("notification", nil); got != "127.0.0.1:8086" {
		t.Fatal(got)
	}
}
func TestBuildRejectsUIInsteadOfReportingNoopSuccess(t *testing.T) {
	k, _ := NewInstanceKey(t.TempDir(), "ui")
	if e := Build(context.Background(), k, []string{"ui"}); e == nil {
		t.Fatal("UI has no independent Go binary")
	}
}
