package devruntime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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
	if selected(specs, "worm-trading") {
		if _, ok := env["ATHENA_WORM_TRADING_LISTEN_ADDRESS"]; !ok {
			env["ATHENA_WORM_TRADING_LISTEN_ADDRESS"] = "127.0.0.1"
		}
		env["ATHENA_WORM_TRADING_SERVER_ADDRESS"] = serviceAddress("worm-trading", env)
		if mode == "external" && strings.TrimSpace(env["ATHENA_WORM_TRADING_POSTGRES_DSN"]) == "" {
			return nil, errors.New("external database requires explicit ATHENA_WORM_TRADING_POSTGRES_DSN")
		}
	}

	if selected(specs, "notification") {
		env["ATHENA_NOTIFICATION_SERVER_ADDRESS"] = serviceAddress("notification", env)
	}
	if selected(specs, "ui") && selected(specs, "api-server") {
		env["ATHENA_API_URL"] = "http://" + serviceAddress("api-server", env)
	}
	for _, spec := range specs {
		for _, key := range spec.EnvironmentKeys {
			if strings.HasSuffix(key, "_FILE") {
				if value, ok := env[key]; ok && value != "" && !filepath.IsAbs(value) {
					env[key] = filepath.Join(m.Key.Checkout, value)
				}
			}
		}
	}
	for _, key := range []string{"ATHENA_SERVER_LISTEN_ADDRESS", "ATHENA_NOTIFICATION_LISTEN_ADDRESS"} {
		if _, ok := env[key]; !ok {
			env[key] = "127.0.0.1"
		}
	}
	if selected(specs, "api-server") {
		for key, value := range map[string]string{"ATHENA_WALLET_INTERNAL_AUTH_TOKEN": "athena-local-wallet-internal-auth-token-2026", "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN": "athena-local-worm-trading-internal-auth-token-2026"} {
			if _, ok := env[key]; !ok {
				env[key] = value
			}
		}
	}
	if selected(specs, "api-server") || selected(specs, "notification") {
		if _, ok := env["ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN"]; !ok {
			env["ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN"] = "athena-local-notification-internal-auth-token-2026"
		}
	}
	if needsDatabase(specs) && mode == "external" && strings.TrimSpace(env["ATHENA_ACCOUNT_STATE_POSTGRES_DSN"]) == "" {
		return nil, errors.New("external database requires explicit ATHENA_ACCOUNT_STATE_POSTGRES_DSN")
	}
	if !selected(specs, "trader-sync") && !selected(specs, "api-server") {
		return env, nil
	}
	lookup := func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	_, direct := env[tokenKey]
	_, file := env[tokenKey+"_FILE"]
	if !direct && !file {
		if mode == "external" {
			return nil, errors.New("external API/Trader Sync requires explicit internal token")
		}
		var value []byte
		err := withLock(m.Key, func() error {
			path := filepath.Join(m.Key.Dir(), "internal-token")
			var err error
			value, err = os.ReadFile(path)
			if errors.Is(err, os.ErrNotExist) {
				raw := make([]byte, 32)
				if _, err = rand.Read(raw); err != nil {
					return err
				}
				value = []byte(hex.EncodeToString(raw))
				return atomicFile(path, value)
			}
			if err != nil {
				return err
			}
			info, err := os.Lstat(path)
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
				return errors.New("unsafe internal token file")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		env[tokenKey] = string(value)
	}
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
