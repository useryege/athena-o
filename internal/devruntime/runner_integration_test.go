//go:build integration

package devruntime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/useryege/athena/internal/accountstate/store/migrations"
	"github.com/useryege/athena/internal/testutil/pgtest"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestRunnerProcess(t *testing.T) {
	if os.Getenv("ATHENA_TASK9_RUN_HELPER") != "1" {
		return
	}
	var o RunOptions
	if e := json.Unmarshal([]byte(os.Getenv("ATHENA_TASK9_OPTIONS")), &o); e != nil {
		os.Exit(92)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()
	if e := Run(ctx, o); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(93)
	}
	os.Exit(0)
}
func fixturePort(t *testing.T) string {
	t.Helper()
	l, e := net.Listen("tcp", "127.0.0.1:0")
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	_, p, _ := net.SplitHostPort(l.Addr().String())
	return p
}
func runnerOptions(t *testing.T, mode string, services []string, extra string) RunOptions {
	t.Helper()
	checkout, e := filepath.Abs("../..")
	if e != nil {
		t.Fatal(e)
	}
	k, e := NewInstanceKey(checkout, "task9-run-"+NewRunID()[:8])
	if e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(t.TempDir(), "run.env")
	content := "ATHENA_URL=http://localhost:4000\nATHENA_TRADER_SYNC_HTTP_URL=http://127.0.0.1:1\nATHENA_TRADER_SYNC_WSS_URL=ws://127.0.0.1:1\nATHENA_TRADER_SYNC_PROXY_URL=\nATHENA_TRADER_SYNC_CURSOR_HMAC_KEY=task9-fixture-cursor\nATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN=" + strings.Repeat("t", 32) + "\nATHENA_TRADER_SYNC_LISTEN_ADDRESS=127.0.0.1:" + fixturePort(t) + "\nATHENA_SERVER_PORT=" + fixturePort(t) + "\nATHENA_NOTIFICATION_PORT=" + fixturePort(t) + "\nATHENA_SERVER_DISABLE_AUTH=true\nATHENA_JWT_SECRET=" + strings.Repeat("j", 32) + "\n" + extra
	for _, key := range []string{"ATHENA_WALLET_SERVER_ADDRESS", "ATHENA_TOKEN_API_SERVER_ADDRESS", "ATHENA_MARKET_RADAR_SERVER_ADDRESS", "ATHENA_MANAGED_OO_SERVER_ADDRESS", "ATHENA_PROFIT_SHARING_SERVER_ADDRESS", "ATHENA_WORM_MARKETS_SERVER_ADDRESS", "ATHENA_WORM_TRADING_SERVER_ADDRESS", "ATHENA_SPORTS_HISTORY_SERVER_ADDRESS", "ATHENA_SPORTS_LIVE_SERVER_ADDRESS"} {
		content += key + "=127.0.0.1:1\n"
	}
	if e = os.WriteFile(file, []byte(content), 0600); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		ctx, c := context.WithTimeout(context.Background(), 120*time.Second)
		defer c()
		m := NewManager(k)
		if e := m.Stop(ctx); e != nil {
			t.Error(e)
			return
		}
		if mode == "managed" {
			if e := m.Reset(ctx); e != nil {
				t.Error(e)
			}
		}
	})
	return RunOptions{Key: k, Services: services, DBMode: mode, EnvFile: file}
}
func startRunner(t *testing.T, o RunOptions) (*exec.Cmd, <-chan error) {
	t.Helper()
	data, _ := json.Marshal(o)
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunnerProcess$")
	env := EnvironmentFor(environmentMap(os.Environ()), []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN"})
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = append(env, RunIDEnv+"="+NewRunID(), "ATHENA_TASK9_RUN_HELPER=1", "ATHENA_TASK9_OPTIONS="+string(data))
	log, e := os.CreateTemp(t.TempDir(), "supervisor-log-")
	if e != nil {
		t.Fatal(e)
	}
	cmd.Stdout = log
	cmd.Stderr = log
	if e = cmd.Start(); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait(); log.Close() }()
	t.Logf("instance=%s namespace=%s supervisor=%d log=%s", o.Key.Name, o.Key.Namespace, cmd.Process.Pid, log.Name())
	return cmd, done
}
func awaitRunning(t *testing.T, o RunOptions, done <-chan error) {
	t.Helper()
	deadline := time.After(180 * time.Second)
	for {
		state, e := NewManager(o.Key).Status()
		if e == nil && state.Phase == "running" {
			return
		}
		select {
		case e := <-done:
			t.Fatalf("runner exited before ready: %v; state=%+v", e, state)
		case <-deadline:
			t.Fatal("runner readiness timeout")
		case <-time.After(100 * time.Millisecond):
		}
	}
}
func TestRealRunnerTwoInstancesAndPersistentSeed(t *testing.T) {
	ctx, c := context.WithTimeout(context.Background(), 300*time.Second)
	defer c()
	a := runnerOptions(t, "managed", []string{"trader-sync"}, "")
	_, ad := startRunner(t, a)
	awaitRunning(t, a, ad)
	if _, e := Seed(ctx, a.Key, "trader-sync"); e != nil {
		t.Fatal(e)
	}
	before, e := os.ReadFile(filepath.Join(a.Key.Dir(), "development-fixture.json"))
	if e != nil {
		t.Fatal(e)
	}
	b := runnerOptions(t, "managed", []string{"trader-sync"}, "")
	_, bd := startRunner(t, b)
	awaitRunning(t, b, bd)
	if e := Stop(ctx, a.Key); e != nil {
		t.Fatal(e)
	}
	if e := <-ad; e != nil {
		t.Fatal(e)
	}
	status, e := Status(ctx, b.Key)
	if e != nil || status.Health["trader-sync"] != "ready" {
		t.Fatal("stopping A affected B", e, status.Health)
	}
	_, ad = startRunner(t, a)
	awaitRunning(t, a, ad)
	if _, e = Seed(ctx, a.Key, "trader-sync"); e != nil {
		t.Fatal(e)
	}
	after, e := os.ReadFile(filepath.Join(a.Key.Dir(), "development-fixture.json"))
	if e != nil || string(before) != string(after) {
		t.Fatal("seed identities lost on restart", e)
	}
	if e = Stop(ctx, a.Key); e != nil {
		t.Fatal(e)
	}
	<-ad
	if e = Stop(ctx, b.Key); e != nil {
		t.Fatal(e)
	}
	<-bd
}
func TestExternalVerificationFailureCreatesNoChildrenOrDatabase(t *testing.T) {
	db := pgtest.NewUnmigrated(t)
	o := runnerOptions(t, "external", []string{"trader-sync"}, "ATHENA_ACCOUNT_STATE_POSTGRES_DSN="+db.DSN+"\n")
	_, done := startRunner(t, o)
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("unmigrated external accepted")
		}
	case <-time.After(120 * time.Second):
		t.Fatal("external verify timeout")
	}
	s, e := NewManager(o.Key).Status()
	if e != nil {
		t.Fatal(e)
	}
	if selectedServiceExited(s) || s.Processes["trader-sync"].PID != 0 || len(s.Resources) != 0 {
		t.Fatal("external verify created resources")
	}
	var absent bool
	if e = db.Pool.QueryRow(context.Background(), "SELECT to_regclass('goose_db_version') IS NULL").Scan(&absent); e != nil || !absent {
		t.Fatal("external verify performed DDL", e)
	}
	if e = Reset(context.Background(), o.Key); e == nil {
		t.Fatal("external reset allowed")
	}
}

func TestRealRunnerTraderSyncFatalKeepsAPIAndNotification(t *testing.T) {
	telegram := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		result := `true`
		switch filepath.Base(r.URL.Path) {
		case "getMe":
			result = `{"id":99,"is_bot":true,"first_name":"Fixture","username":"fixture_bot"}`
		case "getMyName":
			result = `{"name":"Fixture"}`
		case "getMyDescription":
			result = `{"description":"Fixture"}`
		case "getMyShortDescription":
			result = `{"short_description":"Fixture"}`
		case "getWebhookInfo":
			result = `{"url":"","has_custom_certificate":false,"pending_update_count":0}`
		case "getUpdates":
			time.Sleep(100 * time.Millisecond)
			result = `[]`
		}
		fmt.Fprintf(w, `{"ok":true,"result":%s}`, result)
	}))
	defer telegram.Close()
	o := runnerOptions(t, "managed", []string{"api-server", "trader-sync", "notification"}, "ATHENA_NOTIFICATION_TELEGRAM_API_URL="+telegram.URL+"\nATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN=123:task9-fixture\nATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID=-1001\nATHENA_NOTIFICATION_PROD_TELEGRAM_CHAT_ID=-1002\n")
	_, done := startRunner(t, o)
	awaitRunning(t, o, done)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	state, e := Status(ctx, o.Key)
	if e != nil {
		t.Fatal(e)
	}
	url := "http://" + state.Endpoints["api-server"] + "/api/v1/app/bootstrap"
	response, e := http.Get(url)
	if e != nil {
		t.Fatal(e)
	}
	response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("bootstrap", response.StatusCode)
	}
	if e = SignalProcess(state.Processes["trader-sync"], syscall.SIGKILL); e != nil {
		t.Fatal(e)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		state, e = Status(ctx, o.Key)
		if e != nil {
			t.Fatal(e)
		}
		if _, ok := state.ExitCodes["trader-sync"]; ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("fatal exit not recorded")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if state.Phase != "running" || state.Health["api-server"] != "ready" || state.Health["notification"] != "ready" {
		t.Fatalf("fatal cascaded: %s %v", state.Phase, state.Health)
	}
	if e = Stop(ctx, o.Key); e != nil {
		t.Fatal(e)
	}
	if e = <-done; e != nil {
		t.Fatal(e)
	}
}

func TestInitialTraderSyncExitRollsBackBatchAndPreservesEvidence(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	telegram := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-r.Context().Done():
		}
		fmt.Fprint(w, `{"ok":false,"description":"fixture stopped"}`)
	}))
	defer telegram.Close()
	defer close(release)
	o := runnerOptions(t, "managed", []string{"trader-sync", "notification"}, "ATHENA_NOTIFICATION_TELEGRAM_API_URL="+telegram.URL+"\nATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN=123:task9-fixture\nATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID=-1001\nATHENA_NOTIFICATION_PROD_TELEGRAM_CHAT_ID=-1002\n")
	_, done := startRunner(t, o)
	select {
	case <-entered:
	case <-time.After(120 * time.Second):
		t.Fatal("notification did not reach fixture")
	}
	s, e := NewManager(o.Key).Status()
	if e != nil {
		t.Fatal(e)
	}
	if e = SignalProcess(s.Processes["trader-sync"], syscall.SIGKILL); e != nil {
		t.Fatal(e)
	}
	select {
	case e := <-done:
		if e == nil {
			t.Fatal("initial service failure reported success")
		}
	case <-time.After(90 * time.Second):
		t.Fatal("initial rollback did not finish")
	}
	s, e = NewManager(o.Key).Status()
	if e != nil {
		t.Fatal(e)
	}
	if s.Phase != "stopped" || len(s.Failures) == 0 || len(s.Resources) != 2 {
		t.Fatalf("rollback evidence %s %v %v", s.Phase, s.Failures, s.Resources)
	}
	for _, p := range s.Processes {
		if _, e = VerifyProcess(p); !os.IsNotExist(e) {
			t.Fatal("child survived rollback", e)
		}
	}
	for _, path := range s.Logs {
		if _, e = os.Stat(path); e != nil {
			t.Fatal("logs lost", e)
		}
	}
}

func TestExternalHealthyBorrowerStopLeavesDatabaseRunning(t *testing.T) {
	db := pgtest.New(t, migrations.FS, migrations.Dir)
	o := runnerOptions(t, "external", []string{"trader-sync"}, "ATHENA_ACCOUNT_STATE_POSTGRES_DSN="+db.DSN+"\n")
	_, done := startRunner(t, o)
	awaitRunning(t, o, done)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	if _, e := Seed(ctx, o.Key, "trader-sync"); e == nil {
		t.Fatal("external seed permitted")
	}
	if e := Stop(ctx, o.Key); e != nil {
		t.Fatal(e)
	}
	if e := <-done; e != nil {
		t.Fatal(e)
	}
	if e := db.Pool.Ping(ctx); e != nil {
		t.Fatal("borrower stop affected external database", e)
	}
	s, e := NewManager(o.Key).Status()
	if e != nil || len(s.Resources) != 0 {
		t.Fatal("borrowed database owned", e, s.Resources)
	}
}
func TestAPIOnlyRunsWithBrokenCollectorConfiguration(t *testing.T) {
	o := runnerOptions(t, "managed", []string{"api-server"}, "ATHENA_TRADER_SYNC_HTTP_URL=broken\nATHENA_TRADER_SYNC_WSS_URL=broken\nATHENA_TRADER_SYNC_CURSOR_HMAC_KEY_FILE=/missing-cursor-file\n")
	_, done := startRunner(t, o)
	awaitRunning(t, o, done)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	s, e := Status(ctx, o.Key)
	if e != nil || s.Health["api-server"] != "ready" || s.Processes["trader-sync"].PID != 0 || s.Processes["notification"].PID != 0 {
		t.Fatal(s.Health, e)
	}
	data, e := os.ReadFile("/proc/" + strconv.Itoa(s.Processes["api-server"].PID) + "/environ")
	if e != nil {
		t.Fatal(e)
	}
	for _, forbidden := range []string{"ATHENA_TRADER_SYNC_HTTP_URL=", "ATHENA_TRADER_SYNC_WSS_URL=", "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY"} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatal("API received collector config", forbidden)
		}
	}
	if e = Stop(ctx, o.Key); e != nil {
		t.Fatal(e)
	}
	<-done
}

func TestUIOnlyUsesNodeAndNoDatabase(t *testing.T) {
	o := runnerOptions(t, "external", []string{"ui"}, "ATHENA_UI_PORT="+fixturePort(t)+"\n")
	_, done := startRunner(t, o)
	awaitRunning(t, o, done)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	s, e := Status(ctx, o.Key)
	if e != nil || s.Health["ui"] != "ready" || len(s.Resources) != 0 || len(s.Processes) != 1 {
		t.Fatal(s, e)
	}
	if !strings.Contains(s.Processes["ui"].Exe, "node") {
		t.Fatal("UI did not exec explicit node")
	}
	if e = Stop(ctx, o.Key); e != nil {
		t.Fatal(e)
	}
	<-done
}
