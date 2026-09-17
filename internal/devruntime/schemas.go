package devruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func (m *Manager) prepareSelectedDatabases(ctx context.Context, env map[string]string, specs []ServiceSpec, paths map[string]string) error {
	state, err := m.Status()
	if err != nil {
		return err
	}
	admin := ""
	if state.DBMode == "managed" {
		admin, err = m.preparePostgres(ctx)
		if err != nil {
			return err
		}
	}
	return m.prepareDatabaseSchemas(ctx, env, specs, paths, admin)
}

// Preparation is complete before any selected application is launched. External
// targets never reach database creation or schema up.
func (m *Manager) prepareDatabaseSchemas(ctx context.Context, env map[string]string, specs []ServiceSpec, paths map[string]string, admin string) error {
	state, err := m.Status()
	if err != nil {
		return err
	}
	if state.Phase != "starting" {
		return errors.New("schema preparation requires starting instance")
	}
	owners := selectedSchemas(specs)
	if state.DBMode == "managed" {
		location, err := url.Parse(admin)
		if err != nil || location.Host == "" || location.Host != state.Endpoints["postgres"] {
			return errors.New("schema preparation requires owned PostgreSQL endpoint")
		}
		deadline, cancel := context.WithTimeout(ctx, 120*time.Second)
		conn, err := pgx.Connect(deadline, admin)
		if err != nil {
			cancel()
			return err
		}
		for _, owner := range owners {
			var exists bool
			if err = conn.QueryRow(deadline, "SELECT EXISTS(SELECT FROM pg_database WHERE datname=$1)", owner.Database).Scan(&exists); err == nil && !exists {
				_, err = conn.Exec(deadline, "CREATE DATABASE "+pgx.Identifier{owner.Database}.Sanitize())
			}
			if err != nil {
				break
			}
			target := *location
			target.Path = "/" + owner.Database
			target.RawPath = ""
			env[owner.DSNEnv] = target.String()
		}
		conn.Close(context.Background())
		cancel()
		if err != nil {
			return err
		}
	} else if state.DBMode == "external" {
		for _, owner := range owners {
			if strings.TrimSpace(env[owner.DSNEnv]) == "" {
				return fmt.Errorf("external database requires explicit %s", owner.DSNEnv)
			}
			address, database, err := externalDatabaseIdentity(env[owner.DSNEnv])
			if err != nil {
				return err
			}
			sum := sha256.Sum256([]byte(env[owner.DSNEnv]))
			fingerprint := hex.EncodeToString(sum[:])
			if err = m.Update(func(s *State) error {
				if s.Endpoints == nil {
					s.Endpoints = map[string]string{}
				}
				if s.ConfigFingerprints == nil {
					s.ConfigFingerprints = map[string]string{}
				}
				key := "external-" + owner.Name + "-database"
				if old := s.ConfigFingerprints[key]; old != "" && old != fingerprint {
					return fmt.Errorf("external %s database cannot change in place; use a new instance", owner.Name)
				}
				s.ConfigFingerprints[key] = fingerprint
				s.Endpoints[owner.Name+"-postgres"] = address
				s.Endpoints[owner.Name+"-database"] = database
				return nil
			}); err != nil {
				return err
			}
		}
	} else {
		return errors.New("unknown database mode")
	}
	if needsSchema(specs, "solana-discovery") && env["ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN"] != env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"] {
		return errors.New("Solana and account schemas require the same DSN")
	}
	binaries := map[string]string{}
	for _, owner := range owners {
		binary := paths[owner.Name]
		if len(owner.Args) == 0 {
			binary = ""
		}
		if binary == "" {
			binary, err = m.buildSchemaOwner(ctx, owner)
			if err != nil {
				return err
			}
		}
		binaries[owner.Name] = binary
		actions := []string{"verify"}
		if state.DBMode == "managed" {
			actions = []string{"up", "verify"}
		}
		if err = m.runSchemaOwner(ctx, owner, actions, env, binary); err != nil {
			return err
		}
		if owner.Name == "solana-discovery" {
			if err = m.runSchemaOwner(ctx, schemaOwners()[0], []string{"verify"}, env, binaries["account"]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (m *Manager) buildSchemaOwner(ctx context.Context, owner schemaOwner) (string, error) {
	dir, err := os.MkdirTemp(m.Key.Dir(), "migration-")
	if err != nil {
		return "", err
	}
	binary := filepath.Join(dir, "athena-"+owner.Name+"-schema")
	cmd := exec.Command("go", "build", "-o", binary, owner.BuildPackage)
	cmd.Dir = m.Key.Checkout
	cmd.Env = EnvironmentFor(environmentMap(os.Environ()), []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN", "CGO_ENABLED", "CC", "CXX", "PKG_CONFIG_PATH"})
	deadline, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	if err = m.RunHelper(deadline, "build-schema-"+owner.Name, cmd, 30*time.Second); err != nil {
		return "", err
	}
	return binary, nil
}
func (m *Manager) runSchemaOwner(ctx context.Context, owner schemaOwner, actions []string, env map[string]string, binary string) error {
	var err error
	if binary == "" {
		binary, err = m.buildSchemaOwner(ctx, owner)
		if err != nil {
			return err
		}
	}
	for _, action := range actions {
		if action != "up" && action != "verify" {
			return errors.New("unknown schema action")
		}
		args := append(append([]string{}, owner.Args...), action, "--timeout=120s")
		cmd := exec.Command(binary, args...)
		cmd.Dir = m.Key.Checkout
		cmd.Env = EnvironmentFor(env, keys(toolEnvironment, loggingEnvironment, []string{owner.DSNEnv}))
		deadline, cancel := context.WithTimeout(ctx, 120*time.Second)
		err = m.RunHelper(deadline, owner.Name+"-schema-"+action, cmd, 30*time.Second)
		cancel()
		if err != nil {
			return err
		}
	}
	return nil
}
