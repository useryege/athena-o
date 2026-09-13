//go:build integration

package acceptance

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	et "github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/useryege/athena/internal/devruntime"
	"github.com/useryege/athena/internal/testutil/pgtest"
	trpc "github.com/useryege/athena/internal/tradersync/apiclient"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"google.golang.org/grpc"
	hp "google.golang.org/grpc/health/grpc_health_v1"
)

// TestIndependentRuntimeLifecycle catches lost persistent state, hidden in-process
// runtime substitution, WSS-dependent RPC readiness, and historical backfill on
// restart. It requires explicit opt-in because it exercises public Make commands,
// creates a dedicated managed PostgreSQL instance and builds real executables.
func TestIndependentRuntimeLifecycle(t *testing.T) {
	if os.Getenv("ATHENA_INDEPENDENT_ACCEPTANCE") != "1" {
		t.Skip("run make trader-sync-acceptance")
	}
	root, err := filepath.Abs("../../..")
	require.NoError(t, err)
	base := os.Getenv("ATHENA_INDEPENDENT_INSTANCE")
	if base == "" {
		base = "ts-controlled-acceptance"
	}
	key, err := devruntime.NewInstanceKey(root, base+"-"+uuid.NewString()[:8])
	require.NoError(t, err)
	_, err = os.Stat(key.Dir())
	require.True(t, os.IsNotExist(err), "acceptance must never reuse an existing instance")
	artifacts := os.Getenv("ATHENA_INDEPENDENT_ARTIFACTS")
	require.NotEmpty(t, artifacts)
	require.NoError(t, os.MkdirAll(artifacts, 0700))
	fixture, blocked, proxy, ca := independentSources(t, artifacts)
	defer fixture.Close()
	port, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	target := port.Addr().String()
	require.NoError(t, port.Close())
	envFile := filepath.Join(artifacts, "controlled.env")
	require.NoError(t, os.WriteFile(envFile, []byte("ATHENA_URL=http://localhost:4000\nATHENA_TRADER_SYNC_HTTP_URL="+fixture.chainHTTP.URL+"\nATHENA_TRADER_SYNC_WSS_URL=ws"+strings.TrimPrefix(fixture.wsHTTP.URL, "http")+"\nATHENA_TRADER_SYNC_PROXY_URL="+proxy+"\nSSL_CERT_FILE="+ca+"\nATHENA_TRADER_SYNC_CURSOR_HMAC_KEY=controlled-independent-cursor-key\nATHENA_TRADER_SYNC_LISTEN_ADDRESS="+target+"\n"), 0600))
	command := func(name string) *exec.Cmd {
		cmd := exec.Command("make", "--no-print-directory", name, "INSTANCE="+key.Name, "SERVICE=trader-sync", "ENV_FILE="+envFile)
		cmd.Dir = root
		// Keep toolchain and Docker transport configuration, never inherit unrelated
		// application settings that could override the dedicated fixture env file.
		for _, v := range os.Environ() {
			if !strings.HasPrefix(v, "ATHENA_") && !strings.HasPrefix(v, "SSL_CERT_") && !strings.HasPrefix(v, "DB_MODE=") {
				cmd.Env = append(cmd.Env, v)
			}
		}
		return cmd
	}
	invoke := func(name string) []byte {
		t.Helper()
		log, e := os.Create(filepath.Join(artifacts, name+"-"+uuid.NewString()[:8]+".log"))
		require.NoError(t, e)
		defer log.Close()
		cmd := command(name)
		cmd.Stderr = log
		out, e := cmd.Output()
		require.NoError(t, e, "%s: inspect %s", name, log.Name())
		t.Logf("make %s INSTANCE=%s exit=0", name, key.Name)
		return out
	}
	t.Cleanup(func() {
		out, e := command("stop-instance").CombinedOutput()
		if e != nil {
			t.Errorf("owned cleanup stop: %v %s", e, out)
			return
		}
		if !t.Failed() {
			out, e = command("reset-instance").CombinedOutput()
			if e != nil {
				t.Errorf("owned cleanup reset: %v %s", e, out)
			}
		}
		t.Logf("controlled instance %s stopped; reset=%v; logs preserved", key.Name, !t.Failed())
	})
	start := func(label string) <-chan error {
		t.Helper()
		log, e := os.Create(filepath.Join(artifacts, label+"-run.log"))
		require.NoError(t, e)
		cmd := command("run-service")
		cmd.Stdout = log
		cmd.Stderr = log
		require.NoError(t, cmd.Start())
		done := make(chan error, 1)
		go func() { done <- cmd.Wait(); log.Close() }()
		require.Eventually(t, func() bool { s, e := devruntime.LoadState(key.StatePath()); return e == nil && s.Phase == "running" }, 180*time.Second, 100*time.Millisecond, "instance startup; inspect %s", log.Name())
		t.Logf("instance=%s namespace=%s make_pid=%d log=%s", key.Name, key.Namespace, cmd.Process.Pid, log.Name())
		return done
	}
	done := start("initial")
	var state devruntime.State
	require.NoError(t, json.Unmarshal(invoke("runtime-status"), &state))
	independentOwnership(t, state)
	independentJSON(t, artifacts, "initial-state", state)
	var seed devruntime.DevelopmentFixture
	require.NoError(t, json.Unmarshal(invoke("seed-service"), &seed))
	require.NotEmpty(t, seed.MemberID)
	require.NotEmpty(t, seed.AdministratorID)
	var runtimeEnv map[string]string
	secret, e := os.ReadFile(filepath.Join(key.Dir(), "environment.json"))
	require.NoError(t, e)
	require.NoError(t, json.Unmarshal(secret, &runtimeEnv))
	pool, e := pgxpool.New(context.Background(), runtimeEnv["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"])
	require.NoError(t, e)
	defer func() { pool.Close() }()
	fixture.db = &pgtest.DB{Pool: pool}
	// The acceptance owns this synthetic Telegram binding; the public seed tool
	// intentionally creates identities and grants only. No sender is running.
	_, e = pool.Exec(fixture.ctx, `INSERT INTO telegram_bindings(account_id,telegram_user_id,telegram_chat_id,telegram_display_name,revision)VALUES($1,120012,120012,'independent acceptance',1)`, seed.MemberID)
	require.NoError(t, e)
	var dbName string
	var dbOID uint32
	require.NoError(t, pool.QueryRow(fixture.ctx, "SELECT current_database(), oid FROM pg_database WHERE datname=current_database()").Scan(&dbName, &dbOID))
	require.Equal(t, "athena", dbName)
	token, e := os.ReadFile(filepath.Join(key.Dir(), "internal-token"))
	require.NoError(t, e)
	client, closer, e := trpc.NewClient(rpcconfig.Client{Address: target, Transport: "loopback-insecure", Token: string(token), MaxMessageBytes: rpcconfig.DefaultMaxMessageBytes})
	require.NoError(t, e)
	defer closer.Close()
	member := &trpc.Actor{AccountId: seed.MemberID, Realm: trpc.ApplicationRealm_APPLICATION_REALM_MEMBER}
	admin := &trpc.Actor{AccountId: seed.AdministratorID, Realm: trpc.ApplicationRealm_APPLICATION_REALM_ADMIN}
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	get := func(id string) *trpc.Subscription {
		t.Helper()
		r, e := client.GetSubscription(ctx, &trpc.GetSubscriptionRequest{Actor: member, SubscriptionId: id})
		require.NoError(t, e)
		return r.Subscription
	}
	create := func(wallet common.Address) *trpc.Subscription {
		t.Helper()
		r, e := client.ResolveTarget(ctx, &trpc.ResolveTargetRequest{Actor: member, Input: wallet.Hex()})
		require.NoError(t, e)
		c, e := client.CreateSubscription(ctx, &trpc.CreateSubscriptionRequest{Actor: member, ConfirmationToken: r.Target.ConfirmationToken, RequestId: uuid.NewString()})
		require.NoError(t, e)
		return c.Subscription
	}
	ready := func() {
		t.Helper()
		conn, e := grpc.Dial(target, grpc.WithInsecure())
		require.NoError(t, e)
		defer conn.Close()
		hc, stop := context.WithTimeout(ctx, 3*time.Second)
		defer stop()
		r, e := hp.NewHealthClient(conn).Check(hc, &hp.HealthCheckRequest{Service: "tradersync.internal.v1.TraderSyncService"})
		require.NoError(t, e)
		require.Equal(t, hp.HealthCheckResponse_SERVING, r.Status)
	}
	wallet := common.BigToAddress(big.NewInt(1))
	sub := create(wallet)
	require.Eventually(t, func() bool { return get(sub.Id).Observation.State == "healthy" }, 15*time.Second, 50*time.Millisecond)
	fixture.Advance(1100 * time.Millisecond)
	fixture.Push(fixture.Replay(wallet, 0))
	require.Eventually(t, func() bool {
		var n int
		e := pool.QueryRow(ctx, "SELECT count(*) FROM trader_sync_activities").Scan(&n)
		return e == nil && n == 1
	}, 15*time.Second, 50*time.Millisecond)
	var deliveries int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM account_notification_deliveries WHERE status='pending'").Scan(&deliveries))
	require.Equal(t, 1, deliveries, "Notification absent must leave durable work")
	initial := get(sub.Id)
	require.NotNil(t, initial.CurrentInterval)
	oldEpoch := initial.CurrentInterval.Epoch
	independentJSON(t, artifacts, "initial-subscription", initial)
	invoke("stop-instance")
	select {
	case e := <-done:
		require.NoError(t, e)
	case <-time.After(40 * time.Second):
		t.Fatal("Make run did not join after stop")
	}
	require.Eventually(t, func() bool { _, e := devruntime.VerifyProcess(state.Processes["trader-sync"]); return e != nil }, 5*time.Second, 20*time.Millisecond)
	gapLog := fixture.Replay(wallet, 0) // A fact observed only by the provider during downtime.
	done = start("restart")
	var restartedState devruntime.State
	require.NoError(t, json.Unmarshal(invoke("runtime-status"), &restartedState))
	independentOwnership(t, restartedState)
	require.NotEqual(t, state.RunID, restartedState.RunID)
	require.NotEqual(t, state.Processes["trader-sync"].PID, restartedState.Processes["trader-sync"].PID)
	independentJSON(t, artifacts, "restarted-state", restartedState)
	// Docker assigns a fresh host port when recreating the same persisted PG.
	// Reconnect using the new owned instance configuration, then compare DB OID.
	pool.Close()
	secret, e = os.ReadFile(filepath.Join(key.Dir(), "environment.json"))
	require.NoError(t, e)
	require.NoError(t, json.Unmarshal(secret, &runtimeEnv))
	pool, e = pgxpool.New(ctx, runtimeEnv["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"])
	require.NoError(t, e)
	fixture.db = &pgtest.DB{Pool: pool}
	var secondSeed devruntime.DevelopmentFixture
	require.NoError(t, json.Unmarshal(invoke("seed-service"), &secondSeed))
	require.Equal(t, seed, secondSeed)
	require.Eventually(t, func() bool {
		s := get(sub.Id)
		return s.Observation.State == "healthy" && s.CurrentInterval != nil && s.CurrentInterval.Epoch > oldEpoch
	}, 20*time.Second, 50*time.Millisecond)
	resumed := get(sub.Id)
	require.NotNil(t, resumed.Observation.LatestInterruption)
	require.NotEqual(t, "0", resumed.Observation.InterruptionCount)
	var afterOID uint32
	require.NoError(t, pool.QueryRow(ctx, "SELECT oid FROM pg_database WHERE datname=current_database()").Scan(&afterOID))
	require.Equal(t, dbOID, afterOID)
	var activities int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM trader_sync_activities").Scan(&activities))
	require.Equal(t, 1, activities)
	var gapSources int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM trader_sync_source_records WHERE transaction_hash=$1", gapLog.TxHash.Bytes()).Scan(&gapSources))
	require.Zero(t, gapSources, "a provider-only gap fact must not be historically imported")
	independentJSON(t, artifacts, "restarted-subscription", resumed)
	// A blocked WSS handshake leaves the business service ready. Both creation and
	// resumption commit pending baseline; neither invents a monitoring interval.
	blocked.Store(true)
	fixture.mu.Lock()
	sockets := append([]*socket(nil), fixture.sockets...)
	fixture.mu.Unlock()
	for _, s := range sockets {
		s.conn.Close()
	}
	require.Eventually(t, func() bool { return get(sub.Id).Observation.State != "healthy" }, 8*time.Second, 50*time.Millisecond)
	ready()
	pending := create(common.BigToAddress(big.NewInt(2)))
	require.Equal(t, "pending_baseline", pending.Observation.State)
	require.Nil(t, pending.CurrentInterval)
	paused, e := client.PauseSubscription(ctx, &trpc.PauseSubscriptionRequest{Actor: member, SubscriptionId: sub.Id, ExpectedRevision: get(sub.Id).Revision, RequestId: uuid.NewString()})
	require.NoError(t, e)
	resume, e := client.ResumeSubscription(ctx, &trpc.ResumeSubscriptionRequest{Actor: member, SubscriptionId: sub.Id, ExpectedRevision: paused.Subscription.Revision, RequestId: uuid.NewString()})
	require.NoError(t, e)
	require.Equal(t, "pending_baseline", resume.Subscription.Observation.State)
	require.Nil(t, resume.Subscription.CurrentInterval)
	independentJSON(t, artifacts, "wss-outage", map[string]any{"created": pending, "resumed": resume.Subscription, "health": "SERVING"})
	blocked.Store(false)
	require.Eventually(t, func() bool {
		return get(sub.Id).Observation.State == "healthy" && get(pending.Id).Observation.State == "healthy"
	}, 40*time.Second, 100*time.Millisecond)
	status, e := client.GetTraderSyncRuntimeStatus(ctx, &trpc.GetTraderSyncRuntimeStatusRequest{Actor: admin})
	require.NoError(t, e)
	independentJSON(t, artifacts, "final-runtime", status)
	fixture.mu.Lock()
	ranges, sideEffects := fixture.rangeCalls, fixture.sideEffects
	costs := map[string]int{}
	for k, v := range fixture.costs {
		costs[k] = v
	}
	fixture.mu.Unlock()
	require.Zero(t, ranges, "restart must not query historical ranges")
	require.Zero(t, sideEffects, "fixture refuses undeclared provider requests")
	independentJSON(t, artifacts, "evidence", map[string]any{"instance": key, "database": dbName, "databaseOID": dbOID, "seed": seed, "activities": activities, "pendingDeliveriesWithoutNotification": deliveries, "historicalRangeCalls": ranges, "costs": costs})
	invoke("stop-instance")
	select {
	case e := <-done:
		require.NoError(t, e)
	case <-time.After(40 * time.Second):
		t.Fatal("second Make run did not join")
	}
}

func independentJSON(t *testing.T, dir, name string, value any) {
	t.Helper()
	b, e := json.MarshalIndent(value, "", "  ")
	require.NoError(t, e)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name+".json"), b, 0600))
}

func independentOwnership(t *testing.T, s devruntime.State) {
	t.Helper()
	require.Equal(t, []string{"trader-sync"}, s.Services)
	require.Equal(t, "ready", s.Health["trader-sync"])
	for name, p := range s.Processes {
		if name == "trader-sync" {
			actual, e := devruntime.VerifyProcess(p)
			require.NoError(t, e)
			require.True(t, devruntime.SameProcess(p, actual))
			cwd, e := os.Readlink(fmt.Sprintf("/proc/%d/cwd", p.PID))
			require.NoError(t, e)
			require.Equal(t, s.Key.Checkout, cwd)
			require.Contains(t, p.Exe, s.Key.Dir())
		}
	}
	count := 0
	for _, r := range s.Resources {
		require.Equal(t, s.Key.Namespace, r.Namespace)
		require.True(t, r.Owned)
		args := []string{r.Kind, "inspect", "--format", "{{json .Labels}}", r.ID}
		if r.Kind == "container" {
			count++
			args[3] = "{{json .Config.Labels}}"
		}
		b, e := exec.Command("docker", args...).Output()
		require.NoError(t, e)
		var labels map[string]string
		require.NoError(t, json.Unmarshal(b, &labels))
		for key, value := range devruntime.ResourceLabels(s.Key, r.Kind, r.RunID, r.Name) {
			require.Equal(t, value, labels[key])
		}
		require.Contains(t, r.Name, "postgres")
	}
	require.Equal(t, 1, count, "standalone TS starts only its PostgreSQL container")
}

// Only slow external source boundaries are substituted. No Service, Collector,
// Projector or Notification runtime is constructed in this fixture process.
func independentSources(t *testing.T, dir string) (*harness, *atomic.Bool, string, string) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	h := &harness{t: t, ctx: ctx, cancel: cancel, headers: map[common.Hash]*et.Header{}, numbers: map[uint64]*et.Header{}, receipts: map[common.Hash]*et.Receipt{}, code: map[common.Address]string{}, costs: map[string]int{}, telegramUpdates: make(chan []any, 10)}
	b, e := os.ReadFile("../testdata/source_records.json")
	require.NoError(t, e)
	require.NoError(t, json.Unmarshal(b, &h.fixtures))
	for address, file := range map[string]string{"0xe111180000d2663c0091e4f400237545b87b996b": "core-exchange.hex", "0xe2222d279d744050d28e00520010520000310f59": "neg-risk-exchange.hex", "0xe3333700ca9d93003f00f0f71f8515005f6c00aa": "combo-proxy.hex", "0x641b40ec414a076b9e79e703fc7bf4ebec248bb7": "combo-implementation.hex"} {
		b, e := os.ReadFile(filepath.Join("../testdata/runtime", file))
		require.NoError(t, e)
		h.code[common.HexToAddress(address)] = strings.TrimSpace(string(b))
	}
	h.tip = &et.Header{Number: big.NewInt(10000), Difficulty: big.NewInt(0), Time: uint64(time.Now().Unix()), Extra: []byte("independent runtime replay")}
	h.headers[h.tip.Hash()] = h.tip
	h.numbers[10000] = h.tip
	h.chainHTTP = httptest.NewServer(http.HandlerFunc(h.serveRPC))
	blocked := new(atomic.Bool)
	h.wsHTTP = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if blocked.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		h.serveWSS(w, r)
	}))
	// Trust only the fixture CA in the child, and accept only known provider hosts.
	key, e := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, e)
	cert := &x509.Certificate{SerialNumber: big.NewInt(12), Subject: pkix.Name{CommonName: "independent acceptance"}, DNSNames: []string{"*.polymarket.com", "polymarket.com"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, e := x509.CreateCertificate(rand.Reader, cert, cert, &key.PublicKey, key)
	require.NoError(t, e)
	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	ca := filepath.Join(dir, "provider-ca.pem")
	require.NoError(t, os.WriteFile(ca, pemCert, 0600))
	pair, e := tls.X509KeyPair(pemCert, pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	require.NoError(t, e)
	h.profileHTTP = httptest.NewUnstartedServer(http.HandlerFunc(h.serveProfile))
	h.profileHTTP.TLS = &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}
	h.profileHTTP.StartTLS()
	chainAddr := strings.TrimPrefix(h.chainHTTP.URL, "http://")
	wsAddr := strings.TrimPrefix(h.wsHTTP.URL, "http://")
	profileAddr := strings.TrimPrefix(h.profileHTTP.URL, "https://")
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "CONNECT" {
			if r.URL.Host != chainAddr {
				t.Errorf("undeclared proxy destination %s", r.URL.Host)
				w.WriteHeader(502)
				return
			}
			h.serveRPC(w, r)
			return
		}
		target := ""
		host, _, _ := net.SplitHostPort(r.Host)
		if r.Host == wsAddr {
			target = wsAddr
		} else {
			switch host {
			case "gamma-api.polymarket.com", "data-api.polymarket.com", "user-pnl-api.polymarket.com", "polymarket.com", "clob.polymarket.com", "combos-rfq-api.polymarket.com":
				target = profileAddr
			}
		}
		if target == "" {
			t.Errorf("undeclared CONNECT destination %s", r.Host)
			w.WriteHeader(502)
			return
		}
		upstream, e := net.DialTimeout("tcp", target, 3*time.Second)
		if e != nil {
			w.WriteHeader(502)
			return
		}
		client, buffer, e := w.(http.Hijacker).Hijack()
		if e != nil {
			upstream.Close()
			return
		}
		fmt.Fprint(buffer, "HTTP/1.1 200 Connection Established\r\n\r\n")
		buffer.Flush()
		go func() {
			defer client.Close()
			defer upstream.Close()
			go io.Copy(upstream, buffer)
			io.Copy(client, upstream)
		}()
	}))
	t.Cleanup(proxy.Close)
	t.Cleanup(h.Close)
	return h, blocked, proxy.URL, ca
}
