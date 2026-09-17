package devruntime

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestFullStackExplicitServicesAndEnvironment(t *testing.T) {
	modules := map[string]bool{}
	for _, module := range selectedSchemas(fullStackSpecs()) {
		modules[module.Name] = true
	}
	if modules["sports-live"] || modules["sports-history"] {
		t.Fatal("retired Sports databases would be prepared")
	}
	if modules["worm-markets"] || !modules["worm-trading"] {
		t.Fatal("retired Markets storage or retained Trading storage changed")
	}
	want := []string{"wallet", "notification", "etherscan-manager", "api-server", "ui", "trader-sync", "solana-discovery", "market-radar", "managed-oo", "profit-sharing", "worm-trading"}
	if got := FullStackServices(); !reflect.DeepEqual(got, want) {
		t.Fatalf("full stack: %v", got)
	}
	specs := fullStackSpecs()
	if len(specs) != len(want) {
		t.Fatalf("unexpected graph: %v", specs)
	}
	for _, s := range specs {
		if s.Name != "ui" && (s.BuildPackage == "" || s.Binary == "" || slices.Contains(s.Args, "go")) {
			t.Fatalf("not built service: %+v", s)
		}
		for _, key := range []string{"ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY", "ATHENA_TRADER_SYNC_HTTP_URL", "ATHENA_TRADER_SYNC_WSS_URL"} {
			if s.Name != "trader-sync" && slices.Contains(s.EnvironmentKeys, key) {
				t.Fatalf("%s leaks %s", s.Name, key)
			}
		}
		if s.Name != "wallet" && s.Name != "worm-trading" && slices.Contains(s.EnvironmentKeys, "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN") {
			t.Fatalf("%s leaks signer", s.Name)
		}
	}
	if _, err := ResolveServices([]string{"wallet"}); err != nil {
		t.Fatal("wallet missing independent entry")
	}
}

func TestFullStackStopResetCannotTouchLocalInstanceOrOtherCheckout(t *testing.T) {
	for _, otherCheckout := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("other-checkout=%v/reverse=%v", otherCheckout, reverse), func(t *testing.T) {
				root := t.TempDir()
				full, _ := NewInstanceKey(root, "full-stack")
				otherName := "trader-sync"
				if otherCheckout {
					root = t.TempDir()
					otherName = "full-stack"
				}
				other, _ := NewInstanceKey(root, otherName)
				if reverse {
					full, other = other, full
				}
				a, b := initialState(t, full), initialState(t, other)
				ca, _ := temporaryProcess(t, "exec sleep 60")
				cb, _ := temporaryProcess(t, "exec sleep 60")
				time.Sleep(20 * time.Millisecond)
				pa, err := ReadProcess(ca.Process.Pid)
				if err != nil {
					t.Fatal(err)
				}
				pb, err := ReadProcess(cb.Process.Pid)
				if err != nil {
					t.Fatal(err)
				}
				a.Services = FullStackServices()
				b.Services = []string{"trader-sync"}
				if reverse {
					a.Services, b.Services = b.Services, a.Services
				}
				a.Processes["trader-sync"] = pa
				b.Processes["trader-sync"] = pb
				for _, s := range []*State{&a, &b} {
					s.Resources = []ResourceRef{{Kind: "container", ID: s.Key.Namespace + "-cid", Name: s.Key.Namespace + "-pg", Namespace: s.Key.Namespace, RunID: s.RunID, Owned: true}, {Kind: "volume", ID: s.Key.Namespace + "-data", Name: s.Key.Namespace + "-data", Namespace: s.Key.Namespace, RunID: s.RunID, Owned: true}}
				}
				if err := SaveState(full.StatePath(), a); err != nil {
					t.Fatal(err)
				}
				if err := SaveState(other.StatePath(), b); err != nil {
					t.Fatal(err)
				}
				before, err := os.ReadFile(other.StatePath())
				if err != nil {
					t.Fatal(err)
				}
				m := NewManager(full)
				var signalled []int
				var removedContainers, removedVolumes []string
				m.Signal = func(p ProcessIdentity, s syscall.Signal) error {
					signalled = append(signalled, p.PGID)
					return SignalProcess(p, s)
				}
				m.Docker.Exec = func(_ context.Context, name string, args ...string) ([]byte, error) {
					if name != "docker" {
						t.Fatal(name)
					}
					id := args[len(args)-1]
					for _, r := range a.Resources {
						if r.ID != id {
							continue
						}
						switch args[1] {
						case "inspect":
							if r.Kind == "volume" {
								return json.Marshal(map[string]any{"Name": r.Name, "Labels": ResourceLabels(full, r.Kind, r.RunID, r.Name)})
							}
							return json.Marshal(map[string]any{"Id": r.ID, "Name": "/" + r.Name, "Config": map[string]any{"Labels": ResourceLabels(full, r.Kind, r.RunID, r.Name)}})
						case "stop":
							return nil, nil
						case "rm":
							if r.Kind == "container" {
								removedContainers = append(removedContainers, id)
							} else {
								removedVolumes = append(removedVolumes, id)
							}
							return nil, nil
						}
					}
					return nil, fmt.Errorf("unexpected operation %v", args)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := m.Stop(ctx); err != nil {
					t.Fatal(err)
				}
				if err := m.Reset(ctx); err != nil {
					t.Fatal(err)
				}
				if len(signalled) == 0 || len(removedContainers) != 1 || len(removedVolumes) != 1 {
					t.Fatalf("missing own cleanup: %v %v %v", signalled, removedContainers, removedVolumes)
				}
				if slices.Contains(signalled, pb.PGID) || slices.Contains(removedContainers, b.Resources[0].ID) || slices.Contains(removedVolumes, b.Resources[1].Name) {
					t.Fatal("touched another instance")
				}
				after, err := os.ReadFile(other.StatePath())
				if err != nil || string(before) != string(after) {
					t.Fatal("other state changed", err)
				}
				if _, err := VerifyProcess(pb); err != nil {
					t.Fatal("other process stopped", err)
				}
			})
		}
	}
}

func TestFullStackDoesNotPrepareWormMarkets(t *testing.T) {
	for _, module := range selectedSchemas(fullStackSpecs()) {
		if module.Name == "worm-markets" || module.Database == "worm_markets" {
			t.Fatalf("retired module would be recreated: %+v", module)
		}
	}
}

func TestFullStackDefaultsPreserveAPIUIAndConfineWallet(t *testing.T) {
	env := map[string]string{"ATHENA_URL": "http://localhost:4000", "ATHENA_SERVER_PORT": "38110", "ATHENA_WALLET_PORT": "38111", "ATHENA_PROFIT_SHARING_LISTEN_PORT": "38112", "ATHENA_WALLET_ENCRYPTION_KEY": "explicit-wallet-key"}
	prepareFullStackEnvironment(env)
	if err := prepareSelectedEndpoints(env, fullStackSpecs()); err != nil {
		t.Fatal(err)
	}
	if env["ATHENA_WALLET_SERVER_ADDRESS"] != "127.0.0.1:38111" || env["ATHENA_PROFIT_SHARING_SERVER_ADDRESS"] != "127.0.0.1:38112" {
		t.Fatal("selected clients do not follow owned services", env)
	}
	if env["ATHENA_GOOGLE_OIDC_REDIRECT_URI"] != "http://localhost:4000/auth/google/callback" || env["ATHENA_ACCOUNT_AVATAR_MAX_BYTES"] != "2097152" || env["ATHENA_WALLET_ENCRYPTION_KEY"] != "explicit-wallet-key" {
		t.Fatal("lost local defaults or overwrote explicit data", env)
	}
	for _, s := range fullStackSpecs() {
		if s.Name == "api-server" && !slices.Contains(s.EnvironmentKeys, "ATHENA_ACCOUNT_AVATAR_MAX_BYTES") {
			t.Fatal("avatar limit is not passed to API")
		}
	}
}

func TestFullStackServiceAddresses(t *testing.T) {
	env := map[string]string{"ATHENA_WALLET_PORT": "39201", "ATHENA_PROFIT_SHARING_LISTEN_PORT": "39202"}
	if got := serviceAddress("wallet", env); got != "127.0.0.1:39201" {
		t.Fatal(got)
	}
	if got := serviceAddress("profit-sharing", env); got != "127.0.0.1:39202" {
		t.Fatal(got)
	}
}

func TestFullStackPreservesConsumerConfigurationLiterally(t *testing.T) {
	shared := map[string]string{"ATHENA_LOGFORMAT": "text", "ATHENA_LOG_FORMAT_ENABLE_FULL_TIMESTAMP": "1", "ATHENA_LOG_FORMAT_TIMESTAMP": "2006-01-02", "FORCE_LOG_COLORS": "1", "HTTP_PROXY": "http://fixture-proxy:8080", "HTTPS_PROXY": "http://fixture-proxy:8080", "NO_PROXY": "postgres,redis", "ALL_PROXY": "socks5://fixture-proxy:1080", "http_proxy": "http://fixture-proxy:8080", "https_proxy": "http://fixture-proxy:8080", "no_proxy": "postgres,redis", "all_proxy": "socks5://fixture-proxy:1080"}
	api_only := map[string]string{"ETHERSCAN_GATEWAY_IPS": "192.0.2.1,192.0.2.2", "ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN": "fixture-gateway-token", "ATHENA_ETHERSCAN_MANAGER_API_KEYS": "fixture-key-one,fixture-key-two", "ATHENA_ETHERSCAN_GATEWAY_PROBE_QUERY_ADDRESS": "0x0000000000000000000000000000000000000001", "ATHENA_API_CONTENT_TYPES": "application/json;application/cbor", "ATHENA_SERVER_OTLP_ADDRESS": "collector:4317", "ATHENA_SERVER_OTLP_INSECURE": "false", "ATHENA_SERVER_OTLP_HEADERS": "authorization=fixture-trace-token", "ATHENA_SERVER_OTLP_ATTRS": "deployment:acceptance,region:local", "ATHENA_SERVER_X_FRAME_OPTIONS": "deny", "ATHENA_SERVER_CONTENT_SECURITY_POLICY": "frame-ancestors https://portal.example;", "ATHENA_SERVER_CONNECTION_STATUS_CACHE_EXPIRATION": "20m", "ATHENA_DEFAULT_CACHE_EXPIRATION": "12h", "REDISDB": "4", "REDIS_USERNAME": "fixture-cache-user", "REDIS_COMPRESSION": "none", "REDIS_RETRY_COUNT": "2", "REDIS_CREDS_DIR_PATH": "/fixture/cache-credentials", "REDIS_SENTINEL_USERNAME": "fixture-sentinel-user", "REDIS_SENTINEL_PASSWORD": "fixture-sentinel-password", "ATHENA_MAX_COOKIE_NUMBER": "12", "ATHENA_ADDITIONAL_URLS": "https://portal.example,https://member.example", "ATHENA_SESSION_DURATION": "8h", "ATHENA_HELP_CHAT_URL": "https://support.example", "ATHENA_HELP_CHAT_TEXT": "Support", "ATHENA_STATUS_BADGE_ENABLED": "true", "ATHENA_STATUS_BADGE_ROOT_URL": "https://badges.example", "ATHENA_ANONYMOUS_USER_ENABLED": "true", "ATHENA_UI_CSS_URL": "https://static.example/site.css", "ATHENA_UI_BANNER_CONTENT": "Maintenance notice", "ATHENA_UI_BANNER_PERMANENT": "true", "ATHENA_UI_BANNER_POSITION": "top", "ATHENA_UI_BANNER_URL": "https://status.example", "ATHENA_HELP_DOWNLOAD_DARWIN_AMD64": "https://download.example/DARWIN_AMD64", "ATHENA_HELP_DOWNLOAD_DARWIN_ARM64": "https://download.example/DARWIN_ARM64", "ATHENA_HELP_DOWNLOAD_WINDOWS_AMD64": "https://download.example/WINDOWS_AMD64", "ATHENA_HELP_DOWNLOAD_LINUX_AMD64": "https://download.example/LINUX_AMD64", "ATHENA_HELP_DOWNLOAD_LINUX_ARM64": "https://download.example/LINUX_ARM64", "ATHENA_HELP_DOWNLOAD_LINUX_PPC64LE": "https://download.example/LINUX_PPC64LE", "ATHENA_HELP_DOWNLOAD_LINUX_S390X": "https://download.example/LINUX_S390X"}

	for _, empty := range []bool{false, true} {
		input := map[string]string{"ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY": "private-cursor", "ATHENA_NOTIFICATION_TELEGRAM_BOT_TOKEN": "private-telegram", "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN": "private-signer"}
		for k, v := range shared {
			input[k] = v
		}
		for k, v := range api_only {
			input[k] = v
		}
		if empty {
			for k := range input {
				input[k] = ""
			}
		}
		for _, spec := range fullStackSpecs() {
			got := environmentMap(EnvironmentFor(input, spec.EnvironmentKeys))
			for k, v := range input {
				_, common := shared[k]
				_, api := api_only[k]
				want := api && spec.Name == "api-server" || common && (spec.Name == "api-server" || spec.Name == "notification")
				if want {
					actual, exists := got[k]
					if !exists || actual != v {
						t.Errorf("%s lost literal %s (empty=%v)", spec.Name, k, empty)
					}
				}
				if api && spec.Name != "api-server" && !(spec.Name == "etherscan-manager" && (k == "ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN" || k == "ATHENA_ETHERSCAN_MANAGER_API_KEYS")) {
					if _, exists := got[k]; exists {
						t.Errorf("%s leaks %s", spec.Name, k)
					}
				}
			}
		}
	}
}

func TestFullStackRejectsExternalBeforeCreatingState(t *testing.T) {
	key := testKey(t)
	err := RunFullStack(context.Background(), RunOptions{Key: key, DBMode: "external"})
	if err == nil {
		t.Fatal("external full stack accepted")
	}
	if _, err := os.Stat(key.Dir()); !os.IsNotExist(err) {
		t.Fatal("external rejection created state/resources", err)
	}
}

func TestFullStackPortConflictDoesNotClaimExistingListener(t *testing.T) {
	key := testKey(t)
	var content strings.Builder
	for _, field := range []string{"ATHENA_SERVER_PORT", "ATHENA_NOTIFICATION_PORT", "ATHENA_PROFIT_SHARING_LISTEN_PORT", "ATHENA_WALLET_PORT", "ATHENA_TRADER_SYNC_LISTEN_ADDRESS"} {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		_, port, _ := net.SplitHostPort(listener.Addr().String())
		value := port
		if field == "ATHENA_TRADER_SYNC_LISTEN_ADDRESS" {
			value = listener.Addr().String()
		}
		fmt.Fprintf(&content, "%s=%s\n", field, value)
		if field == "ATHENA_WALLET_PORT" {
			defer listener.Close()
		} else {
			listener.Close()
		}
	}
	content.WriteString("ATHENA_URL=http://localhost:4000\nATHENA_TRADER_SYNC_HTTP_URL=http://127.0.0.1:1\nATHENA_TRADER_SYNC_WSS_URL=ws://127.0.0.1:1\nATHENA_TRADER_SYNC_PROXY_URL=\nATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN=task10-internal-token-with-at-least-32-bytes\nATHENA_TRADER_SYNC_CURSOR_HMAC_KEY=task10-cursor\n")
	path := filepath.Join(key.Checkout, "fixture.env")
	if err := os.WriteFile(path, []byte(content.String()), 0600); err != nil {
		t.Fatal(err)
	}
	err := RunFullStack(context.Background(), RunOptions{Key: key, EnvFile: path})
	if err == nil || !strings.Contains(err.Error(), "wallet port is unavailable") {
		t.Fatal("not an ownership conflict", err)
	}
	if _, err := os.Stat(key.StatePath()); !os.IsNotExist(err) {
		t.Fatal("conflict started lifecycle", err)
	}
}
