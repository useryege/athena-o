package tradersync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"google.golang.org/grpc"
)

func runtimeTestConfig() Config {
	return Config{AccountStateDSN: "postgres://fixture@127.0.0.1/test", ShutdownTimeout: time.Second, HTTPURL: "http://127.0.0.1:1", ProxyURL: "http://127.0.0.1:1", WebSocketURL: "ws://127.0.0.1:1", SiteURL: "https://athena.test", CursorHMACKey: "cursor-independent-test-key"}
}
func runtimeTestRPC() rpcconfig.Server {
	return rpcconfig.Server{ListenAddress: "127.0.0.1:8122", Transport: "loopback-insecure", Token: "0123456789abcdef0123456789abcdef", MaxMessageBytes: 1024}
}
func runtimeTestDeps() RuntimeDependencies {
	return RuntimeDependencies{NewRPCServer: func(*Service, func() bool) *grpc.Server { return grpc.NewServer() }}
}
func TestRuntimeRejectsProcessConfigurationBeforeResourceCreation(t *testing.T) {
	for _, change := range []func(*Config){func(c *Config) { c.AccountStateDSN = "" }, func(c *Config) { c.ShutdownTimeout = -1 }, func(c *Config) { c.HTTPURL = "bad" }, func(c *Config) { c.CursorHMACKey = runtimeTestRPC().Token }} {
		cfg := runtimeTestConfig()
		change(&cfg)
		_, err := NewRuntime(cfg, runtimeTestRPC(), runtimeTestDeps())
		require.Error(t, err)
	}
	cfg := runtimeTestConfig()
	cfg.AccountStateDSN = ""
	require.NoError(t, cfg.Validate(), "borrowed domain components do not own process configuration")
}
func TestRuntimeFatalMakesUnreadyBeforeBlockedWorkerJoins(t *testing.T) {
	r, err := NewRuntime(runtimeTestConfig(), runtimeTestRPC(), runtimeTestDeps())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	blocked, entered := make(chan struct{}), make(chan struct{})
	fatal := errors.New("projector failed")
	r.startWorkers(ctx, []runtimeWorker{{"collector", func(context.Context) error { close(entered); <-blocked; return nil }}, {"projector", func(context.Context) error { <-entered; return fatal }}})
	require.ErrorIs(t, r.Wait(), fatal)
	require.False(t, r.Ready())
	short, stop := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer stop()
	require.ErrorIs(t, r.Shutdown(short), context.DeadlineExceeded)
	select {
	case <-r.closed:
		t.Fatal("cleanup completed before worker joined")
	default:
	}
	close(blocked)
	require.NoError(t, r.Shutdown(context.Background()))
}

func TestRuntimeRejectsWorkerLimitsBeforeOpeningResources(t *testing.T) {
	for _, change := range []func(*Config){func(c *Config) { c.Projector.Interval = -1 }, func(c *Config) { c.Projector.MetadataWait = 3 * time.Second }, func(c *Config) { c.Projector.MetadataTimeout = -1 }, func(c *Config) { c.Projector.MaxInFlightSources = -1 }, func(c *Config) { c.ReconnectMin = -1 }, func(c *Config) { c.ReconnectMin = 2 * time.Second; c.ReconnectMax = time.Second }} {
		cfg := runtimeTestConfig()
		change(&cfg)
		_, err := NewRuntime(cfg, runtimeTestRPC(), runtimeTestDeps())
		require.Error(t, err)
	}
}
func TestRuntimeShutdownBeforeStartPreventsStart(t *testing.T) {
	r, err := NewRuntime(runtimeTestConfig(), runtimeTestRPC(), runtimeTestDeps())
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	require.NoError(t, r.Shutdown(ctx))
	require.Error(t, r.Start(context.Background()))
	require.False(t, r.Ready())
}

func TestRuntimeShutdownLogFieldsNeverWaitForWorkerLocks(t *testing.T) {
	t.Setenv("ATHENA_LOCAL_RUNTIME_INSTANCE", "cached-instance")
	t.Setenv("ATHENA_LOCAL_RUNTIME_RUN_ID", "cached-run")
	r, err := NewRuntime(runtimeTestConfig(), runtimeTestRPC(), runtimeTestDeps())
	require.NoError(t, err)
	// Even a stuck holder of both lifecycle and collector locks must not stop
	// the command's final diagnostic read or postpone its process deadline.
	collector := &Collector{}
	r.service = &Service{deps: Dependencies{Collector: collector}}
	r.mu.Lock()
	defer r.mu.Unlock()
	collector.mu.Lock()
	defer collector.mu.Unlock()
	fields := make(chan RuntimeLogFields, 1)
	go func() { fields <- r.ShutdownLogFields() }()
	select {
	case snapshot := <-fields:
		require.Equal(t, "cached-instance", snapshot.Instance)
		require.Equal(t, "cached-run", snapshot.Run)
		require.Equal(t, "unknown", snapshot.RuntimeGeneration)
		require.Equal(t, "unknown", snapshot.CollectorEpoch)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("watchdog diagnostics blocked on a worker lock")
	}
}
