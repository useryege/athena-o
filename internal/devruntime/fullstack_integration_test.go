//go:build integration

package devruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestFullStackPreparesEveryLegacySchemaBeforeConsumers(t *testing.T) {
	o := runnerOptions(t, "managed", FullStackServices(), "")
	m := NewManager(o.Key)
	s := initialState(t, o.Key)
	s.Phase = "starting"
	s.RunID = NewRunID()
	s.Services = FullStackServices()
	if err := SaveState(o.Key.StatePath(), s); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	dsn, err := m.preparePostgres(ctx)
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"ATHENA_ACCOUNT_STATE_POSTGRES_DSN": dsn}
	if err := m.prepareFullStackDatabases(ctx, env); err != nil {
		t.Fatal(err)
	}
	for _, module := range fullStackModules() {
		pool, err := pgxpool.New(ctx, env[module.DSNEnv])
		if err != nil {
			t.Fatal(err)
		}
		var migrations int
		err = pool.QueryRow(ctx, "SELECT count(*) FROM goose_db_version WHERE is_applied AND version_id > 0").Scan(&migrations)
		pool.Close()
		if err != nil || migrations == 0 {
			t.Fatalf("%s missing schema: %d %v", module.Name, migrations, err)
		}
	}
	state, err := m.Status()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range FullStackServices() {
		if state.Processes[name].PID != 0 {
			t.Fatal("consumer started during schema preparation")
		}
	}
	helpers := 0
	for name, code := range state.ExitCodes {
		if strings.Contains(name, "full-stack-schema-") {
			helpers++
			if code != 0 {
				t.Fatalf("schema failed %s %d", name, code)
			}
		}
	}
	if helpers != 8 {
		t.Fatalf("schema tools not supervised: %v", state.ExitCodes)
	}
	data, err := os.ReadFile(o.Key.StatePath())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), dsn) {
		t.Fatal("state leaked DSN")
	}
	t.Logf("full stack all 8 module schemas ready; namespace=%s; zero consumers", o.Key.Namespace)
}

func TestFullStackRunnerProcess(t *testing.T) {
	if os.Getenv("ATHENA_TASK10_FULLSTACK_HELPER") != "1" {
		return
	}
	var o RunOptions
	if err := json.Unmarshal([]byte(os.Getenv("ATHENA_TASK10_OPTIONS")), &o); err != nil {
		os.Exit(92)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer cancel()
	if err := RunFullStack(ctx, o); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(93)
	}
	os.Exit(0)
}
func startFullStackRunner(t *testing.T, o RunOptions, toolPath string) <-chan error {
	t.Helper()
	data, _ := json.Marshal(o)
	cmd := exec.Command(os.Args[0], "-test.run=^TestFullStackRunnerProcess$")
	env := EnvironmentFor(environmentMap(os.Environ()), []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN"})
	if toolPath != "" {
		env = EnvironmentFor(environmentMap(append(env, "PATH="+toolPath+":"+os.Getenv("PATH"))), []string{"PATH", "HOME", "TMPDIR", "GOCACHE", "GOMODCACHE", "GOPATH", "GOROOT", "GOTOOLCHAIN"})
	}
	cmd.Env = append(env, RunIDEnv+"="+NewRunID(), "ATHENA_TASK10_FULLSTACK_HELPER=1", "ATHENA_TASK10_OPTIONS="+string(data))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := ensureDir(o.Key); err != nil {
		t.Fatal(err)
	}
	logfile, err := os.CreateTemp(o.Key.Dir(), "full-stack-supervisor-")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = logfile
	cmd.Stderr = logfile
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait(); logfile.Close() }()
	t.Logf("full stack instance=%s namespace=%s supervisor=%d log=%s", o.Key.Name, o.Key.Namespace, cmd.Process.Pid, logfile.Name())
	return done
}
func fullStackFixtureOptions(t *testing.T) RunOptions {
	t.Helper()
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
	// Stop owned services before closing the loopback Telegram fixture.
	t.Cleanup(telegram.Close)
	extra := "ATHENA_NOTIFICATION_TELEGRAM_API_URL=" + telegram.URL + "\nATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN=123:task10-fixture\nATHENA_NOTIFICATION_TEST_TELEGRAM_CHAT_ID=-1001\nATHENA_NOTIFICATION_PROD_TELEGRAM_CHAT_ID=-1002\n"
	extra += "ATHENA_UI_PORT=" + fixturePort(t) + "\nATHENA_WALLET_PORT=" + fixturePort(t) + "\nATHENA_PROFIT_SHARING_PORT=" + fixturePort(t) + "\nATHENA_API_CONTENT_TYPES=\nATHENA_UI_BANNER_CONTENT=literal banner $value\n"
	return runnerOptions(t, "managed", nil, extra)
}
func TestFullStackRealFatalKeepsAPIAndEveryOtherConsumer(t *testing.T) {
	o := fullStackFixtureOptions(t)
	done := startFullStackRunner(t, o, "")
	awaitRunning(t, o, done)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	state, err := Status(ctx, o.Key)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range FullStackServices() {
		if state.Health[name] != "ready" {
			t.Fatalf("%s not ready: %v", name, state.Health)
		}
	}
	for _, name := range []string{"api-server", "notification", "wallet", "profit-sharing", "trader-sync"} {
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", state.Processes[name].PID))
		if err != nil {
			t.Fatal(err)
		}
		env := environmentMap(strings.Split(string(data), "\x00"))
		if name == "api-server" {
			if v, ok := env["ATHENA_API_CONTENT_TYPES"]; !ok || v != "" {
				t.Fatal("API lost explicit empty")
			}
			if env["ATHENA_UI_BANNER_CONTENT"] != "literal banner $value" {
				t.Fatal("API settings changed")
			}
		}
		if name != "trader-sync" {
			if _, ok := env["ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY"]; ok {
				t.Fatal(name, "leaks cursor")
			}
		}
		if name != "wallet" {
			if _, ok := env["ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN"]; ok {
				t.Fatal(name, "leaks signer")
			}
		}
		if strings.Contains(state.Processes[name].Exe, "go-build") {
			t.Fatal("managed go run wrapper")
		}
	}
	if err = SignalProcess(state.Processes["trader-sync"], syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for {
		state, err = Status(ctx, o.Key)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := state.ExitCodes["trader-sync"]; ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("TS fatal not recorded")
		}
		time.Sleep(100 * time.Millisecond)
	}
	if state.Phase != "running" {
		t.Fatal("TS fatal stopped full stack")
	}
	for _, name := range FullStackServices() {
		if name != "trader-sync" && state.Health[name] != "ready" {
			t.Fatalf("TS fatal affected %s: %v", name, state.Health)
		}
	}
	if err = Stop(ctx, o.Key); err != nil {
		t.Fatal(err)
	}
	if err = <-done; err != nil {
		t.Fatal(err)
	}
	t.Log("TS fatal recorded; API/Notification/Wallet/Profit Sharing/UI remained ready; explicit stop reaped full stack")
}
func TestFullStackSchemaFailureStartsNoConsumers(t *testing.T) {
	o := fullStackFixtureOptions(t)
	toolDir := t.TempDir()
	realGo, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	// Only this test's go compiler is substituted. No old runtime script runs.
	script := "#!/bin/bash\nset -eu\nif [[ $1 == build && $2 == -o && $3 == */migration-*/* ]]; then\n printf '#!/bin/sh\\nexit 17\\n' > \"$3\"\n chmod 700 \"$3\"\n exit 0\nfi\nexec " + realGo + " \"$@\"\n"
	if err = os.WriteFile(filepath.Join(toolDir, "go"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	done := startFullStackRunner(t, o, toolDir)
	select {
	case err = <-done:
		if err == nil {
			t.Fatal("schema failure accepted")
		}
	case <-time.After(180 * time.Second):
		t.Fatal("schema failure timeout")
	}
	state, err := NewManager(o.Key).Status()
	if err != nil {
		t.Fatal(err)
	}
	if state.Phase != "stopped" {
		t.Fatal("failed batch was not stopped", state.Phase)
	}
	for _, name := range FullStackServices() {
		if state.Processes[name].PID != 0 {
			t.Fatal("consumer started despite schema failure", name)
		}
	}
	found := false
	for name, code := range state.ExitCodes {
		if strings.Contains(name, "schema-up") && code == 17 {
			found = true
		}
	}
	if !found {
		t.Fatalf("schema failure not recorded: %v", state.ExitCodes)
	}
	if len(state.Resources) == 0 || len(state.Logs) == 0 {
		t.Fatal("failure lost owned volume/log evidence")
	}
	t.Log("schema exited 17; zero full-stack consumers; own resources stopped and data/logs retained")
}
