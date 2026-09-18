package rpcconfig

import (
	"strings"
	"testing"
)

func TestLoadServerValidatesTokenAndCursorKey(t *testing.T) {
	m := map[string]string{"ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN": "01234567890123456789012345678901", "ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
	c, e := LoadServer(func(k string) (string, bool) { v, ok := m[k]; return v, ok })
	if e != nil {
		t.Fatal(e)
	}
	if c.MaxMessageBytes != 64<<10 {
		t.Fatal(c.MaxMessageBytes)
	}
}
func TestLoadServerRejectsAmbiguousSecret(t *testing.T) {
	m := map[string]string{"ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN": "x", "ATHENA_OPERATION_LOG_INTERNAL_AUTH_TOKEN_FILE": "x", "ATHENA_OPERATION_LOG_CURSOR_HMAC_KEY": "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}
	if _, e := LoadServer(func(k string) (string, bool) { v, ok := m[k]; return v, ok }); e == nil {
		t.Fatal("accepted ambiguous secret")
	}
}

func TestPlaintextClientRequiresLoopback(t *testing.T) {
	c := Client{Address: "10.0.0.2:8124", Transport: "plaintext", Token: strings.Repeat("x", 32)}
	if err := c.Validate(); err == nil {
		t.Fatal("accepted plaintext operation-log client outside loopback")
	}
}

func TestTLSClientAllowsNonLoopbackWithExplicitServerName(t *testing.T) {
	c := Client{Address: "operation-log:8124", Transport: "tls", ServerName: "operation-log", Token: strings.Repeat("x", 32)}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestTLSRequiresExplicitCertificateAndServerName(t *testing.T) {
	server := Server{ListenAddress: "127.0.0.1:8124", Transport: "tls", Token: strings.Repeat("x", 32), CursorKey: strings.Repeat("a", 64)}
	if err := server.Validate(); err == nil {
		t.Fatal("accepted TLS server without certificate/key")
	}
	client := Client{Address: "operation-log:8124", Transport: "tls", Token: strings.Repeat("x", 32)}
	if err := client.Validate(); err == nil {
		t.Fatal("accepted TLS client without server name")
	}
}
