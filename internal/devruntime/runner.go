package devruntime

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type Manager struct {
	Key    InstanceKey
	Docker Docker
	Signal func(ProcessIdentity, syscall.Signal) error
}

func NewManager(k InstanceKey) *Manager {
	return &Manager{Key: k, Docker: Docker{Exec: command}, Signal: SignalProcess}
}
func (m *Manager) Status() (State, error) { return LoadState(m.Key.StatePath()) }

// Update holds the instance lock only for a state read-modify-write. Callbacks
// must not perform builds, network IO, process waits or Docker commands.
func (m *Manager) Update(fn func(*State) error) error {
	return withLock(m.Key, func() error {
		s, e := LoadState(m.Key.StatePath())
		if e != nil {
			return e
		}
		if e = fn(&s); e != nil {
			return e
		}
		return saveState(m.Key.StatePath(), s)
	})
}
func NewRunID() string { return uuid.NewString() }

// Begin must be called by the foreground supervisor after exec with RunIDEnv.
// os.Setenv in an already running process does not update Linux /proc/environ.
func (m *Manager) Begin(services []string, mode string) (State, error) {
	var result State
	if e := CheckPlatform(); e != nil {
		return result, e
	}
	if mode != "managed" && mode != "external" {
		return result, errors.New("invalid database mode")
	}
	self, e := ReadProcess(os.Getpid())
	if e != nil {
		return result, e
	}
	if self.RunID == "" {
		return result, errors.New("supervisor requires run marker at exec time")
	}
	e = withLock(m.Key, func() error {
		old, e := LoadState(m.Key.StatePath())
		if e != nil && !errors.Is(e, os.ErrNotExist) {
			return e
		}
		if e == nil {
			if old.Supervisor.PID > 0 {
				p, err := ReadProcess(old.Supervisor.PID)
				if err == nil && SameProcess(old.Supervisor, p) {
					return errors.New("instance already has a live supervisor")
				}
				if err != nil && !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
			if old.Phase != "stopped" {
				return errors.New("previous instance requires stop and recovery")
			}
			if old.DBMode != mode {
				return errors.New("database mode cannot change in place")
			}
			if len(old.Intents) != 0 {
				return errors.New("instance has unresolved resource creation intents")
			}
			for _, p := range old.Processes {
				now, err := ReadProcess(p.PID)
				if err == nil {
					if SameProcess(p, now) {
						return errors.New("instance still has live processes")
					}
					return ErrIdentityMismatch
				}
				if !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
		}
		result = State{Version: StateVersion, Key: m.Key, RunID: self.RunID, Phase: "starting", DBMode: mode, Supervisor: self, Processes: map[string]ProcessIdentity{}, Resources: old.Resources, InitializationMode: old.InitializationMode, ConfigFingerprints: old.ConfigFingerprints, Services: append([]string(nil), services...), Logs: map[string]string{}, Endpoints: map[string]string{}, ExitCodes: map[string]int{}, StopBudgets: map[string]int64{}}
		return saveState(m.Key.StatePath(), result)
	})
	return result, e
}

// Spawn starts an already built executable, with an independent process group.
// The caller owns cmd.Wait, so the foreground supervisor reaps its own children.
func (m *Manager) Spawn(name string, cmd *exec.Cmd, budget time.Duration) (ProcessIdentity, error) {
	var p ProcessIdentity
	e := m.Update(func(s *State) error {
		if s.Phase != "starting" && s.Phase != "running" {
			return errors.New("instance is stopping")
		}
		if _, ok := s.Processes[name]; ok {
			return errors.New("service already registered")
		}
		if budget <= 0 {
			return errors.New("service stop budget must be positive")
		}
		if cmd.Env == nil {
			return errors.New("child requires explicit environment whitelist")
		}
		for _, entry := range cmd.Env {
			if len(entry) >= len(RunIDEnv)+1 && entry[:len(RunIDEnv)+1] == RunIDEnv+"=" {
				return errors.New("caller must not supply child run marker")
			}
		}
		cmd.Env = append(cmd.Env, RunIDEnv+"="+s.RunID)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if e := cmd.Start(); e != nil {
			return e
		}

		expected, e := filepath.EvalSymlinks(cmd.Path)
		if e == nil {
			expected, e = filepath.Abs(expected)
		}
		deadline := time.Now().Add(250 * time.Millisecond)
		for e == nil {
			p, e = ReadProcess(cmd.Process.Pid)
			if e == nil && p.RunID == s.RunID && p.PGID == p.PID && p.Exe == expected {
				break
			}
			if !time.Now().Before(deadline) || processExited(cmd.Process.Pid) {
				if e == nil {
					e = errors.New("child exec identity could not be established")
				}
				break
			}
			e = nil
			time.Sleep(time.Millisecond)
		}
		if e != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			p = ProcessIdentity{}
			return e
		}

		if s.Processes == nil {
			s.Processes = map[string]ProcessIdentity{}
		}
		s.Processes[name] = p
		if s.StopBudgets == nil {
			s.StopBudgets = map[string]int64{}
		}
		s.StopBudgets[name] = int64(budget)
		return nil
	})
	if e != nil && p.PID > 0 {
		_ = SignalProcess(p, syscall.SIGKILL)
		_ = cmd.Wait()
	}
	return p, e
}
func (m *Manager) RecordExit(name string, code int) error {
	return m.Update(func(s *State) error {
		if s.ExitCodes == nil {
			s.ExitCodes = map[string]int{}
		}
		s.ExitCodes[name] = code
		return nil
	})
}
func Stop(ctx context.Context, k InstanceKey) error  { return NewManager(k).Stop(ctx) }
func Reset(ctx context.Context, k InstanceKey) error { return NewManager(k).Reset(ctx) }
func (m *Manager) Stop(ctx context.Context) error {
	var snapshot State
	if e := m.Update(func(s *State) error {
		if s.Phase == "resetting" {
			return errors.New("instance reset is in progress")
		}
		s.Phase = "stopping"
		s.Failures = nil
		snapshot = *s
		return nil
	}); e != nil {
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		return e
	}
	var failures []error
	// A remote stop requests supervisor cancellation through its verified pidfd.
	// Never wait on ourselves: the caller continues to reap and write exit codes.
	if snapshot.Supervisor.PID > 0 && snapshot.Supervisor.PID != os.Getpid() {
		now, err := VerifyProcess(snapshot.Supervisor)
		if err == nil {
			if !SameProcess(snapshot.Supervisor, now) {
				failures = append(failures, ErrIdentityMismatch)
			} else if err = m.Signal(snapshot.Supervisor, syscall.SIGTERM); err != nil {
				failures = append(failures, err)
			}
			if err == nil {
				budget := 30 * time.Second
				for _, nanos := range snapshot.StopBudgets {
					if time.Duration(nanos) > budget {
						budget = time.Duration(nanos)
					}
				}
				wait, cancel := context.WithTimeout(ctx, budget+42*time.Second)
				err = WaitProcess(wait, snapshot.Supervisor)
				cancel()
				if err != nil {
					failures = append(failures, err)
				}
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			failures = append(failures, err)
		}
	}
	// The supervisor may have registered its last create result while exiting.
	if latest, err := m.Status(); err == nil {
		if latest.RunID != snapshot.RunID {
			return errors.New("run changed during stop")
		}
		snapshot = latest
	} else {
		failures = append(failures, err)
	}
	members, e := DiscoverMembers(snapshot.Processes, snapshot.Supervisor)
	if e != nil {
		failures = append(failures, e)
	}
	if e = m.Update(func(s *State) error {
		if s.RunID != snapshot.RunID {
			return errors.New("run changed during stop")
		}
		s.Processes = members
		s.Phase = "stopping"
		return nil
	}); e != nil {
		return e
	}
	type stopping struct {
		identity ProcessIdentity
		deadline time.Time
	}
	var pending []stopping
	for name, p := range members {
		current, e := VerifyProcess(p)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			failures = append(failures, e)
			continue
		}
		if !SameProcess(p, current) {
			failures = append(failures, fmt.Errorf("%s: %w", name, ErrIdentityMismatch))
			continue
		}
		budget := time.Duration(snapshot.StopBudgets[name])
		if budget <= 0 {
			budget = 30 * time.Second
			for rootName, root := range snapshot.Processes {
				if root.PGID == p.PGID && snapshot.StopBudgets[rootName] > 0 {
					budget = time.Duration(snapshot.StopBudgets[rootName])
					break
				}
			}
		}
		deadline := time.Now().Add(budget)
		if e = m.Signal(p, syscall.SIGTERM); e != nil {
			failures = append(failures, fmt.Errorf("signal %s: %w", name, e))
			continue
		}
		pending = append(pending, stopping{p, deadline})
	}
	for _, p := range pending {
		wait, cancel := context.WithDeadline(ctx, p.deadline)
		e := WaitProcess(wait, p.identity)
		cancel()
		if errors.Is(e, context.DeadlineExceeded) {
			if e = m.Signal(p.identity, syscall.SIGKILL); e == nil {
				reap, cancel := context.WithTimeout(ctx, time.Second)
				e = WaitProcess(reap, p.identity)
				cancel()
			}
		}
		if e != nil {
			failures = append(failures, e)
		}
	}
	// Recheck groups after shutdown to detect forks that escaped the first snapshot.
	if _, err := DiscoverMembers(members, snapshot.Supervisor); err != nil {
		failures = append(failures, err)
	}
	if len(failures) == 0 {
		if e = m.Recover(ctx); e != nil {
			failures = append(failures, e)
		}
	}
	if len(failures) == 0 {
		latest, err := m.Status()
		if err != nil {
			return err
		}
		cleanup, cancel := context.WithTimeout(ctx, 40*time.Second)
		for _, r := range latest.Resources {
			if !r.Owned || r.Kind != "container" {
				continue
			}
			if e = m.Docker.inspect(cleanup, m.Key, r); e != nil {
				failures = append(failures, e)
				continue
			}
			if _, e = m.Docker.Exec(cleanup, "docker", "container", "stop", "--time", "30", r.ID); e != nil {
				failures = append(failures, e)
			}
		}
		cancel()
	}

	e = m.Update(func(s *State) error {
		if s.RunID != snapshot.RunID {
			return errors.New("run changed during stop")
		}
		s.Failures = nil
		for _, f := range failures {
			s.Failures = append(s.Failures, f.Error())
		}
		if len(failures) == 0 {
			s.Phase = "stopped"
		} else {
			s.Phase = "cleanup-failed"
		}
		return nil
	})
	return errors.Join(append(failures, e)...)
}
func (m *Manager) Reset(ctx context.Context) error {
	var snapshot State
	e := m.Update(func(s *State) error {
		if s.Phase != "stopped" {
			return errors.New("reset requires a stopped instance")
		}
		if s.DBMode != "managed" {
			return errors.New("external database cannot be reset")
		}
		if len(s.Intents) > 0 {
			return errors.New("reset requires recovered creation intents")
		}
		snapshot = *s
		s.Phase = "resetting"
		return nil
	})
	if e != nil {
		return e
	}
	var failures []error
	records := make(map[string]ProcessIdentity, len(snapshot.Processes)+1)
	for k, p := range snapshot.Processes {
		records[k] = p
	}
	if snapshot.Supervisor.PID > 0 {
		records["supervisor"] = snapshot.Supervisor
	}
	members, e := DiscoverMembers(records)
	if e != nil {
		failures = append(failures, e)
	}
	for _, p := range members {
		_, e := ReadProcess(p.PID)
		if e == nil {
			failures = append(failures, fmt.Errorf("process %d is still active", p.PID))
		} else if !errors.Is(e, os.ErrNotExist) {
			failures = append(failures, e)
		}
	}
	if len(failures) == 0 {
		// Containers first: Docker must release volume references before volume rm.
		for _, kind := range []string{"container", "volume"} {
			for _, r := range snapshot.Resources {
				if !r.Owned || r.Kind != kind {
					continue
				}
				if e = m.Docker.inspect(ctx, m.Key, r); e != nil {
					failures = append(failures, e)
					continue
				}
				if _, e = m.Docker.Exec(ctx, "docker", kind, "rm", r.ID); e != nil {
					failures = append(failures, e)
					continue
				}
				if e = m.Update(func(s *State) error {
					for i := len(s.Resources) - 1; i >= 0; i-- {
						if s.Resources[i] == r {
							s.Resources = append(s.Resources[:i], s.Resources[i+1:]...)
						}
					}
					return nil
				}); e != nil {
					failures = append(failures, e)
				}
			}
		}
	}
	e = m.Update(func(s *State) error {
		s.Failures = nil
		for _, f := range failures {
			s.Failures = append(s.Failures, f.Error())
		}
		s.Phase = "stopped"
		return nil
	})
	return errors.Join(append(failures, e)...)
}
