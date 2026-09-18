package devruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	operationlogrpcconfig "github.com/useryege/athena/internal/operationlog/rpcconfig"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
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
		cleanup, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()
		if result != nil {
			_ = m.Update(func(s *State) error { s.Failures = append(s.Failures, result.Error()); return nil })
		}
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
		buildCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
		err := NewManager(key).RunHelper(buildCtx, "build-"+s.Name, cmd, 30*time.Second)
		cancel()
		if err != nil {
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
	spec, ok := serviceRegistry()[name]
	if !ok {
		return ""
	}
	return spec.Address(env)
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
	m := NewManager(o.Key)
	if old, e := m.Status(); e == nil && old.Supervisor.PID > 0 {
		if _, e = VerifyProcess(old.Supervisor); e == nil {
			return fmt.Errorf("instance %s already active in %s: phase=%s stage=%s supervisor=%d", o.Key.Name, o.Key.Checkout, old.Phase, old.Stage, old.Supervisor.PID)
		}
	}
	env, err := LoadEnvironment(o.EnvFile, os.Environ())
	if err != nil {
		return err
	}
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
	startup, startupCancel := context.WithTimeout(ctx, 30*time.Minute)
	defer startupCancel()
	var children sync.WaitGroup
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()
		if result != nil {
			_ = m.Update(func(s *State) error { s.Failures = append(s.Failures, result.Error()); return nil })
		}
		stopErr := m.Stop(cleanup)
		if state, err := m.Status(); err == nil && !state.StopDeadline.IsZero() {
			reapContext, reapCancel := context.WithDeadline(cleanup, state.StopDeadline)
			defer reapCancel()
			cleanup = reapContext
		}
		reaped := make(chan struct{})
		go func() { children.Wait(); close(reaped) }()
		select {
		case <-reaped:
		case <-cleanup.Done():
			stopErr = errors.Join(stopErr, errors.New("children were not reaped within cleanup budget"))
		}
		if final, e := m.Status(); e == nil && len(final.Failures) > 0 {
			result = errors.Join(result, errors.New(strings.Join(final.Failures, "; ")))
		}
		result = errors.Join(result, stopErr)
	}()
	if err = m.stage("build"); err != nil {
		return err
	}
	paths, err := buildServices(startup, o.Key, specs)
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
	if err = m.stage("infrastructure-schema"); err != nil {
		return err
	}
	if needsDatabase(specs) {
		if err = m.prepareSelectedDatabases(startup, env, specs, paths); err != nil {
			return err
		}
	}
	if selected(specs, "api-server") {
		if err = m.prepareAPIInfrastructure(startup, env); err != nil {
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
	if err = m.startApplications(startup, specs, paths, env, &children); err != nil {
		return err
	}
	startupCancel()
	gateways, gatewayErr := ProbeGateways(ctx, gatewayEnvironment(specs, env))
	address := ""
	if selected(specs, "api-server") {
		address = serviceAddress("api-server", env)
	}
	access := readAccessSettings(ctx, address, env)
	if err = m.Update(func(s *State) error {
		s.Gateways = gateways
		s.AccessSettings = access
		if gatewayErr != nil {
			s.GatewayError = gatewayErr.Error()
		}
		return nil
	}); err != nil {
		return err
	}
	if state, e := m.Status(); e == nil {
		printRunSummary(state, env)
	}
	<-ctx.Done()
	return nil
}
func (m *Manager) startApplications(ctx context.Context, specs []ServiceSpec, paths map[string]string, env map[string]string, children *sync.WaitGroup) error {
	ordered := append([]ServiceSpec(nil), specs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.Core != b.Core {
			return a.Core
		}
		if a.Core && a.StopLayer != b.StopLayer {
			return a.StopLayer > b.StopLayer
		}
		if a.Core && a.StopLayer == 0 && b.StopLayer == 0 {
			return a.Name == "api-server" && b.Name == "ui"
		}
		return false
	})
	coreAnnounced := false
	for _, spec := range ordered {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !spec.Core && !coreAnnounced {
			if err := m.markCore(specs); err != nil {
				return err
			}
			coreAnnounced = true
		}
		stage := "business-startup"
		if spec.Core {
			stage = "core-startup"
		}
		if err := m.stage(stage); err != nil {
			return err
		}
		if err := m.Update(func(s *State) error {
			if s.Startup == nil {
				s.Startup = map[string]ServiceStartup{}
			}
			s.Startup[spec.Name] = ServiceStartup{Status: "starting", At: time.Now()}
			return nil
		}); err != nil {
			return err
		}
		err := m.startApplication(ctx, spec, paths, env, children)
		if err == nil {
			err = ctx.Err()
		}
		status := "ready"
		message := ""
		if err != nil {
			status = "failed"
			message = err.Error()
		}
		if updateErr := m.Update(func(s *State) error {
			if err == nil {
				if _, exited := s.ExitCodes[spec.Name]; exited {
					err = fmt.Errorf("%s exited before readiness was recorded", spec.Name)
					status = "failed"
					message = err.Error()
				}
			}
			s.Startup[spec.Name] = ServiceStartup{Status: status, Error: message, At: time.Now()}
			if err != nil {
				s.Failures = append(s.Failures, spec.Name+": "+message)
			}
			return nil
		}); updateErr != nil {
			return updateErr
		}
		if err != nil {
			if spec.Core || ctx.Err() != nil {
				return err
			}
			current, stateErr := m.Status()
			if stateErr != nil {
				return stateErr
			}
			for _, candidate := range specs {
				if candidate.Core {
					if _, exited := current.ExitCodes[candidate.Name]; exited {
						return fmt.Errorf("core %s exited during startup: %w", candidate.Name, err)
					}
				}
			}
			cleanup, cancel := context.WithTimeout(context.Background(), spec.ShutdownTimeout+time.Second)
			stopErr := m.stopService(cleanup, spec.Name)
			cancel()
			if stopErr != nil {
				return errors.Join(err, stopErr)
			}
			fmt.Printf("%s failed; retained core and continuing other businesses: %v\n", spec.Name, err)
		}
	}
	if !coreAnnounced {
		if err := m.markCore(specs); err != nil {
			return err
		}
	}
	return m.Update(func(s *State) error {
		if s.Phase != "starting" {
			return errors.New("instance stopped during startup")
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		for _, candidate := range specs {
			if candidate.Core {
				if _, exited := s.ExitCodes[candidate.Name]; exited {
					return fmt.Errorf("core %s exited during startup", candidate.Name)
				}
			}
		}
		s.Phase = "running"
		s.SelectedReady = true
		for _, spec := range specs {
			if s.Startup[spec.Name].Status != "ready" {
				s.SelectedReady = false
			}
			if _, exited := s.ExitCodes[spec.Name]; exited {
				s.SelectedReady = false
			}
		}
		s.FullStackReady = s.SelectedReady && len(specs) == len(FullStackServices())
		s.Stage = "startup-incomplete"
		if s.SelectedReady {
			s.Stage = "selected-ready"
		}
		if s.FullStackReady {
			s.Stage = "full-stack-ready"
		}
		s.StageHistory = append(s.StageHistory, PhaseEvent{Stage: s.Stage, At: time.Now()})
		fmt.Printf("instance %s: %s; status and logs: %s; Token integration deferred\n", m.Key.Name, s.Stage, m.Key.Dir())
		return nil
	})
}
func (m *Manager) stage(stage string) error {
	return m.Update(func(s *State) error {
		if s.Phase != "starting" {
			return errors.New("instance stopped during startup")
		}
		s.Stage = stage
		s.StageHistory = append(s.StageHistory, PhaseEvent{Stage: stage, At: time.Now()})
		return nil
	})
}
func (m *Manager) markCore(specs []ServiceSpec) error {
	return m.Update(func(s *State) error {
		count := 0
		for _, spec := range specs {
			if !spec.Core {
				continue
			}
			count++
			if s.Startup[spec.Name].Status != "ready" {
				return fmt.Errorf("core %s is not ready", spec.Name)
			}
			if _, exited := s.ExitCodes[spec.Name]; exited {
				return fmt.Errorf("core %s exited during startup", spec.Name)
			}
		}
		s.CoreUsable = count > 0
		if s.CoreUsable {
			fmt.Println("核心可用，业务仍在启动")
		}
		return nil
	})
}
func (m *Manager) startApplication(ctx context.Context, s ServiceSpec, paths map[string]string, env map[string]string, children *sync.WaitGroup) error {

	address := serviceAddress(s.Name, env)
	args := append([]string{}, s.Args...)
	if s.Name != "trader-sync" && s.Name != "ui" {
		_, port, _ := net.SplitHostPort(address)
		args = append(args, "--port", port)
	}
	if s.Name == "ui" {
		_, port, _ := net.SplitHostPort(address)
		args = []string{filepath.Join(m.Key.Checkout, "ui/node_modules/vite/bin/vite.js"), "--host", "127.0.0.1", "--port", port, "--strictPort"}
	}
	cmd := exec.Command(paths[s.Name], args...)
	cmd.Dir = m.Key.Checkout
	if s.Name == "ui" {
		cmd.Dir = filepath.Join(m.Key.Checkout, "ui")
	}
	cmd.Env = EnvironmentFor(env, s.EnvironmentKeys)
	logPath := filepath.Join(m.Key.Dir(), "build-logs-"+NewRunID()+"-"+s.Name+".log")
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
	defer cancel()
	return waitService(ready, m, s.Name, address, env)
}

func waitService(ctx context.Context, m *Manager, name, address string, env map[string]string) error {
	for {
		state, e := m.Status()
		if e != nil {
			return e
		}
		if state.Phase != "starting" {
			return errors.New("instance stopped during readiness")
		}
		for _, selected := range state.Services {
			if selected == name || serviceRegistry()[selected].Core {
				if _, exited := state.ExitCodes[selected]; exited {
					return fmt.Errorf("%s exited during startup", selected)
				}
			}
		}
		probe, cancel := context.WithTimeout(ctx, 5*time.Second)
		e = probeService(probe, name, address, env)
		cancel()
		if err := m.recordProbe(name, e); err != nil {
			return err
		}
		if e == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("%s readiness failed (last probe: %v): %w", name, e, ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
}
func probeService(ctx context.Context, name, address string, env map[string]string) error {
	if serviceRegistry()[name].Readiness == "grpc" {
		credentials := insecure.NewCredentials()
		service := serviceRegistry()[name].HealthService
		if name == "trader-sync" {
			service = "tradersync.internal.v1.TraderSyncService"
			cfg := rpcconfig.Client{Address: address, Transport: env["ATHENA_TRADER_SYNC_GRPC_TRANSPORT"], CAFile: env["ATHENA_TRADER_SYNC_TLS_CA_FILE"], ServerName: env["ATHENA_TRADER_SYNC_TLS_SERVER_NAME"]}
			var err error
			credentials, err = rpcconfig.ClientCredentials(cfg)
			if err != nil {
				return err
			}
		} else if name == "operation-log" {
			cfg := operationlogrpcconfig.Client{Address: address, Transport: envDefault(env, "ATHENA_OPERATION_LOG_GRPC_TRANSPORT", "plaintext"), CAFile: env["ATHENA_OPERATION_LOG_TLS_CA_FILE"], ServerName: env["ATHENA_OPERATION_LOG_TLS_SERVER_NAME"]}
			var err error
			credentials, err = operationlogrpcconfig.ClientCredentials(cfg)
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
	return probeHTTP(ctx, name, address, env)
}

func Status(ctx context.Context, k InstanceKey) (State, error) {
	m := NewManager(k)
	snapshot, err := m.Status()
	if err != nil {
		return snapshot, err
	}
	probes := map[string]ProbeResult{}
	env := map[string]string{}
	data, readErr := os.ReadFile(filepath.Join(k.Dir(), "environment.json"))
	if readErr == nil {
		if err = json.Unmarshal(data, &env); err != nil {
			return snapshot, errors.New("invalid instance health configuration")
		}
	}
	for _, name := range snapshot.Services {
		probe := ProbeResult{At: time.Now()}
		p, ok := snapshot.Processes[name]
		switch {
		case !ok:
			probe.Error = "not started"
		case snapshot.Phase == "stopping" || snapshot.Phase == "stopped" || snapshot.Phase == "cleanup-failed":
			probe.Error = "instance " + snapshot.Phase
		default:
			_, err := VerifyProcess(p)
			if err == nil && readErr != nil {
				err = readErr
			}
			if err == nil {
				bounded, cancel := context.WithTimeout(ctx, 5*time.Second)
				err = probeService(bounded, name, snapshot.Endpoints[name], env)
				cancel()
			}
			probe.Ready = err == nil
			if err != nil {
				probe.Error = err.Error()
			}
		}
		probes[name] = probe
	}
	specs, err := ResolveServices(snapshot.Services)
	if err != nil && len(snapshot.Services) > 0 {
		return snapshot, err
	}
	for _, owner := range selectedSchemas(specs) {
		dsn := env[owner.DSNEnv]
		if dsn == "" {
			continue
		}
		bounded, cancel := context.WithTimeout(ctx, time.Second)
		err := waitDatabase(bounded, dsn)
		cancel()
		probe := ProbeResult{Ready: err == nil, At: time.Now()}
		if err != nil {
			probe.Error = err.Error()
		}
		probes[owner.Name+"-postgres"] = probe
	}
	gateways, gatewayErr := ProbeGateways(ctx, gatewayEnvironment(specs, env))
	address := ""
	if selected(specs, "api-server") {
		address = snapshot.Endpoints["api-server"]
	}
	access := readAccessSettings(ctx, address, env)
	err = m.Update(func(s *State) error {
		if s.RunID != snapshot.RunID {
			return errors.New("run changed during status")
		}
		s.Probes = probes
		s.Health = map[string]string{}
		for name, probe := range probes {
			if s.Phase == "stopped" || s.Phase == "stopping" {
				if _, app := serviceRegistry()[name]; app {
					probe.Ready = false
					probe.Error = "instance " + s.Phase
					s.Probes[name] = probe
				}
			}
			if _, exited := s.ExitCodes[name]; exited {
				probe.Ready = false
				probe.Error = "process exited"
				s.Probes[name] = probe
			}
			s.Health[name] = "unavailable"
			if probe.Ready {
				s.Health[name] = "ready"
			}
		}
		s.SelectedReady = s.Phase == "running" && len(s.Services) > 0
		s.CoreUsable = false
		coreCount := 0
		coreReady := true
		for _, name := range s.Services {
			ready := s.Probes[name].Ready && s.Startup[name].Status == "ready"
			if !ready {
				s.SelectedReady = false
			}
			if serviceRegistry()[name].Core {
				coreCount++
				coreReady = coreReady && ready
			}
		}
		s.CoreUsable = (s.Phase == "running" || s.Phase == "starting") && coreCount > 0 && coreReady
		s.FullStackReady = s.SelectedReady && len(s.Services) == len(FullStackServices())
		s.Gateways = gateways
		s.AccessSettings = access
		s.GatewayError = ""
		if gatewayErr != nil {
			s.GatewayError = gatewayErr.Error()
		}
		snapshot = *s
		return nil
	})
	return snapshot, err
}

func (m *Manager) recordProbe(name string, err error) error {
	return m.Update(func(s *State) error {
		if s.Probes == nil {
			s.Probes = map[string]ProbeResult{}
		}
		if s.Health == nil {
			s.Health = map[string]string{}
		}
		if _, exited := s.ExitCodes[name]; exited {
			err = errors.New("process exited")
		}
		probe := ProbeResult{Ready: err == nil, At: time.Now()}
		s.Health[name] = "ready"
		if err != nil {
			probe.Error = err.Error()
			s.Health[name] = "unavailable"
		}
		s.Probes[name] = probe
		return nil
	})
}
