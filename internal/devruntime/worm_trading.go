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
	wormstore "github.com/useryege/athena/internal/wormtrading/store"
)

// prepareWormTradingDatabase owns only the selected Trading schema. The account
// schema is prepared by its existing owner before this function is called.
func (m *Manager) prepareWormTradingDatabase(ctx context.Context, env map[string]string) error {
	state, err := m.Status()
	if err != nil {
		return err
	}
	if state.Phase != "starting" {
		return errors.New("Worm Trading schema requires a starting instance")
	}
	if state.DBMode == "managed" {
		dsn, err := url.Parse(env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"])
		if err != nil || dsn.Host == "" || dsn.Host != state.Endpoints["postgres"] {
			return errors.New("Worm Trading requires this instance's managed PostgreSQL")
		}
		deadline, cancel := context.WithTimeout(ctx, wormstore.DefaultTimeout)
		defer cancel()
		conn, err := pgx.Connect(deadline, dsn.String())
		if err != nil {
			return fmt.Errorf("connect managed PostgreSQL: %w", err)
		}
		defer conn.Close(context.Background())
		var exists bool
		if err := conn.QueryRow(deadline, `SELECT EXISTS(SELECT FROM pg_database WHERE datname='worm_trading')`).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			if _, err := conn.Exec(deadline, `CREATE DATABASE worm_trading`); err != nil {
				return err
			}
		}
		dsn.Path = "/worm_trading"
		dsn.RawPath = ""
		env[wormstore.DSNEnv] = dsn.String()
	} else if state.DBMode == "external" {
		if strings.TrimSpace(env[wormstore.DSNEnv]) == "" {
			return fmt.Errorf("external database requires explicit %s", wormstore.DSNEnv)
		}
		address, database, err := externalDatabaseIdentity(env[wormstore.DSNEnv])
		if err != nil {
			return err
		}
		hash := sha256.Sum256([]byte(env[wormstore.DSNEnv]))
		fingerprint := hex.EncodeToString(hash[:])
		if err := m.Update(func(s *State) error {
			if s.ConfigFingerprints == nil {
				s.ConfigFingerprints = map[string]string{}
			}
			old := s.ConfigFingerprints["external-worm-trading-database"]
			if old != "" && old != fingerprint {
				return errors.New("external Worm Trading database cannot change in place; use a new instance")
			}
			s.ConfigFingerprints["external-worm-trading-database"] = fingerprint
			s.Endpoints["worm-trading-postgres"] = address
			s.Endpoints["worm-trading-database"] = database
			return nil
		}); err != nil {
			return err
		}
	} else {
		return errors.New("unknown database mode")
	}
	deadline, cancel := context.WithTimeout(ctx, wormstore.DefaultTimeout)
	defer cancel()
	dir, err := os.MkdirTemp(m.Key.Dir(), "worm-trading-migration-")
	if err != nil {
		return err
	}
	binary := filepath.Join(dir, "athena-worm-trading-migrate")
	build := exec.Command("go", "build", "-o", binary, "./cmd/athena-worm-trading-migrate")
	build.Dir = m.Key.Checkout
	build.Env = EnvironmentFor(environmentMap(os.Environ()), []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN", "CGO_ENABLED", "CC", "CXX", "PKG_CONFIG_PATH"})
	if err := m.RunHelper(deadline, "build-worm-trading-schema", build, 30*time.Second); err != nil {
		return err
	}
	actions := []string{"verify"}
	if state.DBMode == "managed" {
		actions = []string{"up", "verify"}
	}
	for _, action := range actions {
		cmd := exec.Command(binary, action, "--timeout=120s")
		cmd.Dir = m.Key.Checkout
		cmd.Env = EnvironmentFor(env, keys(toolEnvironment, loggingEnvironment, []string{wormstore.DSNEnv}))
		if err := m.RunHelper(deadline, "worm-trading-schema-"+action, cmd, 30*time.Second); err != nil {
			return err
		}
	}
	return nil
}
