package devruntime

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// A stable shell remains the registered ancestry root until the actual tool has
// exited. The gate prevents the tool from starting before Spawn persisted this
// root. TERM/INT interrupt wait, but do not make the root abandon its child;
// Manager.Stop signals every verified descendant and waits before infrastructure.
const helperProgram = `
IFS= read -r gate || exit 125
[[ "$gate" = start ]] || exit 125
trap ':' INT TERM
"$@" &
child=$!
while :; do
  completed=''
  wait -f -n -p completed "$child"
  code=$?
  if [[ -n "$completed" ]]; then exit "$code"; fi
  if [[ "$code" = 127 ]]; then wait "$child"; exit "$?"; fi
done
`

// RunHelper runs one non-interactive startup tool under the instance's existing
// supervisor. The command must have an explicit environment and no secrets in
// argv. Successful and failed tool identities and compiler logs remain in state.
// Cancellation stops the startup batch through the same ownership protocol.
func (m *Manager) RunHelper(ctx context.Context, name string, command *exec.Cmd, budget time.Duration) error {
	if !instanceName.MatchString(name) || command.Env == nil || command.Path == "" || budget <= 0 {
		return errors.New("invalid startup helper configuration")
	}
	state, err := m.Status()
	if err != nil {
		return err
	}
	if state.Phase != "starting" {
		return errors.New("startup helper requires a starting instance")
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	id := NewRunID()
	label := "helper:" + name + ":" + id
	logPath := filepath.Join(m.Key.Dir(), "helper-"+name+"-"+id+".log")
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer log.Close()
	if err = m.Update(func(s *State) error {
		if s.RunID != state.RunID {
			return errors.New("run changed before helper startup")
		}
		if s.Logs == nil {
			s.Logs = map[string]string{}
		}
		s.Logs[label] = logPath
		return nil
	}); err != nil {
		return err
	}
	gate, writer, err := os.Pipe()
	if err != nil {
		return err
	}
	defer gate.Close()
	defer writer.Close()
	shell, err := exec.LookPath("bash")
	if err != nil {
		return err
	}
	shell, err = filepath.EvalSymlinks(shell)
	if err != nil {
		return err
	}
	args := []string{"--noprofile", "--norc", "-c", helperProgram, "athena-startup-helper", command.Path}
	args = append(args, command.Args[1:]...)
	child := exec.Command(shell, args...)
	child.Env = append([]string{}, command.Env...)
	child.Dir = command.Dir
	child.Stdin = gate
	child.Stdout = log
	child.Stderr = log
	if _, err = m.Spawn(label, child, budget); err != nil {
		return fmt.Errorf("start %s; log %s: %w", name, logPath, err)
	}
	done := make(chan error, 1)
	go func() {
		err := child.Wait()
		code := 0
		if err != nil {
			code = child.ProcessState.ExitCode()
		}
		recordErr := m.RecordExit(label, code)
		done <- errors.Join(err, recordErr)
	}()
	// Spawn's durable identity is now sufficient for recovery even if this
	// supervisor disappears while the target command is using the database.
	if _, err = writer.Write([]byte("start\n")); err != nil {
		writer.Close()
		return errors.Join(err, <-done)
	}
	writer.Close()
	gate.Close()
	select {
	case err = <-done:
		if err != nil {
			return fmt.Errorf("%s failed; log %s: %w", name, logPath, err)
		}
		return nil
	case <-ctx.Done():
		cleanup, cancel := context.WithTimeout(context.Background(), budget+45*time.Second)
		defer cancel()
		stopErr := m.Stop(cleanup)
		select {
		case waitErr := <-done:
			return errors.Join(ctx.Err(), stopErr, waitErr)
		case <-cleanup.Done():
			return errors.Join(ctx.Err(), stopErr, errors.New("startup helper did not exit within cleanup budget"))
		}
	}
}

func selectedServiceExited(s State) bool {
	for _, name := range s.Services {
		if _, ok := s.ExitCodes[name]; ok {
			return true
		}
	}
	return false
}
