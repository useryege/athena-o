package devruntime

import (
	"errors"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
)

// LoadEnvironment parses data without sourcing a shell. An explicitly exported
// empty value overrides the file, and missing keys remain absent.
func LoadEnvironment(path string, exported []string) (map[string]string, error) {
	env, e := godotenv.Read(path)
	if e != nil {
		return nil, errors.New("cannot parse environment file")
	}
	for _, entry := range exported {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			env[key] = value
		}
	}
	return env, nil
}
func EnvironmentFor(env map[string]string, allow []string) []string {
	selected := map[string]string{}
	for _, key := range allow {
		if value, ok := env[key]; ok {
			selected[key] = value
		}
	}
	keys := make([]string, 0, len(selected))
	for key := range selected {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+selected[key])
	}
	return result
}
func (m *Manager) SaveSecret(name string, data []byte) error {
	if name == "" || name != filepath.Base(name) || name == "." || name == ".." || name == "state.json" || name == "lock" {
		return errors.New("invalid secret filename")
	}
	return withLock(m.Key, func() error { return atomicFile(filepath.Join(m.Key.Dir(), name), data) })
}
