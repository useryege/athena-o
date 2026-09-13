package devruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func temporaryProcess(t *testing.T, script string) (*exec.Cmd, ProcessIdentity) {
	t.Helper()
	c := exec.Command("/bin/sh", "-c", script)
	c.Env = append(os.Environ(), RunIDEnv+"=test-"+NewRunID())
	c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if e := c.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = c.Process.Kill(); _ = c.Wait() })
	p, e := ReadProcess(c.Process.Pid)
	if e != nil {
		t.Fatal(e)
	}
	return c, p
}
func TestSameProcessRejectsIdentityReuse(t *testing.T) {
	p := ProcessIdentity{PID: 42, PGID: 42, StartTicks: 123, BootID: "boot", Exe: "/bin/test", RunID: "run"}
	for _, change := range []func(*ProcessIdentity){func(x *ProcessIdentity) { x.StartTicks++ }, func(x *ProcessIdentity) { x.RunID = "other" }, func(x *ProcessIdentity) { x.PGID++ }, func(x *ProcessIdentity) { x.BootID = "other" }, func(x *ProcessIdentity) { x.Exe = "other" }} {
		q := p
		change(&q)
		if SameProcess(p, q) {
			t.Fatal("accepted recycled identity")
		}
	}
	if SameProcess(ProcessIdentity{}, ProcessIdentity{}) {
		t.Fatal("accepted empty identities")
	}
}
func TestPidfdStopsOnlyVerifiedProcess(t *testing.T) {
	c, p := temporaryProcess(t, "exec sleep 60") // exec may change exe before capture; wait for stable image.
	time.Sleep(20 * time.Millisecond)
	p, _ = ReadProcess(c.Process.Pid)
	bad := p
	bad.StartTicks++
	if e := SignalProcess(bad, syscall.SIGTERM); !errors.Is(e, ErrIdentityMismatch) {
		t.Fatalf("wrong stale signal result %v", e)
	}
	if e := c.Process.Signal(syscall.Signal(0)); e != nil {
		t.Fatal("stale identity killed process")
	}
	if e := SignalProcess(p, syscall.SIGTERM); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e := WaitProcess(ctx, p); e != nil {
		t.Fatal(e)
	}
}
func TestReadProcessCapturesKernelIdentity(t *testing.T) {
	_, p := temporaryProcess(t, "while :; do sleep 1; done")
	if p.PID <= 0 || p.PGID != p.PID || p.StartTicks == 0 || p.BootID == "" || p.Exe == "" || !strings.HasPrefix(p.RunID, "test-") {
		t.Fatalf("incomplete identity %+v", p)
	}
}

func TestDiscoverAndStopChildAfterParentExit(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	c, p := temporaryProcess(t, "trap 'exit 0' TERM; sleep 60 & wait")
	time.Sleep(40 * time.Millisecond)
	p, _ = ReadProcess(c.Process.Pid)
	s.Processes["service"] = p
	s.StopBudgets = map[string]int64{"service": int64(200 * time.Millisecond)}
	_ = SaveState(k.StatePath(), s)
	members, e := DiscoverMembers(s.Processes)
	if e != nil {
		t.Fatal(e)
	}
	if len(members) < 2 {
		t.Fatal("did not discover actual child")
	}
	// Once parent exits, previously recorded children remain independently owned.
	s.Processes = members
	_ = SaveState(k.StatePath(), s)
	if e = SignalProcess(p, syscall.SIGTERM); e != nil {
		t.Fatal(e)
	}
	_ = c.Wait()
	m := NewManager(k)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if e = m.Stop(ctx); e != nil {
		t.Fatal(e)
	}
	for _, member := range members {
		if _, e = ReadProcess(member.PID); !errors.Is(e, os.ErrNotExist) {
			t.Fatalf("member still present %d: %v", member.PID, e)
		}
	}
}
func TestOrphanAndReusedGroupAreNeverSilentlyCleaned(t *testing.T) {
	for _, foreign := range []bool{false, true} {
		t.Run(fmt.Sprint(foreign), func(t *testing.T) {
			k := testKey(t)
			s := initialState(t, k)
			script := "trap 'exit 0' TERM; sleep 60 & wait"
			if foreign {
				script = "trap 'exit 0' TERM; env " + RunIDEnv + "=foreign sleep 60 & wait"
			}
			c, p := temporaryProcess(t, script)
			time.Sleep(40 * time.Millisecond)
			p, _ = ReadProcess(c.Process.Pid)
			// Capture child's actual identity for test cleanup without relying on group kills.
			b, e := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", p.PID, p.PID))
			if e != nil {
				t.Fatal(e)
			}
			pid, e := strconv.Atoi(strings.Fields(string(b))[0])
			if e != nil {
				t.Fatal(e)
			}
			child, e := ReadProcess(pid)
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { _ = SignalProcess(child, syscall.SIGKILL) })
			s.Processes["service"] = p
			_ = SaveState(k.StatePath(), s)
			if !foreign {
				_ = SignalProcess(p, syscall.SIGTERM)
				_ = c.Wait()
			}
			m := NewManager(k)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if e = m.Stop(ctx); e == nil {
				t.Fatal("silently accepted unverified member")
			}
			if _, e = ReadProcess(child.PID); e != nil {
				t.Fatal("killed unverified child")
			}
			final, _ := m.Status()
			if final.Phase == "stopped" || len(final.Failures) == 0 {
				t.Fatal("orphan evidence lost")
			}
		})
	}
}

func TestDiscoversDescendantThatCreatesItsOwnProcessGroup(t *testing.T) {
	c, p := temporaryProcess(t, "setsid sleep 60 & wait")
	time.Sleep(40 * time.Millisecond)
	p, _ = ReadProcess(c.Process.Pid)
	b, e := os.ReadFile(fmt.Sprintf("/proc/%d/task/%d/children", p.PID, p.PID))
	if e != nil {
		t.Fatal(e)
	}
	pid, e := strconv.Atoi(strings.Fields(string(b))[0])
	if e != nil {
		t.Fatal(e)
	}
	child, e := ReadProcess(pid)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = SignalProcess(child, syscall.SIGKILL) })
	if child.PGID == p.PGID {
		t.Fatal("fixture did not create a separate process group")
	}
	members, e := DiscoverMembers(map[string]ProcessIdentity{"root": p})
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, m := range members {
		if m.PID == child.PID {
			found = true
		}
	}
	if !found {
		t.Fatal("missed child that left parent process group")
	}
}
