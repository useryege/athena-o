package devruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLockDescriptorsAreNotInheritedByExec(t *testing.T) {
	k := testKey(t)
	var inherited []string
	e := withOperation(k, func() error {
		return withLock(k, func() error {
			c := exec.Command("/bin/sleep", "60")
			if e := c.Start(); e != nil {
				return e
			}
			defer func() { _ = c.Process.Kill(); _ = c.Wait() }()
			entries, e := os.ReadDir(filepath.Join("/proc", strconv.Itoa(c.Process.Pid), "fd"))
			if e != nil {
				return e
			}
			for _, entry := range entries {
				target, _ := os.Readlink(filepath.Join("/proc", strconv.Itoa(c.Process.Pid), "fd", entry.Name()))
				if strings.HasPrefix(target, filepath.Join(k.Checkout, ".run")) {
					inherited = append(inherited, target)
				}
			}
			return nil
		})
	})
	if e != nil {
		t.Fatal(e)
	}
	if len(inherited) > 0 {
		t.Fatalf("exec inherited instance descriptors: %v", inherited)
	}
}

func TestBeginAttemptHelper(t *testing.T) {
	checkout := os.Getenv("DEVRUNTIME_BEGIN_ATTEMPT")
	if checkout == "" {
		return
	}
	k, e := NewInstanceKey(checkout, "test")
	if e != nil {
		t.Fatal(e)
	}
	_, e = NewManager(k).Begin([]string{"test"}, "managed")
	if e == nil {
		fmt.Println("started")
	} else if errors.Is(e, ErrOperationInProgress) {
		fmt.Println("operation-busy")
	} else {
		fmt.Println("unexpected:", e)
	}
}
func TestSlowStopExcludesOtherCleanupAndNewRun(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	s.Resources = []ResourceRef{{Kind: "container", ID: "cid", Name: "db", Namespace: k.Namespace, RunID: s.RunID, Owned: true}}
	if e := SaveState(k.StatePath(), s); e != nil {
		t.Fatal(e)
	}
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	m := NewManager(k)
	calls := make(chan []string, 10)
	m.Docker.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls <- args
		if args[1] == "inspect" {
			select {
			case <-entered:
			default:
				close(entered)
				<-release
			}
			return json.Marshal(map[string]any{"Id": "cid", "Name": "/db", "Config": map[string]any{"Labels": ResourceLabels(k, "container", s.RunID, "db")}})
		}
		return nil, nil
	}
	go func() { done <- m.Stop(context.Background()) }()
	<-entered
	defer func() { close(release); <-done }()
	// A second call must fail before performing its own Docker actions.
	second := make(chan error, 1)
	go func() { second <- m.Stop(context.Background()) }()
	select {
	case e := <-second:
		if e == nil {
			t.Error("concurrent stop entered cleanup")
		}
	case <-time.After(time.Second):
		t.Fatal("concurrent stop must return busy rather than block state access")
	}
	if e := m.Reset(context.Background()); e == nil {
		t.Error("reset entered while prior stop still holds a Docker operation")
	}
	// Even a late state writer cannot release the actual cleanup exclusion.
	if e := m.Update(func(s *State) error { s.Phase = "stopped"; return nil }); e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestBeginAttemptHelper$")
	cmd.Env = append(os.Environ(), "DEVRUNTIME_BEGIN_ATTEMPT="+k.Checkout, RunIDEnv+"="+NewRunID())
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatal(e, string(out))
	}
	if !strings.Contains(string(out), "operation-busy") {
		t.Errorf("Begin did not reject active cleanup: %s", out)
	}
	if len(calls) != 1 {
		t.Fatalf("other lifecycle command performed Docker actions: %d", len(calls))
	}
}

func TestResetResumesAfterDeletionStatePersistenceFailure(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	s.Phase = "stopped"
	s.Resources = []ResourceRef{{Kind: "container", ID: "cid", Name: "db", Namespace: k.Namespace, RunID: s.RunID, Owned: true}, {Kind: "volume", ID: "foreign", Name: "foreign", Owned: false}}
	if e := SaveState(k.StatePath(), s); e != nil {
		t.Fatal(e)
	}
	m := NewManager(k)
	offline := k.Dir() + ".offline"
	deleted := false
	var calls [][]string
	m.Docker.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		switch args[1] {
		case "inspect":
			if deleted {
				return nil, errors.New("container no longer exists")
			}
			return json.Marshal(map[string]any{"Id": "cid", "Name": "/db", "Config": map[string]any{"Labels": ResourceLabels(k, "container", s.RunID, "db")}})
		case "rm":
			if deleted {
				t.Fatal("issued duplicate deletion after absence was proven")
			}
			deleted = true
			return nil, os.Rename(k.Dir(), offline)
		case "ls":
			return []byte("foreign-id\n"), nil
		}
		return nil, fmt.Errorf("unexpected args %v", args)
	}
	if e := m.Reset(context.Background()); e == nil || !deleted {
		t.Fatalf("failure injection did not run: %v", e)
	}
	if e := os.RemoveAll(k.Dir()); e != nil {
		t.Fatal(e)
	}
	if e := os.Rename(offline, k.Dir()); e != nil {
		t.Fatal(e)
	}
	if e := m.Reset(context.Background()); e != nil {
		t.Fatalf("cannot resume after filesystem recovery: %v", e)
	}
	final, e := m.Status()
	if e != nil {
		t.Fatal(e)
	}
	if final.Phase != "stopped" || len(final.Resources) != 1 || final.Resources[0].ID != "foreign" {
		t.Fatalf("bad resumed state: %+v", final)
	}
	for _, args := range calls {
		if args[len(args)-1] == "foreign" {
			t.Fatal("touched borrowed resource")
		}
	}
}

func TestSlowResetExcludesStopAndBegin(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	s.Phase = "stopped"
	s.Resources = []ResourceRef{{Kind: "container", ID: "cid", Name: "db", Namespace: k.Namespace, RunID: s.RunID, Owned: true}}
	if e := SaveState(k.StatePath(), s); e != nil {
		t.Fatal(e)
	}
	m := NewManager(k)
	entered := make(chan struct{})
	release := make(chan struct{})
	done := make(chan error, 1)
	m.Docker.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[1] == "inspect" {
			return json.Marshal(map[string]any{"Id": "cid", "Name": "/db", "Config": map[string]any{"Labels": ResourceLabels(k, "container", s.RunID, "db")}})
		}
		if args[1] == "rm" {
			close(entered)
			<-release
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected Docker argv %v", args)
	}
	go func() { done <- m.Reset(context.Background()) }()
	<-entered
	if e := m.Stop(context.Background()); !errors.Is(e, ErrOperationInProgress) {
		t.Errorf("Stop did not reject active reset: %v", e)
	}
	if _, e := m.Begin(nil, "managed"); !errors.Is(e, ErrOperationInProgress) {
		t.Errorf("Begin did not reject active reset: %v", e)
	}
	if e := m.RecordExit("completed", 0); e != nil {
		t.Errorf("state lock blocked during deletion: %v", e)
	}
	state, e := m.Status()
	if e != nil || state.Phase != "resetting" || len(state.Deletions) != 1 {
		t.Errorf("lost active deletion state: %+v %v", state, e)
	}
	close(release)
	if e := <-done; e != nil {
		t.Fatal(e)
	}
}

func TestDeletionRecoveryRequiresExactIntentAndPositiveAbsence(t *testing.T) {
	for _, scenario := range []string{"no-intent", "list-denied", "still-present", "replaced-volume", "absent"} {
		t.Run(scenario, func(t *testing.T) {
			k := testKey(t)
			s := initialState(t, k)
			s.Phase = "stopped"
			r := ResourceRef{Kind: "volume", ID: "data", Name: "data", Namespace: k.Namespace, RunID: s.RunID, Owned: true}
			s.Resources = []ResourceRef{r}
			if scenario != "no-intent" {
				s.Phase = "resetting"
				s.Cleanup = &CleanupOperation{ID: "reset-op", Kind: "reset", RunID: s.RunID}
				s.Deletions = []DeletionIntent{{OperationID: "reset-op", Resource: r}}
			}
			if e := SaveState(k.StatePath(), s); e != nil {
				t.Fatal(e)
			}
			m := NewManager(k)
			var calls [][]string
			m.Docker.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
				calls = append(calls, args)
				switch args[1] {
				case "inspect":
					if scenario == "replaced-volume" {
						return json.Marshal(map[string]any{"Name": "data", "Labels": ResourceLabels(k, "volume", "foreign-run", "data")})
					}
					return nil, errors.New("inspect denied or lost")
				case "ls":
					if scenario == "list-denied" {
						return nil, errors.New("daemon unavailable")
					}
					if scenario == "still-present" || scenario == "replaced-volume" {
						return []byte("data\n"), nil
					}
					return nil, nil
				default:
					t.Fatalf("recovery mutated unverified resource: %v", args)
					return nil, nil
				}
			}
			e := m.Reset(context.Background())
			final, _ := m.Status()
			if scenario == "absent" {
				if e != nil || len(final.Resources) != 0 || len(final.Deletions) != 0 || final.Cleanup != nil || final.Phase != "stopped" {
					t.Fatalf("positive absence did not complete recovery: %v %+v", e, final)
				}
			} else {
				if e == nil || len(final.Resources) != 1 {
					t.Fatalf("absence inferred without evidence: %v %+v", e, final)
				}
			}
			if scenario == "no-intent" && len(calls) != 1 {
				t.Fatalf("queried absence without deletion authorization: %v", calls)
			}
		})
	}
}
func TestSaveSecretCannotReplaceOperationLockName(t *testing.T) {
	if e := NewManager(testKey(t)).SaveSecret("operation.lock", []byte("secret")); e == nil {
		t.Fatal("accepted lifecycle lock filename")
	}
}

func TestOtherLifecycleCommandsCannotDiscardDeletionJournal(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	s.Phase = "stopped"
	s.Cleanup = &CleanupOperation{ID: "reset-op", Kind: "reset", RunID: s.RunID}
	r := ResourceRef{Kind: "volume", ID: "data", Name: "data", Namespace: k.Namespace, RunID: s.RunID, Owned: true}
	s.Resources = []ResourceRef{r}
	s.Deletions = []DeletionIntent{{OperationID: "reset-op", Resource: r}}
	if e := SaveState(k.StatePath(), s); e != nil {
		t.Fatal(e)
	}
	m := NewManager(k)
	if e := m.Stop(context.Background()); e == nil {
		t.Error("Stop accepted an unfinished reset journal")
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestBeginAttemptHelper$")
	cmd.Env = append(os.Environ(), "DEVRUNTIME_BEGIN_ATTEMPT="+k.Checkout, RunIDEnv+"="+NewRunID())
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatal(e, string(out))
	}
	if strings.Contains(string(out), "started") {
		t.Error("Begin discarded pending deletion authorization")
	}
	final, e := m.Status()
	if e != nil || len(final.Deletions) != 1 || final.Cleanup == nil {
		t.Errorf("lost deletion journal: %+v %v", final, e)
	}
}

func TestResetCrashHelper(t *testing.T) {
	checkout := os.Getenv("DEVRUNTIME_RESET_CRASH")
	if checkout == "" {
		return
	}
	k, e := NewInstanceKey(checkout, "test")
	if e != nil {
		t.Fatal(e)
	}
	m := NewManager(k)
	state, e := m.Status()
	if e != nil {
		t.Fatal(e)
	}
	m.Docker.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		if args[1] == "inspect" {
			return json.Marshal(map[string]any{"Name": "data", "Labels": ResourceLabels(k, "volume", state.RunID, "data")})
		}
		if args[1] == "rm" {
			if e := os.Remove(filepath.Join(checkout, "docker-object")); e != nil {
				t.Fatal(e)
			}
			os.Exit(91)
		}
		return nil, fmt.Errorf("unexpected args %v", args)
	}
	if e = m.Reset(context.Background()); e != nil {
		t.Fatal(e)
	}
	t.Fatal("fixture did not crash during deletion")
}
func TestResetRecoversAfterDeletingProcessCrashes(t *testing.T) {
	k := testKey(t)
	s := initialState(t, k)
	s.Phase = "stopped"
	s.Resources = []ResourceRef{{Kind: "volume", ID: "data", Name: "data", Namespace: k.Namespace, RunID: s.RunID, Owned: true}, {Kind: "volume", ID: "borrowed", Name: "borrowed", Owned: false}}
	if e := SaveState(k.StatePath(), s); e != nil {
		t.Fatal(e)
	}
	marker := filepath.Join(k.Checkout, "docker-object")
	if e := os.WriteFile(marker, []byte("owned volume exists"), 0600); e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestResetCrashHelper$")
	cmd.Env = append(os.Environ(), "DEVRUNTIME_RESET_CRASH="+k.Checkout)
	out, e := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(e, &exitErr) || exitErr.ExitCode() != 91 {
		t.Fatalf("expected deletion crash, got %v: %s", e, out)
	}
	if _, e = os.Stat(marker); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("fixture did not perform deletion before crash")
	}
	m := NewManager(k)
	after, e := m.Status()
	if e != nil || after.Phase != "resetting" || len(after.Deletions) != 1 {
		t.Fatalf("lost crash journal: %+v %v", after, e)
	}
	var calls [][]string
	m.Docker.Exec = func(_ context.Context, _ string, args ...string) ([]byte, error) {
		calls = append(calls, args)
		switch args[1] {
		case "inspect":
			return nil, errors.New("resource absent")
		case "ls":
			return []byte("borrowed\n"), nil
		default:
			return nil, fmt.Errorf("unexpected mutation after crash: %v", args)
		}
	}
	if e = m.Reset(context.Background()); e != nil {
		t.Fatalf("could not recover released operation lock and deletion journal: %v", e)
	}
	after, e = m.Status()
	if e != nil || after.Phase != "stopped" || len(after.Deletions) != 0 || after.Cleanup != nil || len(after.Resources) != 1 || after.Resources[0].ID != "borrowed" {
		t.Fatalf("bad recovered state %+v %v", after, e)
	}
	if len(calls) != 2 {
		t.Fatalf("unexpected recovery commands: %v", calls)
	}
}
