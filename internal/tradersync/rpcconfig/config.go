// Package rpcconfig owns only the internal RPC boundary configuration.
package rpcconfig

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	ListenAddress, Transport, Token, CertFile, KeyFile string
	MaxMessageBytes                                    int
}
type Client struct {
	Address, Transport, Token, CAFile, ServerName string
	MaxMessageBytes                               int
}

const DefaultMaxMessageBytes = 200 * 1024 * 1024
const prefix = "ATHENA_TRADER_SYNC_"

func LoadServer(lookup func(string) (string, bool)) (Server, error) {
	c := Server{ListenAddress: valueOr(lookup, prefix+"LISTEN_ADDRESS", "127.0.0.1:8122"), Transport: valueOr(lookup, prefix+"GRPC_TRANSPORT", "tls")}
	var err error
	c.Token, err = ResolveSecret(lookup, prefix+"INTERNAL_AUTH_TOKEN")
	if err != nil {
		return c, err
	}
	c.MaxMessageBytes, err = messageSize(lookup)
	if err != nil {
		return c, err
	}
	c.CertFile, _ = lookup(prefix + "TLS_CERT_FILE")
	c.KeyFile, _ = lookup(prefix + "TLS_KEY_FILE")
	if err = c.Validate(); err != nil {
		return c, err
	}
	if c.Transport == "loopback-insecure" {
		host, port, _ := net.SplitHostPort(c.ListenAddress)
		if host == "localhost" {
			c.ListenAddress = net.JoinHostPort("127.0.0.1", port)
		}
	}
	if _, err = ServerCredentials(c); err != nil {
		return c, err
	}
	return c, nil
}
func LoadClient(lookup func(string) (string, bool)) (Client, error) {
	c := Client{Transport: valueOr(lookup, prefix+"GRPC_TRANSPORT", "tls")}
	c.Address, _ = lookup(prefix + "SERVER_ADDRESS")
	var err error
	c.Token, err = ResolveSecret(lookup, prefix+"INTERNAL_AUTH_TOKEN")
	if err != nil {
		return c, err
	}
	c.MaxMessageBytes, err = messageSize(lookup)
	if err != nil {
		return c, err
	}
	c.CAFile, _ = lookup(prefix + "TLS_CA_FILE")
	c.ServerName, _ = lookup(prefix + "TLS_SERVER_NAME")
	if err = c.Validate(); err != nil {
		return c, err
	}
	if _, err = ClientCredentials(c); err != nil {
		return c, err
	}
	return c, nil
}
func valueOr(lookup func(string) (string, bool), name, fallback string) string {
	if v, ok := lookup(name); ok {
		return v
	}
	return fallback
}
func messageSize(lookup func(string) (string, bool)) (int, error) {
	value, ok := lookup("ATHENA_GRPC_MAX_SIZE_MB")
	if !ok {
		return DefaultMaxMessageBytes, nil
	}
	n, err := strconv.ParseUint(value, 10, 64)
	max := uint64(^uint(0)>>1) / (1024 * 1024)
	if err != nil || n == 0 || n > max {
		return 0, fmt.Errorf("ATHENA_GRPC_MAX_SIZE_MB requires a positive integer within message size limits")
	}
	return int(n) * 1024 * 1024, nil
}
func (c Server) Validate() error {
	if err := validateBoundary(c.ListenAddress, c.Transport, false); err != nil {
		return err
	}
	if err := validateToken(c.Token); err != nil {
		return err
	}
	if c.MaxMessageBytes <= 0 {
		return fmt.Errorf("Trader Sync maximum message bytes must be positive")
	}
	return nil
}
func (c Client) Validate() error {
	if err := validateBoundary(c.Address, c.Transport, true); err != nil {
		return err
	}
	if err := validateToken(c.Token); err != nil {
		return err
	}
	if c.MaxMessageBytes <= 0 {
		return fmt.Errorf("Trader Sync maximum message bytes must be positive")
	}
	return nil
}
func validateBoundary(address, transport string, client bool) error {
	if transport != "tls" && transport != "loopback-insecure" {
		return fmt.Errorf("Trader Sync transport must be tls or loopback-insecure")
	}
	host, port, err := net.SplitHostPort(address)
	n, portErr := strconv.Atoi(port)
	if err != nil || portErr != nil || n < 1 || n > 65535 || strings.ContainsAny(host, "/\\?#@ \t\r\n") || strings.ContainsAny(port, "+- \t\r\n") {
		return fmt.Errorf("Trader Sync requires a valid host:port address")
	}
	ip := net.ParseIP(host)
	if client && (host == "" || (ip != nil && ip.IsUnspecified())) {
		return fmt.Errorf("Trader Sync client requires a connectable address")
	}
	if ip == nil && host != "" && !validHostname(host) {
		return fmt.Errorf("Trader Sync requires a valid hostname")
	}
	if transport == "loopback-insecure" {
		if host == "localhost" {
			return nil
		}
		if ip == nil || !ip.IsLoopback() {
			return fmt.Errorf("Trader Sync plaintext requires a loopback address")
		}
	}
	return nil
}
func validHostname(host string) bool {
	if len(host) > 253 {
		return false
	}
	for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}

// MethodBudget is an explicit allowlist; new RPCs need an intentional budget.
func MethodBudget(fullMethod string) (time.Duration, error) {
	const service = "/tradersync.internal.v1.TraderSyncService/"
	switch fullMethod {
	case service + "GetActivity", service + "GetSubscription", service + "ListSubscriptions", service + "ListActivities", service + "ListSubscriptionHistory", service + "GetSummaryBatch", service + "ListSummaryParts", service + "GetSubscriptionSummary", service + "ListSubscriptionSummaries", service + "GetTraderSyncRuntimeStatus":
		return 5 * time.Second, nil
	case service + "ResolveTarget", service + "CreateSubscription", service + "PauseSubscription", service + "ResumeSubscription", service + "CancelSubscription", service + "UpdateTargetNote":
		return 15 * time.Second, nil
	default:
		return 0, fmt.Errorf("unknown Trader Sync method")
	}
}
