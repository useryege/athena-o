package serviceschema_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	accountschema "github.com/useryege/athena/internal/accountstate/schema"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// Exercise the actual independent executables. All network dependencies point
// at a local fixture/proxy; no remote service or shared database is contacted.
func TestStandaloneProcessesVerifyAndStop(t *testing.T) {
	if os.Getenv("SERVICE_SCHEMA_TEST_POSTGRES_DSN") == "" {
		t.Skip("requires isolated PostgreSQL")
	}
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	bins := t.TempDir()
	names := []string{"wallet", "etherscan-manager", "market-radar", "managed-oo", "profit-sharing", "solana-discovery"}
	args := []string{"build", "-o", bins}
	for _, name := range names {
		args = append(args, "./cmd/athena-"+name)
	}
	build := exec.Command("go", args...)
	build.Dir = root
	out, err := build.CombinedOutput()
	require.NoError(t, err, string(out))
	t.Log("independent build: six binaries passed")
	for _, testName := range append(names, "profit-sharing-blocked") {
		name := testName
		blocked := name == "profit-sharing-blocked"
		if blocked {
			name = "profit-sharing"
		}
		t.Run(testName, func(t *testing.T) {
			pool, dsn := database(t)
			requests := make(chan struct{}, 8)
			releaseFixture := make(chan struct{})
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				select {
				case requests <- struct{}{}:
				default:
				}
				if name == "solana-discovery" {
					var body struct {
						Method string `json:"method"`
						ID     any    `json:"id"`
					}
					_ = json.NewDecoder(r.Body).Decode(&body)
					if body.Method == "getGenesisHash" {
						_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": body.ID, "result": "5eykt4UsFv8P8NJdTREpY1vzqKqZKvdpKuc147dw2N9d"})
						return
					}
				}
				select {
				case <-r.Context().Done():
				case <-releaseFixture:
				}
			}))
			defer func() { close(releaseFixture); upstream.Close() }()
			env := []string{"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "HTTP_PROXY=" + upstream.URL, "HTTPS_PROXY=" + upstream.URL, "NO_PROXY=127.0.0.1,localhost"}
			envKey := map[string]string{"wallet": "ATHENA_WALLET_POSTGRES_DSN", "managed-oo": "ATHENA_MANAGED_OO_POSTGRES_DSN", "profit-sharing": "ATHENA_PROFIT_SHARING_POSTGRES_DSN", "solana-discovery": "ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN"}[name]
			if envKey != "" {
				env = append(env, envKey+"="+dsn)
			}
			binary := filepath.Join(bins, "athena-"+name)
			runSchema := func(action string) error {
				c := exec.Command(binary, "schema", action, "--timeout=5s")
				c.Env = env
				output, err := c.CombinedOutput()
				t.Logf("schema %s: %s", action, output)
				return err
			}
			if envKey != "" {
				before := catalog(t, pool)
				require.Error(t, runSchema("verify"))
				require.Equal(t, before, catalog(t, pool))
				require.NoError(t, runSchema("up"))
				require.NoError(t, runSchema("verify"))
			}
			env = append(env, "ATHENA_WALLET_ENCRYPTION_KEY=standalone-test-key", "ATHENA_WALLET_INTERNAL_AUTH_TOKEN="+strings.Repeat("w", 32), "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN="+strings.Repeat("x", 32), "ATHENA_SOLANA_DISCOVERY_INTERNAL_AUTH_TOKEN="+strings.Repeat("s", 32), "ATHENA_ETHERSCAN_MANAGER_API_KEYS=test-key", "ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS=127.0.0.1:1", "ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN=test-token")
			l, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			port := l.Addr().(*net.TCPAddr).Port
			require.NoError(t, l.Close())
			args := []string{"--address=127.0.0.1", "--port=" + strconv.Itoa(port)}
			if name == "managed-oo" || name == "market-radar" {
				args = append(args, "--notification-enabled=false")
			}
			if name == "managed-oo" {
				args = append(args, "--polygon-rpc-url="+upstream.URL)
			}
			if name == "solana-discovery" {
				args = append(args, "--rpc-url="+upstream.URL)
				// Own schema alone is insufficient to start the shared-account consumer.
				c := exec.Command(binary, args...)
				c.Env = env
				before := catalog(t, pool)
				out, err := c.CombinedOutput()
				require.Error(t, err)
				require.Contains(t, string(out), "verify Solana account schema")
				require.Equal(t, before, catalog(t, pool))
				require.Empty(t, requests)
				require.NoError(t, accountschema.Up(context.Background(), dsn))
				_, err = pool.Exec(context.Background(), "ALTER TABLE solana_discovery.projects DROP COLUMN symbol")
				require.NoError(t, err)
				c = exec.Command(binary, args...)
				c.Env = env
				before = catalog(t, pool)
				out, err = c.CombinedOutput()
				require.Error(t, err)
				require.Contains(t, string(out), "verify Solana discovery")
				require.Equal(t, before, catalog(t, pool))
				require.Empty(t, requests)
				require.NoError(t, runSchema("up"))

			}
			var logs bytes.Buffer
			c := exec.Command(binary, args...)
			c.Env = env
			c.Stdout = &logs
			c.Stderr = &logs
			require.NoError(t, c.Start())
			exited := make(chan error, 1)
			go func() { exited <- c.Wait() }()
			stopped := false
			defer func() {
				if !stopped {
					_ = c.Process.Kill()
					<-exited
				}
			}()
			client, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", port), grpc.WithTransportCredentials(insecure.NewCredentials()))
			require.NoError(t, err)
			defer client.Close()
			ready := false
			for deadline := time.Now().Add(8 * time.Second); time.Now().Before(deadline); {
				ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
				response, err := healthpb.NewHealthClient(client).Check(ctx, &healthpb.HealthCheckRequest{})
				cancel()
				if err == nil && response.Status == healthpb.HealthCheckResponse_SERVING {
					ready = true
					break
				}
				time.Sleep(25 * time.Millisecond)
			}
			require.True(t, ready, "service never ready")
			if name == "managed-oo" || name == "market-radar" || name == "solana-discovery" {
				select {
				case <-requests:
				case <-time.After(3 * time.Second):
					t.Fatal("background work did not reach local fixture")
				}
			}
			// A healthy long-lived watch is intentionally held open across TERM. It
			// forces the real binary through its bounded transport-stop fallback.
			if blocked {
				stream, err := healthpb.NewHealthClient(client).Watch(context.Background(), &healthpb.HealthCheckRequest{})
				require.NoError(t, err)
				_, err = stream.Recv()
				require.NoError(t, err)
			}
			start := time.Now()
			require.NoError(t, c.Process.Signal(syscall.SIGTERM))
			select {
			case err := <-exited:
				stopped = true
				if blocked {
					require.Error(t, err)
					require.Contains(t, logs.String(), "shutdown exceeded")
				} else {
					require.NoError(t, err, logs.String())
				}
			case <-time.After(35 * time.Second):
				t.Fatal("TERM did not bound exit")
			}
			t.Logf("TERM exit after %s; output: %s", time.Since(start), logs.String())
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 100*time.Millisecond)
			if err == nil {
				conn.Close()
				t.Fatal("listener still accepts after exit")
			}
		})
	}
}
