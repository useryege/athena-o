package devruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type CommandExecutor func(context.Context, string, ...string) ([]byte, error)
type Docker struct{ Exec CommandExecutor }

func command(ctx context.Context, name string, args ...string) ([]byte, error) {
	// Do not return stderr: daemon diagnostics can echo configuration secrets.
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = EnvironmentFor(environmentMap(os.Environ()), keys(toolEnvironment, []string{"DOCKER_HOST", "DOCKER_CONTEXT", "DOCKER_CONFIG", "DOCKER_CERT_PATH", "DOCKER_TLS_VERIFY", "XDG_RUNTIME_DIR", "SSH_AUTH_SOCK"}))
	b, e := cmd.Output()
	if e != nil {
		return nil, fmt.Errorf("%s command failed: %w", name, e)
	}
	return b, nil
}
func ResourceLabels(k InstanceKey, kind, run string, name ...string) map[string]string {
	labels := map[string]string{"athena.namespace": k.Namespace, "athena.instance": k.Name, "athena.run": run, "athena.kind": kind}
	if len(name) > 0 {
		labels["athena.resource"] = name[0]
	}
	return labels
}
func validKind(kind string) bool { return kind == "container" || kind == "volume" }
func labelArgs(k InstanceKey, kind, run, name, flag string) []string {
	labels := ResourceLabels(k, kind, run, name)
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var args []string
	for _, key := range keys {
		value := key + "=" + labels[key]
		if flag == "--filter" {
			value = "label=" + value
		}
		args = append(args, flag, value)
	}
	return args
}
func (d Docker) inspect(ctx context.Context, k InstanceKey, r ResourceRef) error {
	if !validKind(r.Kind) || r.ID == "" || r.Namespace != k.Namespace || r.RunID == "" || !r.Owned {
		return errors.New("resource is not owned by instance")
	}
	b, e := d.Exec(ctx, "docker", r.Kind, "inspect", "--format", "{{json .}}", r.ID)
	if e != nil {
		return e
	}
	var v struct {
		ID     string `json:"Id"`
		Name   string
		Labels map[string]string
		Config struct{ Labels map[string]string }
	}
	if e = json.Unmarshal(b, &v); e != nil {
		return e
	}
	labels := v.Labels
	if r.Kind == "container" {
		labels = v.Config.Labels
		if v.ID != r.ID {
			return errors.New("container ID mismatch")
		}
	} else if v.Name != r.ID {
		return errors.New("volume identity mismatch")
	}
	if strings.TrimPrefix(v.Name, "/") != r.Name {
		return errors.New("resource name mismatch")
	}
	for name, value := range ResourceLabels(k, r.Kind, r.RunID, r.Name) {
		if labels[name] != value {
			return fmt.Errorf("resource ownership label mismatch: %s", name)
		}
	}
	return nil
}
func (m *Manager) CreateResource(ctx context.Context, kind, name string, args []string) (ResourceRef, error) {
	var r ResourceRef
	if !validKind(kind) || name == "" || strings.HasPrefix(name, "-") {
		return r, errors.New("invalid resource kind or name")
	}
	// Ownership labels and the resource name belong exclusively to this layer.
	for _, arg := range args {
		if arg == "--label" || strings.HasPrefix(arg, "--label=") || strings.HasPrefix(arg, "-l") || arg == "--label-file" || strings.HasPrefix(arg, "--label-file=") || arg == "--name" || strings.HasPrefix(arg, "--name=") {
			return r, errors.New("resource ownership arguments are reserved")
		}
	}
	if kind == "volume" && len(args) > 0 {
		return r, errors.New("volume creation accepts no extra arguments")
	}
	var intent CreationIntent
	e := m.Update(func(s *State) error {
		if s.Phase != "starting" && s.Phase != "running" {
			return errors.New("instance is not starting or running")
		}
		for _, i := range s.Intents {
			if i.Kind == kind && i.Name == name {
				return errors.New("resource creation already pending")
			}
		}
		intent = CreationIntent{kind, name, m.Key.Namespace, s.RunID}
		s.Intents = append(s.Intents, intent)
		return nil
	})
	if e != nil {
		return r, e
	}
	argv := append([]string{kind, "create"}, labelArgs(m.Key, kind, intent.RunID, name, "--label")...)
	if kind == "container" {
		argv = append(argv, "--name", name)
		argv = append(argv, args...)
	} else {
		if len(args) != 0 {
			return r, errors.New("volume creation accepts no extra arguments")
		}
		argv = append(argv, name)
	}
	b, e := m.Docker.Exec(ctx, "docker", argv...)
	if e != nil {
		return r, e
	}
	id := strings.TrimSpace(string(b))
	if id == "" || strings.ContainsAny(id, "\r\n ") {
		return r, errors.New("docker returned invalid resource ID")
	}
	r = ResourceRef{Kind: kind, ID: id, Name: name, Namespace: m.Key.Namespace, RunID: intent.RunID, Owned: true}
	if e = m.Docker.inspect(ctx, m.Key, r); e != nil {
		return r, e
	}
	e = m.Update(func(s *State) error {
		if s.RunID != intent.RunID {
			return errors.New("run changed during creation")
		}
		addResource(s, r)
		removeIntent(s, intent)
		return nil
	})
	return r, e
}
func addResource(s *State, r ResourceRef) {
	for i, old := range s.Resources {
		if old.Kind == r.Kind && old.ID == r.ID {
			s.Resources[i] = r
			return
		}
	}
	s.Resources = append(s.Resources, r)
}
func removeIntent(s *State, intent CreationIntent) {
	for i := len(s.Intents) - 1; i >= 0; i-- {
		if s.Intents[i] == intent {
			s.Intents = append(s.Intents[:i], s.Intents[i+1:]...)
		}
	}
}
func (m *Manager) Recover(ctx context.Context) error {
	s, e := m.Status()
	if e != nil {
		return e
	}
	for _, intent := range s.Intents {
		if !validKind(intent.Kind) || intent.Namespace != m.Key.Namespace || intent.RunID == "" {
			return errors.New("invalid resource creation intent")
		}
		args := []string{intent.Kind, "ls", "--quiet"}
		if intent.Kind == "container" {
			args = append(args, "--all", "--no-trunc")
		}
		args = append(args, labelArgs(m.Key, intent.Kind, intent.RunID, intent.Name, "--filter")...)
		b, e := m.Docker.Exec(ctx, "docker", args...)
		if e != nil {
			return e
		}
		var found []ResourceRef
		for _, id := range strings.Fields(string(b)) {
			r := ResourceRef{Kind: intent.Kind, ID: id, Name: intent.Name, Namespace: intent.Namespace, RunID: intent.RunID, Owned: true}
			if e = m.Docker.inspect(ctx, m.Key, r); e != nil {
				return e
			}
			found = append(found, r)
		}
		if len(found) > 1 {
			return errors.New("creation intent matched multiple resources")
		}
		// No result cannot establish that an in-flight create will never finish.
		if len(found) == 0 {
			return errors.New("creation intent is unresolved; no exactly owned resource found")
		}
		if e = m.Update(func(s *State) error {
			for _, r := range found {
				addResource(s, r)
			}
			removeIntent(s, intent)
			return nil
		}); e != nil {
			return e
		}
	}
	return nil
}

// absent is positive daemon evidence, not an interpretation of inspect stderr.
// Callers must already possess an exact durable deletion intent for this ID.
func (d Docker) absent(ctx context.Context, r ResourceRef) (bool, error) {
	if !validKind(r.Kind) || r.ID == "" {
		return false, errors.New("invalid deletion resource")
	}
	args := []string{r.Kind, "ls", "--quiet"}
	if r.Kind == "container" {
		args = append(args, "--all", "--no-trunc")
	}
	b, e := d.Exec(ctx, "docker", args...)
	if e != nil {
		return false, e
	}
	for _, id := range strings.Fields(string(b)) {
		if id == r.ID {
			return false, nil
		}
	}
	return true, nil
}
