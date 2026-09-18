package devruntime

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/useryege/athena/internal/tradersync/rpcconfig"
)

func prepareSelectedEndpoints(env map[string]string, specs []ServiceSpec) error {
	for _, spec := range specs {
		if _, ok := env[spec.ListenKey]; !ok {
			env[spec.ListenKey] = "127.0.0.1"
			if spec.Name == "trader-sync" || spec.Name == "operation-log" {
				env[spec.ListenKey] += ":" + spec.DefaultPort
			}
		}
		for _, key := range spec.AddressKeys {
			env[key] = spec.Address(env)
		}
	}
	if selected(specs, "ui") && selected(specs, "api-server") {
		env["ATHENA_API_URL"] = "http://" + serviceAddress("api-server", env)
	}
	if selected(specs, "ui") {
		base := envDefault(env, "ATHENA_SERVER_BASEHREF", "/")
		if !strings.HasPrefix(base, "/") || strings.ContainsAny(base, "?#\\") || strings.Contains(base, "..") {
			return errors.New("invalid UI BaseHRef")
		}
		base = "/" + strings.Trim(base, "/")
		if base != "/" {
			base += "/"
		}
		env["ATHENA_SERVER_BASEHREF"] = base
		origin := envDefault(env, "ATHENA_URL", "http://localhost:"+envDefault(env, "ATHENA_UI_PORT", "4000")+strings.TrimSuffix(base, "/"))
		u, err := url.Parse(origin)
		if err != nil || u.Scheme != "http" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") || u.Port() != envDefault(env, "ATHENA_UI_PORT", "4000") || strings.TrimSuffix(u.Path, "/") != strings.TrimSuffix(base, "/") || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
			return errors.New("UI origin must match local port and BaseHRef")
		}
		env["ATHENA_URL"] = origin
		callback := strings.TrimSuffix(origin, "/") + "/auth/google/callback"
		if value, ok := env["ATHENA_GOOGLE_OIDC_REDIRECT_URI"]; ok && value != callback {
			return errors.New("Google callback must match UI origin and BaseHRef")
		}
		env["ATHENA_GOOGLE_OIDC_REDIRECT_URI"] = callback
	}
	return nil
}

func (m *Manager) prepareCredentials(env map[string]string, specs []ServiceSpec, mode string) error {
	candidates := []string{tokenKey, "ATHENA_TRADER_SYNC_CURSOR_HMAC_KEY", "ATHENA_NOTIFICATION_INTERNAL_AUTH_TOKEN", "ATHENA_WALLET_INTERNAL_AUTH_TOKEN", "ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN", "ATHENA_WALLET_ENCRYPTION_KEY", "ATHENA_WORM_TRADING_INTERNAL_AUTH_TOKEN", "ATHENA_WORM_TRADING_CREDENTIAL_ENCRYPTION_KEY", "ATHENA_SOLANA_DISCOVERY_INTERNAL_AUTH_TOKEN", "ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN", "ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY"}
	for _, key := range candidates {
		used := false
		for _, spec := range specs {
			used = used || slices.Contains(spec.EnvironmentKeys, key)
		}
		if !used {
			continue
		}
		_, direct := env[key]
		_, file := env[key+"_FILE"]
		if mode == "external" && key == tokenKey && !direct && !file {
			return errors.New("external API/Trader Sync requires explicit internal token")
		}
		value := ""
		if direct || file {
			var err error
			value, err = rpcconfig.ResolveSecret(func(k string) (string, bool) { v, ok := env[k]; return v, ok }, key)
			if err != nil {
				return err
			}
			if value == "" {
				return fmt.Errorf("empty %s", key)
			}
		}
		path := filepath.Join(m.Key.Dir(), "credential-"+strings.ToLower(key))
		if key == tokenKey {
			path = filepath.Join(m.Key.Dir(), "internal-token")
		}
		err := withLock(m.Key, func() error {
			info, err := os.Lstat(path)
			if err == nil {
				if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
					return fmt.Errorf("unsafe credential file for %s", key)
				}
				saved, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if len(saved) == 0 {
					return fmt.Errorf("empty persistent credential %s", key)
				}
				if value != "" && value != string(saved) {
					return fmt.Errorf("configured %s conflicts with persistent instance credential", key)
				}
				value = string(saved)
				return nil
			}
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			// The recorded environment is an existing instance identity too.
			// Reuse it before creating a dedicated credential file.
			previousPath := filepath.Join(m.Key.Dir(), "environment.json")
			if info, e := os.Lstat(previousPath); e == nil {
				if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
					return errors.New("unsafe recorded environment")
				}
				data, e := os.ReadFile(previousPath)
				if e != nil {
					return e
				}
				var previous map[string]string
				if e = json.Unmarshal(data, &previous); e != nil {
					return errors.New("invalid recorded environment")
				}
				if saved := previous[key]; saved != "" {
					if value != "" && value != saved {
						return fmt.Errorf("configured %s conflicts with recorded instance credential", key)
					}
					value = saved
				}
			} else if !errors.Is(e, os.ErrNotExist) {
				return e
			}
			if value == "" {
				raw := make([]byte, 32)
				if _, err := rand.Read(raw); err != nil {
					return err
				}
				value = hex.EncodeToString(raw)
			}
			return atomicFile(path, []byte(value))
		})
		if err != nil {
			return err
		}
		// Resolve _FILE once and pass the stable value, avoiding later file changes.
		delete(env, key+"_FILE")
		env[key] = value
	}
	if env["ATHENA_WALLET_INTERNAL_AUTH_TOKEN"] != "" && env["ATHENA_WALLET_INTERNAL_AUTH_TOKEN"] == env["ATHENA_WALLET_WORM_EXECUTION_SIGNER_TOKEN"] {
		return errors.New("wallet signer and internal credentials must differ")
	}
	if env["ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN"] != "" && env["ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN"] == env["ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY"] {
		return errors.New("operation-log internal token must differ from cursor key")
	}
	return nil
}

func normalizeGatewayConfiguration(env map[string]string) error {
	parse := func(raw string, addresses bool) ([]string, error) {
		var ips []string
		for _, value := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ' ' || r == '\n' || r == '\t' || r == '\r' }) {
			host := value
			if addresses {
				var port string
				var err error
				host, port, err = net.SplitHostPort(value)
				if err != nil || port != "6776" {
					return nil, errors.New("Gateways require pure IP:6776")
				}
			}
			ip := net.ParseIP(host)
			if ip == nil {
				return nil, errors.New("Gateways require pure IP addresses")
			}
			ips = append(ips, ip.String())
		}
		slices.Sort(ips)
		return slices.Compact(ips), nil
	}
	a, err := parse(env["ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS"], true)
	if err != nil {
		return err
	}
	b, err := parse(env["ETHERSCAN_GATEWAY_IPS"], false)
	if err != nil {
		return err
	}
	if len(a) > 0 && len(b) > 0 && !slices.Equal(a, b) {
		return errors.New("Manager and API Gateway address sets conflict")
	}
	if len(a) == 0 {
		a = b
	}
	if len(a) == 0 {
		return nil
	}
	if len(a) != 5 {
		return errors.New("local runtime requires exactly five remote Gateways")
	}
	addresses := make([]string, len(a))
	for i, ip := range a {
		addresses[i] = net.JoinHostPort(ip, "6776")
	}
	env["ETHERSCAN_GATEWAY_IPS"] = strings.Join(a, ",")
	env["ATHENA_ETHERSCAN_MANAGER_GATEWAY_ADDRS"] = strings.Join(addresses, ",")
	return nil
}
