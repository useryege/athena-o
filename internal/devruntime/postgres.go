package devruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const postgresImage = "postgres:16"

func (m *Manager) preparePostgres(ctx context.Context) (string, error) {
	lease, err := acquireOperation(m.Key)
	if err != nil {
		return "", err
	}
	defer lease.close()
	s, err := m.Status()
	if err != nil {
		return "", err
	}
	if s.InitializationMode != "" && s.InitializationMode != "athena-postgres-v1" {
		return "", errors.New("database initialization layout cannot change; use a new instance")
	}
	if s.DBMode != "managed" || s.Phase != "starting" {
		return "", errors.New("managed postgres requires starting managed instance")
	}
	if _, err = m.Docker.Exec(ctx, "docker", "image", "inspect", postgresImage); err != nil {
		if _, err = m.Docker.Exec(ctx, "docker", "pull", postgresImage); err != nil {
			return "", errors.New("PostgreSQL image unavailable; no database resources created")
		}
	}
	passwordPath := filepath.Join(m.Key.Dir(), "postgres-password")
	password, err := os.ReadFile(passwordPath)
	if errors.Is(err, os.ErrNotExist) {
		raw := make([]byte, 32)
		if _, err = rand.Read(raw); err != nil {
			return "", err
		}
		password = []byte(hex.EncodeToString(raw))
		if err = m.SaveSecret("postgres-password", password); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", errors.New("cannot read instance postgres password")
	}
	if len(password) != 64 || strings.ContainsAny(string(password), "\r\n") {
		return "", errors.New("invalid instance postgres password")
	}
	volumeName := "athena-" + m.Key.Namespace + "-postgres-data"
	containerName := "athena-" + m.Key.Namespace + "-postgres"
	var volume, container ResourceRef
	for _, r := range s.Resources {
		if r.Name == volumeName {
			volume = r
		}
		if r.Name == containerName {
			container = r
		}
	}
	if volume.ID != "" {
		if err = m.Docker.inspect(ctx, m.Key, volume); err != nil {
			return "", err
		}
	} else {
		volume, err = m.CreateResource(ctx, "volume", volumeName, nil)
		if err != nil {
			return "", err
		}
	}
	if container.ID != "" {
		if err = m.Docker.inspect(ctx, m.Key, container); err != nil {
			return "", err
		}
	} else {
		if err = m.SaveSecret("postgres.env", []byte("POSTGRES_USER=athena\nPOSTGRES_DB=athena\nPOSTGRES_PASSWORD="+string(password)+"\n")); err != nil {
			return "", err
		}
		container, err = m.CreateResource(ctx, "container", containerName, []string{"--env-file", filepath.Join(m.Key.Dir(), "postgres.env"), "--publish", "127.0.0.1::5432", "--mount", "type=volume,source=" + volume.ID + ",target=/var/lib/postgresql/data", postgresImage})
		if err != nil {
			return "", err
		}
	}
	if _, err = m.Docker.Exec(ctx, "docker", "container", "start", container.ID); err != nil {
		return "", err
	}
	address, err := m.containerAddress(ctx, container, "5432/tcp")
	if err != nil {
		return "", err
	}
	dsn := (&url.URL{Scheme: "postgres", User: url.UserPassword("athena", string(password)), Host: address, Path: "/athena", RawQuery: "sslmode=disable"}).String()
	ready, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	if err = waitDatabase(ready, dsn); err != nil {
		return "", err
	}
	err = m.Update(func(s *State) error {
		if s.RunID == "" || s.Phase != "starting" {
			return errors.New("instance stopped during database initialization")
		}
		if s.Endpoints == nil {
			s.Endpoints = map[string]string{}
		}
		s.Endpoints["postgres"] = address
		s.InitializationMode = "athena-postgres-v1"
		return nil
	})
	return dsn, err
}
func waitDatabase(ctx context.Context, dsn string) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return errors.New("invalid database connection configuration")
	}
	defer pool.Close()
	for {
		probe, cancel := context.WithTimeout(ctx, time.Second)
		err = pool.Ping(probe)
		cancel()
		if err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("database readiness: %w", ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func (m *Manager) containerAddress(ctx context.Context, container ResourceRef, port string) (string, error) {
	b, err := m.Docker.Exec(ctx, "docker", "container", "inspect", "--format", "{{json .NetworkSettings.Ports}}", container.ID)
	if err != nil {
		return "", err
	}
	var ports map[string][]struct{ HostIp, HostPort string }
	if err = json.Unmarshal(b, &ports); err != nil {
		return "", err
	}
	bindings := ports[port]
	if len(bindings) != 1 || bindings[0].HostIp != "127.0.0.1" {
		return "", errors.New("postgres requires one loopback port binding")
	}
	address := net.JoinHostPort(bindings[0].HostIp, bindings[0].HostPort)
	return address, nil
}

func externalDatabaseIdentity(dsn string) (string, string, error) {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return "", "", errors.New("invalid external database connection configuration")
	}
	return net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port))), cfg.Database, nil
}
