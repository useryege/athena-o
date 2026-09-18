package devruntime

import (
	"context"
	"errors"
	"github.com/useryege/athena/internal/tradersync/rpcconfig"
	"github.com/useryege/athena/util/ethws"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const tokenKey = "ATHENA_TRADER_SYNC_INTERNAL_AUTH_TOKEN"

func (m *Manager) PrepareEnvironment(input map[string]string, specs []ServiceSpec, mode string) (map[string]string, error) {
	if mode != "managed" && mode != "external" {
		return nil, errors.New("unknown database mode")
	}
	env := map[string]string{}
	for k, v := range input {
		env[k] = v
	}
	env["ATHENA_LOCAL_RUNTIME_INSTANCE"] = m.Key.Name

	for _, spec := range specs {
		for _, key := range spec.EnvironmentKeys {
			if strings.HasSuffix(key, "_FILE") {
				if value := env[key]; value != "" && !filepath.IsAbs(value) {
					env[key] = filepath.Join(m.Key.Checkout, value)
				}
			}
		}
	}
	if err := prepareSelectedEndpoints(env, specs); err != nil {
		return nil, err
	}
	if selected(specs, "api-server") || selected(specs, "etherscan-manager") {
		if err := normalizeGatewayConfiguration(env); err != nil {
			return nil, err
		}
	}
	if needsSchema(specs, "solana-discovery") && mode == "external" {
		account, own := env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"], env["ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN"]
		if own != "" && own != account {
			return nil, errors.New("Solana and account schemas require the same explicit DSN")
		}
		env["ATHENA_SOLANA_DISCOVERY_POSTGRES_DSN"] = account
	}
	if mode == "external" {
		for _, owner := range selectedSchemas(specs) {
			if strings.TrimSpace(env[owner.DSNEnv]) == "" {
				return nil, errors.New("external database requires explicit " + owner.DSNEnv)
			}
		}
		if needsSchema(specs, "operation-log") {
			env["ATHENA_OPERATION_LOG_POSTGRES_DSN"] = env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"]
		}
	}
	if err := m.prepareCredentials(env, specs, mode); err != nil {
		return nil, err
	}
	if !selected(specs, "trader-sync") && !selected(specs, "api-server") {
		return env, nil
	}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	token, err := rpcconfig.ResolveSecret(lookup, tokenKey)
	if err != nil {
		return nil, err
	}
	cursor := env["ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY"]
	if selected(specs, "trader-sync") {
		cursor, err = rpcconfig.ResolveSecret(lookup, "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY")
	}
	if err != nil {
		return nil, err
	}
	if cursor != "" && token == cursor {
		return nil, errors.New("internal token must differ from cursor key")
	}
	if _, ok := env["ATHENA_TRADER_SYNC_GRPC_TRANSPORT"]; !ok {
		env["ATHENA_TRADER_SYNC_GRPC_TRANSPORT"] = "loopback-insecure"
	}
	if _, ok := env["ATHENA_TRADER_SYNC_LISTEN_ADDRESS"]; !ok {
		env["ATHENA_TRADER_SYNC_LISTEN_ADDRESS"] = "127.0.0.1:8122"
	}
	if _, ok := env["ATHENA_TRADER_SYNC_SERVER_ADDRESS"]; !ok {
		env["ATHENA_TRADER_SYNC_SERVER_ADDRESS"] = env["ATHENA_TRADER_SYNC_LISTEN_ADDRESS"]
	}
	if selected(specs, "trader-sync") {
		for _, field := range []struct {
			key     string
			schemes []string
		}{{"ATHENA_TRADER_SYNC_HTTP_URL", []string{"http", "https"}}, {"ATHENA_TRADER_SYNC_WSS_URL", []string{"ws", "wss"}}, {"ATHENA_URL", []string{"http", "https"}}} {
			u, e := url.Parse(env[field.key])
			valid := false
			if e == nil {
				for _, scheme := range field.schemes {
					valid = valid || u.Scheme == scheme
				}
			}
			if !valid || u.Host == "" || u.User != nil || u.Fragment != "" {
				return nil, errors.New("invalid " + field.key)
			}
		}
		if cursor == "" {
			return nil, errors.New("Trader Sync requires stable cursor key")
		}
		if v, ok := env["ATHENA_TRADER_SYNC_MAX_IN_FLIGHT_SOURCES"]; ok {
			n, e := strconv.Atoi(v)
			if e != nil || n < 1 {
				return nil, errors.New("invalid Trader Sync max in flight sources")
			}
		}
		if v, ok := env["ATHENA_TRADER_SYNC_SHUTDOWN_TIMEOUT"]; ok {
			d, e := time.ParseDuration(v)
			if e != nil || d <= 0 {
				return nil, errors.New("invalid Trader Sync shutdown timeout")
			}
		}
		if _, err = rpcconfig.LoadServer(lookup); err != nil {
			return nil, err
		}
		release, _ := os.ReadFile("/proc/sys/kernel/osrelease")
		wsl := env["WSL_DISTRO_NAME"] != "" || env["WSL_INTEROP"] != "" || strings.Contains(strings.ToLower(string(release)), "microsoft")
		err = applyLocalProxy(env, wsl, func() (string, error) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			b, e := command(ctx, "ip", "route", "show", "default")
			if e != nil {
				return "", e
			}
			words := strings.Fields(string(b))
			for i, w := range words {
				if w == "via" && i+1 < len(words) {
					return words[i+1], nil
				}
			}
			return "", errors.New("WSL default gateway missing")
		})
		if err != nil {
			return nil, err
		}
		if err = ethws.ValidateProxyURL(env["ATHENA_TRADER_SYNC_PROXY_URL"]); err != nil {
			return nil, errors.New("invalid Trader Sync proxy URL")
		}
	}
	if selected(specs, "api-server") {
		if _, err = rpcconfig.LoadClient(lookup); err != nil {
			return nil, err
		}
	}
	return env, nil
}
func applyLocalProxy(env map[string]string, wsl bool, gateway func() (string, error)) error {
	if _, ok := env["ATHENA_TRADER_SYNC_PROXY_URL"]; ok || !wsl {
		return nil
	}
	host, err := gateway()
	if err != nil {
		return err
	}
	if net.ParseIP(host) == nil {
		return errors.New("invalid WSL default gateway")
	}
	env["ATHENA_TRADER_SYNC_PROXY_URL"] = "http://" + net.JoinHostPort(host, "10809")
	return nil
}
