package devruntime

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	e := withOperation(m.Key, func() error { var e error; result, e = m.begin(services, mode); return e })
	return result, e
}
func (m *Manager) begin(services []string, mode string) (State, error) {
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
			if old.Cleanup != nil || len(old.Deletions) > 0 {
				return errors.New("instance has an unfinished reset; retry reset before starting")
			}
			if len(old.Intents) != 0 {
				return errors.New("instance has unresolved resource creation intents")
			}
			for _, p := range old.Processes {
				now, err := VerifyProcess(p)
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
		expected := (strings.HasPrefix(name, "helper:") && code == 0) || s.Phase == "stopping" || s.Phase == "stopped" || s.StopRequested[name]
		s.Exits = append(s.Exits, ExitEvent{Service: name, Code: code, Expected: expected, At: time.Now()})
		for _, selected := range s.Services {
			if selected == name {
				if s.Health == nil {
					s.Health = map[string]string{}
				}
				s.Health[name] = "exited"
				if s.Probes == nil {
					s.Probes = map[string]ProbeResult{}
				}
				s.Probes[name] = ProbeResult{Error: fmt.Sprintf("process exited with code %d", code), At: time.Now()}
				s.FullStackReady = false
				s.SelectedReady = false
				if serviceRegistry()[name].Core {
					s.CoreUsable = false
				}
				if !expected {
					if s.Phase == "running" {
						s.Stage = "runtime-degraded"
						s.StageHistory = append(s.StageHistory, PhaseEvent{Stage: s.Stage, At: time.Now()})
					}
					s.Failures = append(s.Failures, fmt.Sprintf("%s exited unexpectedly with code %d", name, code))
				}
			}
		}
		return nil
	})
}
func Stop(ctx context.Context, k InstanceKey) error  { return NewManager(k).Stop(ctx) }
func Reset(ctx context.Context, k InstanceKey) error { return NewManager(k).Reset(ctx) }
func (m *Manager) Stop(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()
	lease, lockErr := acquireOperation(m.Key)
	if lockErr != nil {
		return lockErr
	}
	defer func() { lease.close() }()
	// A helper and its supervisor share one cleanup deadline. A later recovery
	// after the supervisor has gone may establish a new bounded stop operation.
	reuseDeadline := false
	if previous, err := m.Status(); err == nil {
		if previous.Supervisor.PID == os.Getpid() {
			reuseDeadline = true
		} else if previous.Supervisor.PID > 0 {
			_, err = VerifyProcess(previous.Supervisor)
			reuseDeadline = err == nil
		}
	}
	var snapshot State
	if e := m.Update(func(s *State) error {
		if s.Phase == "resetting" || s.Cleanup != nil || len(s.Deletions) > 0 {
			return errors.New("instance reset is in progress")
		}
		if s.StopDeadline.IsZero() || !reuseDeadline {
			s.StopDeadline, _ = ctx.Deadline()
		}
		s.Phase = "stopping"
		s.FullStackReady = false
		s.CoreUsable = false
		s.SelectedReady = false
		for _, name := range s.Services {
			if s.Health != nil {
				s.Health[name] = "stopping"
			}
			if s.Probes != nil {
				s.Probes[name] = ProbeResult{Error: "instance stopping", At: time.Now()}
			}
		}
		s.StageHistory = append(s.StageHistory, PhaseEvent{Stage: "stopping", At: time.Now()})
		snapshot = *s
		return nil
	}); e != nil {
		if errors.Is(e, os.ErrNotExist) {
			return nil
		}
		return e
	}
	budgetCtx, budgetCancel := context.WithDeadline(ctx, snapshot.StopDeadline)
	defer budgetCancel()
	ctx = budgetCtx
	var failures []error
	// A remote stop requests supervisor cancellation through its verified pidfd.
	// Never wait on ourselves: the caller continues to reap and write exit codes.
	if snapshot.Supervisor.PID > 0 && snapshot.Supervisor.PID != os.Getpid() {
		now, err := VerifyProcess(snapshot.Supervisor)
		if err == nil {
			if !SameProcess(snapshot.Supervisor, now) {
				failures = append(failures, ErrIdentityMismatch)
			} else {
				// The supervisor must acquire this lock for its own cleanup.
				// Keep only the durable stopping phase while waiting for it.
				lease.close()
				if err = m.Signal(snapshot.Supervisor, syscall.SIGTERM); err != nil {
					failures = append(failures, err)
				}
			}
			if err == nil {
				wait, cancel := context.WithCancel(ctx)
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
	if lease.fd < 0 {
		lease, lockErr = acquireOperation(m.Key)
		if lockErr != nil {
			return errors.Join(append(failures, lockErr)...)
		}
	}
	// The supervisor may have registered its last create result while exiting.
	if latest, err := m.Status(); err == nil {
		if latest.RunID != snapshot.RunID {
			return errors.New("run changed during stop")
		}
		if latest.Phase == "resetting" || latest.Cleanup != nil || len(latest.Deletions) > 0 {
			return errors.New("instance reset requires recovery")
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
	for layer := 0; layer <= 2; layer++ {
		group := map[string]ProcessIdentity{}
		for name, p := range members {
			stopLayer := 1
			if spec, ok := serviceRegistry()[name]; ok {
				stopLayer = spec.StopLayer
			} else {
				for rootName, root := range snapshot.Processes {
					if root.PGID == p.PGID {
						if spec, ok := serviceRegistry()[rootName]; ok {
							stopLayer = spec.StopLayer
							break
						}
					}
				}
			}
			if stopLayer == layer {
				group[name] = p
			}
		}
		if err := m.stopMembers(ctx, snapshot, group); err != nil {
			failures = append(failures, err)
			break
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
		outcome := CleanupResult{Service: "instance", At: time.Now()}
		for _, f := range failures {
			s.Failures = append(s.Failures, f.Error())
		}
		if err := errors.Join(failures...); err != nil {
			outcome.Error = err.Error()
		}
		s.CleanupResults = append(s.CleanupResults, outcome)
		if len(failures) == 0 {
			s.Phase = "stopped"
			s.Stage = "stopped"
		} else {
			s.Phase = "cleanup-failed"
		}
		return nil
	})
	return errors.Join(append(failures, e)...)
}
func (m *Manager) Reset(ctx context.Context) error {
	return withOperation(m.Key, func() error { return m.reset(ctx) })
}
func (m *Manager) reset(ctx context.Context) error {
	var snapshot State
	e := m.Update(func(s *State) error {
		if s.Phase != "stopped" && s.Phase != "resetting" {
			return errors.New("reset requires a stopped instance")
		}
		if s.DBMode != "managed" {
			return errors.New("external database cannot be reset")
		}
		if len(s.Intents) > 0 {
			return errors.New("reset requires recovered creation intents")
		}
		if s.Cleanup == nil {
			if s.Phase == "resetting" || len(s.Deletions) > 0 {
				return errors.New("reset has no recoverable operation identity")
			}
			s.Cleanup = &CleanupOperation{ID: NewRunID(), Kind: "reset", RunID: s.RunID}
		}
		if s.Cleanup.ID == "" || s.Cleanup.Kind != "reset" || s.Cleanup.RunID != s.RunID {
			return errors.New("reset operation identity mismatch")
		}
		for _, intent := range s.Deletions {
			if intent.OperationID != s.Cleanup.ID {
				return errors.New("deletion intent operation mismatch")
			}
		}
		s.Phase = "resetting"
		snapshot = *s
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
		_, e := VerifyProcess(p)
		if e == nil {
			failures = append(failures, fmt.Errorf("process %d is still active", p.PID))
		} else if !errors.Is(e, os.ErrNotExist) {
			failures = append(failures, e)
		}
	}
	if len(failures) == 0 {
		// Stop on the first failed destructive step. Further deletion must never
		// continue after the operation can no longer persist its exact progress.
		cleanup, cancel := context.WithTimeout(ctx, 40*time.Second)
	deletion:
		for _, kind := range []string{"container", "volume"} {
			for _, r := range snapshot.Resources {
				if !r.Owned || r.Kind != kind {
					continue
				}
				if e = m.deleteResource(cleanup, snapshot, r); e != nil {
					failures = append(failures, e)
					break deletion
				}
			}
		}
		cancel()
	}
	e = m.updateReset(snapshot, func(s *State) error {
		s.Failures = nil
		for _, failure := range failures {
			s.Failures = append(s.Failures, failure.Error())
		}
		if len(failures) == 0 && len(s.Deletions) > 0 {
			return errors.New("reset has unresolved deletion intents")
		}
		if len(s.Deletions) == 0 {
			s.Phase = "stopped"
			s.Cleanup = nil
		} else {
			s.Phase = "resetting"
		}
		return nil
	})
	return errors.Join(append(failures, e)...)
}
func (m *Manager) updateReset(snapshot State, fn func(*State) error) error {
	return m.Update(func(s *State) error {
		if snapshot.Cleanup == nil || s.Cleanup == nil || *s.Cleanup != *snapshot.Cleanup || s.RunID != snapshot.RunID || s.Phase != "resetting" {
			return errors.New("reset operation changed")
		}
		return fn(s)
	})
}
func (m *Manager) deleteResource(ctx context.Context, snapshot State, r ResourceRef) error {
	if !r.Owned || r.Namespace != m.Key.Namespace || r.RunID == "" || r.Name == "" || !validKind(r.Kind) {
		return errors.New("invalid owned deletion resource")
	}
	intent := DeletionIntent{OperationID: snapshot.Cleanup.ID, Resource: r}
	current, e := m.Status()
	if e != nil {
		return e
	}
	pending := false
	for _, record := range current.Deletions {
		if record == intent {
			pending = true
		}
	}
	if e = m.Docker.inspect(ctx, m.Key, r); e != nil {
		if !pending {
			return e
		}
		absent, checkErr := m.Docker.absent(ctx, r)
		if checkErr != nil {
			return errors.Join(e, checkErr)
		}
		if !absent {
			return e
		}
		return m.finishDeletion(snapshot, intent)
	}
	if pending {
		if e = m.updateReset(snapshot, func(*State) error { return nil }); e != nil {
			return e
		}
	}
	if !pending {
		if e = m.updateReset(snapshot, func(s *State) error { s.Deletions = append(s.Deletions, intent); return nil }); e != nil {
			return e
		}
	}
	if _, e = m.Docker.Exec(ctx, "docker", r.Kind, "rm", r.ID); e != nil {
		return e
	}
	return m.finishDeletion(snapshot, intent)
}
func (m *Manager) finishDeletion(snapshot State, intent DeletionIntent) error {
	return m.updateReset(snapshot, func(s *State) error {
		for i := len(s.Resources) - 1; i >= 0; i-- {
			if s.Resources[i] == intent.Resource {
				s.Resources = append(s.Resources[:i], s.Resources[i+1:]...)
			}
		}
		for i := len(s.Deletions) - 1; i >= 0; i-- {
			if s.Deletions[i] == intent {
				s.Deletions = append(s.Deletions[:i], s.Deletions[i+1:]...)
			}
		}
		return nil
	})
}

// stopMembers signals a layer together, preserving each service's own deadline.
func (m *Manager) stopMembers(ctx context.Context, snapshot State, members map[string]ProcessIdentity) error {
	type pendingStop struct {
		name     string
		identity ProcessIdentity
		deadline time.Time
	}
	var pending []pendingStop
	var failures []error
	for name, p := range members {
		if _, err := VerifyProcess(p); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", name, err))
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
		if end, ok := ctx.Deadline(); ok {
			reserve := time.Until(end) / 10
			if reserve > time.Second {
				reserve = time.Second
			}
			if deadline.After(end.Add(-reserve)) {
				deadline = end.Add(-reserve)
			}
		}
		if err := m.Signal(p, syscall.SIGTERM); err != nil {
			failures = append(failures, fmt.Errorf("signal %s: %w", name, err))
			continue
		}
		pending = append(pending, pendingStop{name, p, deadline})
	}
	for _, p := range pending {
		wait, cancel := context.WithDeadline(ctx, p.deadline)
		err := WaitProcess(wait, p.identity)
		cancel()
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			if err = m.Signal(p.identity, syscall.SIGKILL); err == nil {
				reap, cancel := context.WithTimeout(ctx, time.Second)
				err = WaitProcess(reap, p.identity)
				cancel()
			}
		}
		if err != nil {
			failures = append(failures, fmt.Errorf("wait %s: %w", p.name, err))
		}
		recordErr := m.Update(func(s *State) error {
			result := CleanupResult{Service: p.name, At: time.Now()}
			if err != nil {
				result.Error = err.Error()
			}
			s.CleanupResults = append(s.CleanupResults, result)
			return nil
		})
		if recordErr != nil {
			failures = append(failures, recordErr)
		}
	}
	return errors.Join(failures...)
}

func (m *Manager) stopService(ctx context.Context, name string) error {
	var snapshot State
	if err := m.Update(func(s *State) error {
		if s.StopRequested == nil {
			s.StopRequested = map[string]bool{}
		}
		s.StopRequested[name] = true
		snapshot = *s
		return nil
	}); err != nil {
		return err
	}
	root, ok := snapshot.Processes[name]
	if !ok {
		return nil
	}
	members, err := discoverMembers(map[string]ProcessIdentity{name: root}, nil, true)
	if err != nil {
		return err
	}
	if err = m.Update(func(s *State) error {
		for label, p := range members {
			s.Processes[label] = p
			s.StopBudgets[label] = snapshot.StopBudgets[name]
		}
		return nil
	}); err != nil {
		return err
	}
	if err = m.stopMembers(ctx, snapshot, members); err != nil {
		return err
	}
	// Preserve the same fail-closed rule as whole-instance cleanup when a
	// process forks after discovery and its ancestry can no longer be proved.
	_, err = discoverMembers(members, nil, true)
	return err
}
