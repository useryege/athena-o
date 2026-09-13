// Package devruntime owns only resources recorded for a local checkout instance.
package devruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"syscall"
)

const StateVersion = 1

var instanceName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,62}$`)

type InstanceKey struct{ Checkout, Name, Namespace string }
type ProcessIdentity struct {
	PID, PGID          int
	StartTicks         uint64
	BootID, Exe, RunID string
}
type ResourceRef struct {
	Kind, ID, Name, Namespace, RunID string
	Owned                            bool
}
type CreationIntent struct{ Kind, Name, Namespace, RunID string }
type State struct {
	Version                               int
	Key                                   InstanceKey
	RunID, Phase, DBMode                  string
	Supervisor                            ProcessIdentity
	Processes                             map[string]ProcessIdentity
	Resources                             []ResourceRef
	Intents                               []CreationIntent
	Services                              []string
	Logs, Endpoints                       map[string]string
	ExitCodes                             map[string]int
	BuildFingerprints, ConfigFingerprints map[string]string
	InitializationMode                    string
	Failures                              []string
	StopBudgets                           map[string]int64 // Nanoseconds, supplied by the service registry.
}

func NewInstanceKey(checkout, name string) (InstanceKey, error) {
	if !instanceName.MatchString(name) {
		return InstanceKey{}, errors.New("invalid instance name")
	}
	path, err := filepath.Abs(checkout)
	if err != nil {
		return InstanceKey{}, err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return InstanceKey{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return InstanceKey{}, err
	}
	if !info.IsDir() {
		return InstanceKey{}, errors.New("checkout is not a directory")
	}
	sum := sha256.Sum256([]byte(path + "\x00" + name))
	return InstanceKey{path, name, hex.EncodeToString(sum[:16])}, nil
}
func (k InstanceKey) Dir() string       { return filepath.Join(k.Checkout, ".run", "instances", k.Name) }
func (k InstanceKey) StatePath() string { return filepath.Join(k.Dir(), "state.json") }
func (k InstanceKey) validate() error {
	actual, e := NewInstanceKey(k.Checkout, k.Name)
	if e != nil {
		return e
	}
	if k != actual {
		return errors.New("noncanonical instance key")
	}
	return nil
}
func ensureDir(k InstanceKey) error {
	if e := k.validate(); e != nil {
		return e
	}
	p := k.Checkout
	for _, part := range []string{".run", "instances", k.Name} {
		p = filepath.Join(p, part)
		if e := os.Mkdir(p, 0700); e != nil && !errors.Is(e, os.ErrExist) {
			return e
		}
		s, e := os.Lstat(p)
		if e != nil {
			return e
		}
		if !s.IsDir() || s.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe instance directory %s", p)
		}
	}
	return nil
}
func withLock(k InstanceKey, fn func() error) error {
	if e := ensureDir(k); e != nil {
		return e
	}
	fd, e := syscall.Open(filepath.Join(k.Dir(), "lock"), syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if e != nil {
		return e
	}
	defer syscall.Close(fd)
	if e = syscall.Flock(fd, syscall.LOCK_EX); e != nil {
		return e
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)
	return fn()
}
func SaveState(path string, s State) error {
	if filepath.Clean(path) != s.Key.StatePath() {
		return errors.New("state path is outside instance")
	}
	return withLock(s.Key, func() error { return saveState(path, s) })
}
func saveState(path string, s State) error {
	if e := s.Key.validate(); e != nil {
		return e
	}
	if path != s.Key.StatePath() {
		return errors.New("state instance mismatch")
	}
	if s.Version != StateVersion {
		return errors.New("unsupported state version")
	}
	b, e := json.MarshalIndent(s, "", "  ")
	if e != nil {
		return e
	}
	return atomicFile(path, append(b, '\n'))
}
func atomicFile(path string, b []byte) error {
	f, e := os.CreateTemp(filepath.Dir(path), ".write-*")
	if e != nil {
		return e
	}
	defer os.Remove(f.Name())
	defer f.Close()
	if e = f.Chmod(0600); e != nil {
		return e
	}
	if _, e = f.Write(b); e != nil {
		return e
	}
	if e = f.Sync(); e != nil {
		return e
	}
	if e = f.Close(); e != nil {
		return e
	}
	if e = os.Rename(f.Name(), path); e != nil {
		return e
	}
	d, e := os.Open(filepath.Dir(path))
	if e != nil {
		return e
	}
	defer d.Close()
	return d.Sync()
}
func LoadState(path string) (State, error) {
	var s State
	// Refuse links even for read-only status; do not read another instance's state.
	fd, e := syscall.Open(path, syscall.O_RDONLY|syscall.O_NOFOLLOW, 0)
	if e != nil {
		return s, e
	}
	f := os.NewFile(uintptr(fd), path)
	defer f.Close()
	if e = json.NewDecoder(f).Decode(&s); e != nil {
		return s, e
	}
	if s.Version != StateVersion {
		return s, errors.New("unsupported state version")
	}
	if e = s.Key.validate(); e != nil {
		return s, e
	}
	if path != s.Key.StatePath() {
		return s, errors.New("state instance mismatch")
	}
	if e = ensureDir(s.Key); e != nil {
		return s, e
	}
	return s, nil
}
