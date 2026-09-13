//go:build integration

package devruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/useryege/athena/internal/accountstate/schema"
)

func TestStartupToolSupervisorProcess(t *testing.T) {
	mode := os.Getenv("ATHENA_TASK9_TOOL_SCENARIO")
	if mode == "" {
		return
	}
	var o RunOptions
	if err := json.Unmarshal([]byte(os.Getenv("ATHENA_TASK9_OPTIONS")), &o); err != nil {
		os.Exit(91)
	}
	m := NewManager(o.Key)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	err := func() error {
		if _, err := m.Begin(o.Services, o.DBMode); err != nil {
			return err
		}
		dsn, err := m.preparePostgres(ctx)
		if err != nil {
			return err
		}
		if err = os.WriteFile(filepath.Join(o.Key.Dir(), "test-database-ready"), nil, 0600); err != nil {
			return err
		}
		for {
			if _, err = os.Stat(filepath.Join(o.Key.Dir(), "test-release")); err == nil {
				break
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(10 * time.Millisecond):
			}
		}
		if mode != "migration" {
			if err = os.Setenv("PATH", filepath.Join(o.Key.Dir(), "test-tools")+":"+os.Getenv("PATH")); err != nil {
				return err
			}
		}
		if mode == "build-service" {
			specs, err := ResolveServices([]string{"trader-sync"})
			if err != nil {
				return err
			}
			_, err = buildServices(ctx, o.Key, specs)
			return err
		}
		return runSchema(ctx, o.Key, "up", map[string]string{schema.DSNEnv: dsn})
	}()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(92)
	}
	os.Exit(0)
}

func TestStartupToolsRemainOwnedAfterSupervisorCrash(t *testing.T) {
	for _, mode := range []string{"build-service", "build-schema", "migration"} {
		t.Run(mode, func(t *testing.T) {
			o := runnerOptions(t, "managed", []string{"trader-sync"}, "")
			m := NewManager(o.Key)
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
			defer cancel()
			if err := ensureDir(o.Key); err != nil {
				t.Fatal(err)
			}
			// Real Go is held at its compiler boundary. No fake Docker or database:
			// this fixture gives crash recovery a deterministic live build to find.
			if mode != "migration" {
				dir := filepath.Join(o.Key.Dir(), "test-tools")
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				goPath, err := exec.LookPath("go")
				if err != nil {
					t.Fatal(err)
				}
				hold := filepath.Join(dir, "hold-compiler")
				body := "#!/bin/bash\nif [[ ${2:-} = -V=full ]]; then exec \"$@\"; fi\nprintf '%s' \"$$\" > \"$(dirname \"$0\")/compiler-pid\"\nexec /bin/sleep 120\n"
				if err = os.WriteFile(hold, []byte(body), 0700); err != nil {
					t.Fatal(err)
				}
				body = "#!/bin/bash\nexec " + strconv.Quote(goPath) + " build -a -p=1 -toolexec=" + strconv.Quote(hold) + " \"${@:2}\"\n"
				if err = os.WriteFile(filepath.Join(dir, "go"), []byte(body), 0700); err != nil {
					t.Fatal(err)
				}
			}
			data, _ := json.Marshal(o)
			cmd := exec.Command(os.Args[0], "-test.run=^TestStartupToolSupervisorProcess$")
			cmd.Env = append(EnvironmentFor(environmentMap(os.Environ()), []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN"}), RunIDEnv+"="+NewRunID(), "ATHENA_TASK9_TOOL_SCENARIO="+mode, "ATHENA_TASK9_OPTIONS="+string(data))
			cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
			log, err := os.CreateTemp(t.TempDir(), "supervisor-")
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
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			waitFor := func(what string, predicate func() bool) {
				t.Helper()
				deadline := time.Now().Add(60 * time.Second)
				for time.Now().Before(deadline) {
					if predicate() {
						return
					}
					select {
					case err := <-done:
						t.Fatalf("supervisor exited during %s: %v; log=%s", what, err, log.Name())
					case <-time.After(20 * time.Millisecond):
					}
				}
				t.Fatal("timeout", what, log.Name())
			}
			waitFor("database startup", func() bool { _, err := os.Stat(filepath.Join(o.Key.Dir(), "test-database-ready")); return err == nil })
			s, err := m.Status()
			if err != nil {
				t.Fatal(err)
			}
			dsn, err := m.currentManagedDSN(ctx, s)
			if err != nil {
				t.Fatal(err)
			}
			conn, err := pgx.Connect(ctx, dsn)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close(context.Background())
			if mode == "migration" {
				if _, err = conn.Exec(ctx, "SELECT pg_advisory_lock(hashtext('athena:postgres:schema:v1')::bigint)"); err != nil {
					t.Fatal(err)
				}
			}
			if err = os.WriteFile(filepath.Join(o.Key.Dir(), "test-release"), nil, 0600); err != nil {
				t.Fatal(err)
			}
			var members map[string]ProcessIdentity
			waitFor("blocked owned tool", func() bool {
				s, err = m.Status()
				if err != nil {
					return false
				}
				if mode == "migration" {
					var blocked bool
					if conn.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event='advisory' AND pid<>pg_backend_pid())").Scan(&blocked) != nil || !blocked {
						return false
					}
				} else {
					if _, err := os.Stat(filepath.Join(o.Key.Dir(), "test-tools/compiler-pid")); err != nil {
						return false
					}
				}
				members, err = DiscoverMembers(s.Processes, s.Supervisor)
				return err == nil && len(members) > len(s.Processes)
			})
			for _, p := range members {
				if p.RunID != s.RunID {
					t.Fatal("tool lost run marker")
				}
			}
			if err = SignalProcess(s.Supervisor, syscall.SIGKILL); err != nil {
				t.Fatal(err)
			}
			<-done
			// Reconstruct from durable state after parent death, not a pre-crash
			// in-memory descendant snapshot.
			crashed, err := m.Status()
			if err != nil {
				t.Fatal(err)
			}
			recovered, err := DiscoverMembers(crashed.Processes, crashed.Supervisor)
			if err != nil {
				t.Fatal(err)
			}
			if len(recovered) <= len(crashed.Processes) {
				t.Fatal("active tool lost its persisted ancestry after crash")
			}
			stoppedDB := false
			m.Docker.Exec = func(callCtx context.Context, name string, args ...string) ([]byte, error) {
				if name == "docker" && len(args) > 1 && args[0] == "container" && args[1] == "stop" {
					for _, p := range recovered {
						if _, err := VerifyProcess(p); !os.IsNotExist(err) {
							return nil, fmt.Errorf("database cleanup preceded tool exit pid=%d: %v", p.PID, err)
						}
					}
					stoppedDB = true
				}
				return command(callCtx, name, args...)
			}
			if err = m.Stop(ctx); err != nil {
				t.Fatal(err)
			}
			if !stoppedDB {
				t.Fatal("owned PostgreSQL was not stopped")
			}
			for name, path := range crashed.Logs {
				if strings.HasPrefix(name, "helper:") {
					if _, err = os.Stat(path); err != nil {
						t.Fatal("helper log lost", err)
					}
				}
			}
			t.Logf("scenario=%s namespace=%s supervisor=%d persisted=%d recovered=%d; helpers exited before real Docker stop", mode, o.Key.Namespace, crashed.Supervisor.PID, len(crashed.Processes), len(recovered))
		})
	}
}
