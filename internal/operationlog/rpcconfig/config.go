package rpcconfig

import (
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
)

type Server struct {
	ListenAddress, Transport, Token, CursorKey, CertFile, KeyFile string
	MaxMessageBytes                                               int
}
type Client struct {
	Address, Transport, Token, CAFile, ServerName string
	MaxMessageBytes                               int
}

const DefaultMaxMessageBytes = 64 << 10

func ResolveSecret(lookup func(string) (string, bool), name string) (string, error) {
	v, ok := lookup(name)
	p, pok := lookup(name + "_FILE")
	if ok && pok {
		return "", fmt.Errorf("%s and %s_FILE are mutually exclusive", name, name)
	}
	if !pok {
		return v, nil
	}
	if p == "" {
		return "", fmt.Errorf("%s_FILE requires a path", name)
	}
	b, e := os.ReadFile(p)
	if e != nil {
		return "", fmt.Errorf("cannot read %s_FILE", name)
	}
	return strings.TrimSuffix(string(b), "\n"), nil
}
func LoadServer(lookup func(string) (string, bool)) (Server, error) {
	c := Server{ListenAddress: value(lookup, "ATHENA_OPERATION_LOG_LISTEN_ADDRESS", "127.0.0.1:8124"), Transport: value(lookup, "ATHENA_OPERATION_LOG_GRPC_TRANSPORT", "plaintext")}
	var e error
	c.Token, e = ResolveSecret(lookup, "ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN")
	if e != nil {
		return c, e
	}
	c.CursorKey, e = ResolveSecret(lookup, "ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY")
	if e != nil {
		return c, e
	}
	c.CertFile, _ = lookup("ATHENA_OPERATION_LOG_TLS_CERT_FILE")
	c.KeyFile, _ = lookup("ATHENA_OPERATION_LOG_TLS_KEY_FILE")
	c.MaxMessageBytes = DefaultMaxMessageBytes
	if e = c.Validate(); e != nil {
		return c, e
	}
	return c, nil
}
func value(l func(string) (string, bool), k, d string) string {
	if v, ok := l(k); ok && v != "" {
		return v
	}
	return d
}
func (c Server) Validate() error {
	if err := address(c.ListenAddress, c.Transport); err != nil {
		return err
	}
	if len(c.Token) < 32 || strings.ContainsAny(c.Token, " \t\r\n") {
		return fmt.Errorf("operation-log internal token requires at least 32 bytes")
	}
	if len(c.CursorKey) != 64 {
		return fmt.Errorf("operation-log cursor HMAC key requires 64 hex characters")
	}
	if _, e := hex.DecodeString(c.CursorKey); e != nil {
		return fmt.Errorf("operation-log cursor HMAC key must be hexadecimal")
	}
	if c.Transport == "tls" && (c.CertFile == "" || c.KeyFile == "") {
		return fmt.Errorf("operation-log TLS certificate and key files are required")
	}
	return nil
}
func address(a, t string) error {
	if t != "plaintext" && t != "tls" {
		return fmt.Errorf("operation-log transport must be plaintext or tls")
	}
	h, p, e := net.SplitHostPort(a)
	if e != nil {
		return fmt.Errorf("operation-log listen address must be host:port")
	}
	n, e := strconv.Atoi(p)
	if e != nil || n < 1 || n > 65535 {
		return fmt.Errorf("operation-log port invalid")
	}
	if t == "plaintext" {
		ip := net.ParseIP(h)
		if h != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return fmt.Errorf("operation-log plaintext requires loopback address")
		}
	}
	return nil
}

func LoadClient(lookup func(string) (string, bool)) (Client, error) {
	c := Client{Address: value(lookup, "ATHENA_OPERATION_LOG_SERVER_ADDRESS", "127.0.0.1:8124"), Transport: value(lookup, "ATHENA_OPERATION_LOG_GRPC_TRANSPORT", "plaintext"), ServerName: value(lookup, "ATHENA_OPERATION_LOG_TLS_SERVER_NAME", "")}
	var err error
	c.Token, err = ResolveSecret(lookup, "ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN")
	if err != nil {
		return c, err
	}
	c.CAFile, _ = lookup("ATHENA_OPERATION_LOG_TLS_CA_FILE")
	c.MaxMessageBytes = DefaultMaxMessageBytes
	if err := c.Validate(); err != nil {
		return c, err
	}
	return c, nil
}
func (c Client) Validate() error {
	if err := address(c.Address, c.Transport); err != nil {
		return err
	}
	if len(c.Token) < 32 || strings.ContainsAny(c.Token, " \t\r\n") {
		return fmt.Errorf("operation-log internal token requires at least 32 bytes")
	}
	if c.Transport == "tls" && strings.TrimSpace(c.ServerName) == "" {
		return fmt.Errorf("operation-log TLS server name is required")
	}
	return nil
}
