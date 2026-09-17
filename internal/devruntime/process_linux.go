//go:build linux

package devruntime

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

const RunIDEnv = "ATHENA_LOCAL_RUNTIME_RUN_ID"

var ErrIdentityMismatch = errors.New("process identity mismatch")

func SameProcess(a, b ProcessIdentity) bool {
	return a.PID > 0 && a.PGID > 0 && a.StartTicks > 0 && a.BootID != "" && a.Exe != "" && a.RunID != "" && a == b
}

type procInfo struct {
	identity ProcessIdentity
	parent   int
	state    string
}

func readProc(pid int) (p procInfo, err error) {
	defer func() {
		if errors.Is(err, syscall.ESRCH) {
			err = os.ErrNotExist
		}
	}()
	root := filepath.Join("/proc", strconv.Itoa(pid))
	b, e := os.ReadFile(filepath.Join(root, "stat"))
	if e != nil {
		return p, e
	}
	end := bytes.LastIndexByte(b, ')')
	if end < 0 {
		return p, errors.New("invalid proc stat")
	}
	fields := strings.Fields(string(b[end+1:]))
	if len(fields) < 20 {
		return p, errors.New("short proc stat")
	}
	p.state = fields[0]
	p.parent, e = strconv.Atoi(fields[1])
	if e != nil {
		return p, e
	}
	p.identity.PID = pid
	p.identity.PGID, e = strconv.Atoi(fields[2])
	if e != nil {
		return p, e
	}
	p.identity.StartTicks, e = strconv.ParseUint(fields[19], 10, 64)
	if e != nil {
		return p, e
	}
	if p.state == "Z" || p.state == "X" {
		return p, os.ErrNotExist
	}
	b, e = os.ReadFile("/proc/sys/kernel/random/boot_id")
	if e != nil {
		return p, e
	}
	p.identity.BootID = strings.TrimSpace(string(b))
	p.identity.Exe, e = os.Readlink(filepath.Join(root, "exe"))
	if e != nil {
		return p, e
	}
	b, e = os.ReadFile(filepath.Join(root, "environ"))
	if e != nil {
		return p, e
	}
	for _, entry := range bytes.Split(b, []byte{0}) {
		if bytes.HasPrefix(entry, []byte(RunIDEnv+"=")) {
			p.identity.RunID = string(entry[len(RunIDEnv)+1:])
		}
	}
	return p, nil
}
func ReadProcess(pid int) (ProcessIdentity, error) { p, e := readProc(pid); return p.identity, e }

// A pidfd is opened BEFORE the second identity read. Even if the PID is recycled
// afterwards, the signal and exit wait refer only to the pinned kernel task.
func openVerified(ctx context.Context, record ProcessIdentity) (int, error) {
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return -1, err
	}
	if record.BootID != "" && record.BootID != strings.TrimSpace(string(boot)) {
		return -1, os.ErrNotExist
	}
	fd, e := unix.PidfdOpen(record.PID, 0)
	if e != nil {
		if errors.Is(e, unix.ESRCH) {
			return -1, os.ErrNotExist
		}
		return -1, e
	}
	now, e := ReadProcess(record.PID)
	if e != nil || !SameProcess(record, now) {
		// A thread-group leader can lose /proc metadata while other threads are
		// still exiting. Keep the same pidfd pinned and wait briefly for positive
		// group-exit evidence. Missing metadata alone never authorizes a signal.
		// Bound this uncommon path independently and honor WaitProcess's deadline.
		var exited bool
		var pollErr error
		if errors.Is(e, os.ErrNotExist) {
			settle, cancel := context.WithTimeout(ctx, 250*time.Millisecond)
			exited, pollErr = waitPidfdExit(settle, fd)
			cancel()
			if errors.Is(pollErr, context.DeadlineExceeded) && ctx.Err() == nil {
				pollErr = nil // Still live: preserve the identity-unavailable error.
			}
		} else {
			exited, pollErr = pidfdExited(fd, 10)
		}
		_ = unix.Close(fd)
		if pollErr != nil {
			return -1, pollErr
		}
		if exited {
			return -1, os.ErrNotExist
		}
		if e != nil {
			if errors.Is(e, os.ErrNotExist) {
				return -1, fmt.Errorf("live pid %d has unavailable identity: %s", record.PID, e)
			}
			return -1, e
		}
		return -1, fmt.Errorf("%w for pid %d", ErrIdentityMismatch, record.PID)
	}
	return fd, nil
}
func pidfdExited(fd, timeout int) (bool, error) {
	events := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	_, e := unix.Poll(events, timeout)
	if errors.Is(e, unix.EINTR) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	return events[0].Revents&(unix.POLLIN|unix.POLLHUP) != 0, nil
}

func waitPidfdExit(ctx context.Context, fd int) (bool, error) {
	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		exited, err := pidfdExited(fd, 10)
		if err != nil || exited {
			return exited, err
		}
	}
}
func processExited(pid int) bool {
	fd, e := unix.PidfdOpen(pid, 0)
	if errors.Is(e, unix.ESRCH) {
		return true
	}
	if e != nil {
		return false
	}
	defer unix.Close(fd)
	exited, e := pidfdExited(fd, 10)
	return e == nil && exited
}
func VerifyProcess(record ProcessIdentity) (ProcessIdentity, error) {
	fd, e := openVerified(context.Background(), record)
	if e != nil {
		return ProcessIdentity{}, e
	}
	_ = unix.Close(fd)
	return record, nil
}
func SignalProcess(record ProcessIdentity, sig syscall.Signal) error {
	fd, e := openVerified(context.Background(), record)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	if e != nil {
		return e
	}
	defer unix.Close(fd)
	e = unix.PidfdSendSignal(fd, unix.Signal(sig), nil, 0)
	if errors.Is(e, unix.ESRCH) {
		return nil
	}
	return e
}
func CheckPlatform() error {
	fd, e := unix.PidfdOpen(os.Getpid(), 0)
	if e != nil {
		return fmt.Errorf("Linux pidfd identity protection unavailable: %w", e)
	}
	defer unix.Close(fd)
	return unix.PidfdSendSignal(fd, 0, nil, 0)
}
func WaitProcess(ctx context.Context, p ProcessIdentity) error {
	fd, e := openVerified(ctx, p)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	if e != nil {
		return e
	}
	defer unix.Close(fd)
	_, e = waitPidfdExit(ctx, fd)
	return e
}

// DiscoverMembers proves parentage before recording new members. A run marker
// narrows the scan but never grants permission to signal an orphan by itself.
// Supervisors are ancestry anchors only, not returned as business processes.
func DiscoverMembers(records map[string]ProcessIdentity, supervisors ...ProcessIdentity) (map[string]ProcessIdentity, error) {
	return discoverMembers(records, supervisors, false)
}
func discoverMembers(records map[string]ProcessIdentity, supervisors []ProcessIdentity, scoped bool) (map[string]ProcessIdentity, error) {
	result := make(map[string]ProcessIdentity, len(records))
	groups := map[int]string{}
	runs := map[string]bool{}
	roots := map[int]ProcessIdentity{}
	excluded := map[int]bool{}
	boot, err := os.ReadFile("/proc/sys/kernel/random/boot_id")
	if err != nil {
		return result, err
	}
	for name, p := range records {
		result[name] = p
		if p.BootID != "" && p.BootID != strings.TrimSpace(string(boot)) {
			continue
		}
		groups[p.PGID] = p.RunID
		if p.RunID != "" {
			runs[p.RunID] = true
		}
	}
	for _, p := range supervisors {
		if p.PID <= 0 || (p.BootID != "" && p.BootID != strings.TrimSpace(string(boot))) {
			continue
		}
		excluded[p.PID] = true
		if p.RunID != "" {
			runs[p.RunID] = true
		}
		now, e := VerifyProcess(p)
		if e == nil && SameProcess(p, now) {
			roots[p.PID] = p
		}
	}
	for _, p := range records {
		now, e := VerifyProcess(p)
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			return result, e
		}
		if !SameProcess(p, now) {
			return result, fmt.Errorf("%w discovering pid %d: recorded=%+v current=%+v", ErrIdentityMismatch, p.PID, p, now)
		}
		roots[p.PID] = p
		// Ancestors are not descendants to be stopped. Validate the kernel chain
		// instead of excluding any same-run PID solely by its numeric value.
		info, e := readProc(p.PID)
		if e != nil {
			continue
		}
		parent := info.parent
		seen := map[int]bool{}
		for parent > 1 && !seen[parent] {
			seen[parent] = true
			up, e := readProc(parent)
			if e != nil || up.identity.RunID != p.RunID {
				break
			}
			excluded[parent] = true
			parent = up.parent
		}
	}
	entries, e := os.ReadDir("/proc")
	if e != nil {
		return result, e
	}
	for _, entry := range entries {
		pid, e := strconv.Atoi(entry.Name())
		if e != nil {
			continue
		}
		p, e := readProc(pid)
		run, inGroup := groups[p.identity.PGID]
		if !inGroup && !runs[p.identity.RunID] {
			continue
		}
		if e != nil {
			if errors.Is(e, os.ErrNotExist) || processExited(pid) {
				continue
			}
			return result, fmt.Errorf("cannot verify group member %d: %w", pid, e)
		}
		if _, ok := roots[pid]; ok {
			continue
		}
		if excluded[pid] {
			continue
		}
		if inGroup && p.identity.RunID != run {
			if processExited(pid) {
				continue
			}
			return result, fmt.Errorf("unverified member %d in recorded group %d", pid, p.identity.PGID)
		}
		run = p.identity.RunID
		parent := p.parent
		found := false
		seen := map[int]bool{}
		for parent > 1 && !seen[parent] {
			seen[parent] = true
			if root, ok := roots[parent]; ok {
				now, e := ReadProcess(parent)
				found = e == nil && SameProcess(root, now) && root.RunID == run
				break
			}
			up, e := readProc(parent)
			if e != nil || up.identity.RunID != run {
				break
			}
			parent = up.parent
		}
		if !found {
			if scoped && !inGroup {
				continue
			}
			return result, fmt.Errorf("orphaned member %d has no verified recorded ancestor", pid)
		}
		result["descendant/"+strconv.Itoa(pid)] = p.identity
	}
	return result, nil
}
