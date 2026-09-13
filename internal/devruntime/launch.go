package devruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/useryege/athena/internal/accountstate/schema"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type RunOptions struct {
	Key             InstanceKey
	Services        []string
	DBMode, EnvFile string
}

func Build(ctx context.Context, key InstanceKey, names []string) (result error) {
	specs, err := ResolveServices(names)
	if err != nil {
		return err
	}
	for _, s := range specs {
		if s.Name == "ui" {
			return errors.New("build-service builds Go services; build UI with its own package build command")
		}
	}
	m := NewManager(key)
	if _, err = m.Begin(names, "managed"); err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		result = errors.Join(result, m.Stop(cleanup))
	}()
	paths, err := buildServices(ctx, key, specs)
	if err != nil {
		return err
	}
	for _, s := range specs {
		fmt.Printf("built %s: %s\n", s.Name, paths[s.Name])
	}
	return nil
}
func buildServices(ctx context.Context, key InstanceKey, specs []ServiceSpec) (map[string]string, error) {
	if err := ensureDir(key); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(key.Dir(), "build-")
	if err != nil {
		return nil, err
	}
	paths := map[string]string{}
	for _, s := range specs {
		if s.Name == "ui" {
			node, e := exec.LookPath("node")
			if e != nil {
				return nil, e
			}
			paths[s.Name], e = filepath.EvalSymlinks(node)
			if e != nil {
				return nil, e
			}
			if _, e = os.Stat(filepath.Join(key.Checkout, "ui/node_modules/vite/bin/vite.js")); e != nil {
				return nil, errors.New("UI dependencies missing; install using the project's Node version")
			}
			continue
		}
		path := filepath.Join(dir, s.Binary)
		cmd := exec.Command("go", "build", "-o", path, s.BuildPackage)
		cmd.Dir = key.Checkout
		cmd.Env = EnvironmentFor(environmentMap(os.Environ()), []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN", "CGO_ENABLED", "CC", "CXX", "PKG_CONFIG_PATH"})
		if err := NewManager(key).RunHelper(ctx, "build-"+s.Name, cmd, 30*time.Second); err != nil {
			return nil, err
		}
		paths[s.Name] = path
	}
	return paths, nil
}
func environmentMap(entries []string) map[string]string {
	m := map[string]string{}
	for _, s := range entries {
		for i, c := range s {
			if c == '=' {
				m[s[:i]] = s[i+1:]
				break
			}
		}
	}
	return m
}

// ImmutableExecutable protects /proc/exe identity even if a later build replaces
// a public dist executable. Existing hash paths are never overwritten.
func ImmutableExecutable(k InstanceKey, source string) (string, error) {
	if err := ensureDir(k); err != nil {
		return "", err
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	dir := filepath.Join(k.Dir(), "executables")
	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	path := filepath.Join(dir, hex.EncodeToString(sum[:]))
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if errors.Is(err, os.ErrExist) {
		existing, e := os.ReadFile(path)
		if e != nil || sha256.Sum256(existing) != sum {
			return "", errors.New("immutable executable mismatch")
		}
		return path, nil
	}
	if err != nil {
		return "", err
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	return path, errors.Join(err, closeErr)
}
func envDefault(env map[string]string, key, value string) string {
	if v, ok := env[key]; ok {
		return v
	}
	return value
}
func serviceAddress(name string, env map[string]string) string {
	switch name {
	case "trader-sync":
		return envDefault(env, "ATHENA_TRADER_SYNC_LISTEN_ADDRESS", "127.0.0.1:8122")
	case "api-server":
		return net.JoinHostPort(envDefault(env, "ATHENA_SERVER_LISTEN_ADDRESS", "127.0.0.1"), envDefault(env, "ATHENA_SERVER_PORT", "8080"))
	case "wallet":
		return net.JoinHostPort(envDefault(env, "ATHENA_WALLET_LISTEN_ADDRESS", "127.0.0.1"), envDefault(env, "ATHENA_WALLET_PORT", "8088"))
	case "profit-sharing":
		return net.JoinHostPort(envDefault(env, "ATHENA_PROFIT_SHARING_LISTEN_ADDRESS", "127.0.0.1"), envDefault(env, "ATHENA_PROFIT_SHARING_PORT", "8108"))
	case "notification":
		return net.JoinHostPort(envDefault(env, "ATHENA_NOTIFICATION_LISTEN_ADDRESS", "127.0.0.1"), envDefault(env, "ATHENA_NOTIFICATION_PORT", "8086"))
	case "ui":
		return net.JoinHostPort("127.0.0.1", envDefault(env, "ATHENA_UI_PORT", "4000"))
	}
	return ""
}
func preflightPorts(specs []ServiceSpec, env map[string]string) error {
	seen := map[string]bool{}
	for _, s := range specs {
		address := serviceAddress(s.Name, env)
		host, p, e := net.SplitHostPort(address)
		port, pe := strconv.Atoi(p)
		if e != nil || pe != nil || port < 1 || port > 65535 || host == "" {
			return fmt.Errorf("invalid %s listen address", s.Name)
		}
		if seen[address] {
			return errors.New("selected services have overlapping ports")
		}
		seen[address] = true
		l, e := net.Listen("tcp", address)
		if e != nil {
			return fmt.Errorf("%s port is unavailable", s.Name)
		}
		l.Close()
	}
	return nil
}
func Run(ctx context.Context, o RunOptions) (result error) {
	specs, err := ResolveServices(o.Services)
	if err != nil {
		return err
	}
	return runResolved(ctx, o, specs, false)
}

func runResolved(ctx context.Context, o RunOptions, specs []ServiceSpec, fullStack bool) (result error) {
	if o.DBMode == "" {
		o.DBMode = "managed"
	}
	if o.EnvFile == "" {
		o.EnvFile = ".env"
	}
	if !filepath.IsAbs(o.EnvFile) {
		o.EnvFile = filepath.Join(o.Key.Checkout, o.EnvFile)
	}
	env, err := LoadEnvironment(o.EnvFile, os.Environ())
	if err != nil {
		return err
	}
	m := NewManager(o.Key)
	if fullStack {
		prepareFullStackEnvironment(env)
	}
	env, err = m.PrepareEnvironment(env, specs, o.DBMode)
	if err != nil {
		return err
	}
	if err = preflightPorts(specs, env); err != nil {
		return err
	}
	if _, err = m.Begin(o.Services, o.DBMode); err != nil {
		return err
	}
	var children sync.WaitGroup
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		stopErr := m.Stop(cleanup)
		reaped := make(chan struct{})
		go func() { children.Wait(); close(reaped) }()
		select {
		case <-reaped:
		case <-cleanup.Done():
			stopErr = errors.Join(stopErr, errors.New("children were not reaped within cleanup budget"))
		}
		if result != nil {
			_ = m.Update(func(s *State) error {
				s.Failures = append(s.Failures, "initial startup failed; inspect instance logs")
				return nil
			})
		}
		result = errors.Join(result, stopErr)
	}()
	paths, err := buildServices(ctx, o.Key, specs)
	if err != nil {
		return err
	}
	fingerprints := map[string]string{}
	for name, path := range paths {
		data, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		sum := sha256.Sum256(data)
		fingerprints[name] = hex.EncodeToString(sum[:])
	}
	if err = m.Update(func(s *State) error { s.BuildFingerprints = fingerprints; return nil }); err != nil {
		return err
	}
	if needsDatabase(specs) {
		if o.DBMode == "external" {
			address, database, e := externalDatabaseIdentity(env[schema.DSNEnv])
			if e != nil {
				return e
			}
			sum := sha256.Sum256([]byte(env[schema.DSNEnv]))
			fingerprint := hex.EncodeToString(sum[:])
			if err = m.Update(func(s *State) error {
				if s.ConfigFingerprints == nil {
					s.ConfigFingerprints = map[string]string{}
				}
				old := s.ConfigFingerprints["external-database"]
				if old != "" && old != fingerprint {
					return errors.New("external database cannot change in place; use a new instance")
				}
				s.ConfigFingerprints["external-database"] = fingerprint
				s.Endpoints["postgres"] = address
				s.Endpoints["database"] = database
				return nil
			}); err != nil {
				return err
			}
		}
		if o.DBMode == "managed" {
			dsn, e := m.preparePostgres(ctx)
			if e != nil {
				return e
			}
			env[schema.DSNEnv] = dsn
			if e = runSchema(ctx, o.Key, "up", env); e != nil {
				return e
			}
		}
		if err = runSchema(ctx, o.Key, "verify", env); err != nil {
			return err
		}
	}
	if fullStack {
		if err = m.prepareFullStackDatabases(ctx, env); err != nil {
			return err
		}
	}
	if selected(specs, "api-server") {
		if err = m.prepareAPIInfrastructure(ctx, env); err != nil {
			return err
		}
	}
	persisted := map[string]string{}
	for _, s := range specs {
		for _, k := range s.EnvironmentKeys {
			if v, ok := env[k]; ok {
				persisted[k] = v
			}
		}
	}
	for _, k := range []string{"ATHENA_TRADER_SYNC_TLS_CA_FILE", "ATHENA_TRADER_SYNC_TLS_SERVER_NAME"} {
		if v, ok := env[k]; ok {
			persisted[k] = v
		}
	}
	data, _ := json.Marshal(persisted)
	configHash := sha256.Sum256(data)
	if err = m.Update(func(s *State) error {
		if s.ConfigFingerprints == nil {
			s.ConfigFingerprints = map[string]string{}
		}
		s.ConfigFingerprints["environment"] = hex.EncodeToString(configHash[:])
		return nil
	}); err != nil {
		return err
	}
	if err = m.SaveSecret("environment.json", data); err != nil {
		return err
	}
	// Trader Sync takes ownership and reaches RPC readiness before other services.
	var ordered []ServiceSpec
	for _, s := range specs {
		if s.Name == "trader-sync" {
			ordered = append(ordered, s)
		}
	}
	for _, s := range specs {
		if s.Name != "trader-sync" {
			ordered = append(ordered, s)
		}
	}
	for _, s := range ordered {
		address := serviceAddress(s.Name, env)
		args := append([]string{}, s.Args...)
		if s.Name == "api-server" || s.Name == "notification" || s.Name == "wallet" || s.Name == "profit-sharing" {
			_, port, _ := net.SplitHostPort(address)
			args = append(args, "--port", port)
		}
		if s.Name == "ui" {
			_, port, _ := net.SplitHostPort(address)
			args = []string{filepath.Join(o.Key.Checkout, "ui/node_modules/vite/bin/vite.js"), "--host", "127.0.0.1", "--port", port, "--strictPort"}
		}
		cmd := exec.Command(paths[s.Name], args...)
		cmd.Dir = o.Key.Checkout
		if s.Name == "ui" {
			cmd.Dir = filepath.Join(o.Key.Checkout, "ui")
		}
		cmd.Env = EnvironmentFor(env, s.EnvironmentKeys)
		logPath := filepath.Join(o.Key.Dir(), "build-logs-"+NewRunID()+"-"+s.Name+".log")
		log, e := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if e != nil {
			return e
		}
		cmd.Stdout = log
		cmd.Stderr = log
		budget := s.ShutdownTimeout
		if s.Name == "trader-sync" {
			if value, ok := env["ATHENA_TRADER_SYNC_SHUTDOWN_TIMEOUT"]; ok {
				budget, e = time.ParseDuration(value)
				if e != nil || budget <= 0 {
					log.Close()
					return errors.New("invalid Trader Sync shutdown timeout")
				}
			}
		}
		if _, e = m.Spawn(s.Name, cmd, budget); e != nil {
			log.Close()
			return e
		}
		children.Add(1)
		go func(name string) {
			defer children.Done()
			e := cmd.Wait()
			code := 0
			if e != nil {
				code = 1
				if cmd.ProcessState != nil {
					code = cmd.ProcessState.ExitCode()
				}
			}
			_ = m.RecordExit(name, code)
			log.Close()
		}(s.Name)
		if e = m.Update(func(state *State) error { state.Logs[s.Name] = logPath; state.Endpoints[s.Name] = address; return nil }); e != nil {
			return e
		}
		ready, cancel := context.WithTimeout(ctx, s.StartupTimeout)
		e = waitService(ready, m, s.Name, address, env)
		cancel()
		if e != nil {
			return e
		}
	}
	if err = m.Update(func(s *State) error {
		if s.Phase != "starting" {
			return errors.New("instance stopped during startup")
		}
		if selectedServiceExited(*s) {
			return errors.New("service exited before all services were ready")
		}
		s.Phase = "running"
		return nil
	}); err != nil {
		return err
	}
	fmt.Printf("instance %s is ready; status and logs: %s\n", o.Key.Name, o.Key.Dir())
	<-ctx.Done()
	return nil
}
func runSchema(ctx context.Context, k InstanceKey, action string, env map[string]string) error {
	deadline, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	dir, err := os.MkdirTemp(k.Dir(), "migration-")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "athena-account-state-migrate")
	build := exec.Command("go", "build", "-o", path, "./cmd/athena-account-state-migrate")
	build.Dir = k.Checkout
	build.Env = EnvironmentFor(environmentMap(os.Environ()), []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN"})
	m := NewManager(k)
	if err = m.RunHelper(deadline, "build-schema-"+action, build, 30*time.Second); err != nil {
		return err
	}
	cmd := exec.Command(path, action, "--timeout=120s")
	cmd.Dir = k.Checkout
	cmd.Env = EnvironmentFor(env, keys(toolEnvironment, []string{schema.DSNEnv}))
	return m.RunHelper(deadline, "schema-"+action, cmd, 30*time.Second)
}

func waitService(ctx context.Context, m *Manager, name, address string, env map[string]string) error {
	for {
		state, e := m.Status()
		if e != nil {
			return e
		}
		if selectedServiceExited(state) {
			return errors.New("service exited during initial startup; inspect instance logs")
		}
		probe, cancel := context.WithTimeout(ctx, time.Second)
		e = probeService(probe, name, address, env)
		cancel()
		if e == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s readiness failed: %w", name, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
func probeService(ctx context.Context, name, address string, env map[string]string) error {
	if name == "trader-sync" || name == "notification" || name == "wallet" || name == "profit-sharing" {
		credentials := insecure.NewCredentials()
		service := ""
		if name == "trader-sync" {
			service = "tradersync.internal.v1.TraderSyncService"
			cfg := rpcconfig.Client{Address: address, Transport: env["ATHENA_TRADER_SYNC_GRPC_TRANSPORT"], CAFile: env["ATHENA_TRADER_SYNC_TLS_CA_FILE"], ServerName: env["ATHENA_TRADER_SYNC_TLS_SERVER_NAME"]}
			var err error
			credentials, err = rpcconfig.ClientCredentials(cfg)
			if err != nil {
				return err
			}
		}
		conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(credentials))
		if err != nil {
			return err
		}
		defer conn.Close()
		response, err := healthpb.NewHealthClient(conn).Check(ctx, &healthpb.HealthCheckRequest{Service: service})
		if err != nil {
			return err
		}
		if response.Status != healthpb.HealthCheckResponse_SERVING {
			return errors.New("not serving")
		}
		return nil
	}
	path := "/"
	if name == "api-server" {
		path = "/healthz"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+path, nil)
	if err != nil {
		return err
	}
	client := http.Client{Transport: &http.Transport{Proxy: nil}}
	defer client.CloseIdleConnections()
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
	if res.StatusCode != http.StatusOK {
		return errors.New("HTTP readiness failed")
	}
	return nil
}
func Status(ctx context.Context, k InstanceKey) (State, error) {
	m := NewManager(k)
	s, err := m.Status()
	if err != nil {
		return s, err
	}
	s.Health = map[string]string{}
	data, err := os.ReadFile(filepath.Join(k.Dir(), "environment.json"))
	if err != nil {
		return s, nil
	}
	var env map[string]string
	if json.Unmarshal(data, &env) != nil {
		return s, errors.New("invalid instance health configuration")
	}
	for _, name := range s.Services {
		p, ok := s.Processes[name]
		if !ok {
			s.Health[name] = "not started"
			continue
		}
		if _, e := VerifyProcess(p); e != nil {
			s.Health[name] = "unavailable"
			continue
		}
		probe, cancel := context.WithTimeout(ctx, time.Second)
		e := probeService(probe, name, s.Endpoints[name], env)
		cancel()
		s.Health[name] = "unavailable"
		if e == nil {
			s.Health[name] = "ready"
		}
	}
	if dsn := env[schema.DSNEnv]; dsn != "" {
		probe, cancel := context.WithTimeout(ctx, time.Second)
		err = waitDatabase(probe, dsn)
		cancel()
		s.Health["postgres"] = "unavailable"
		if err == nil {
			s.Health["postgres"] = "ready"
		}
	}
	return s, nil
}
