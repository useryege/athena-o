package rpcconfig

import "testing"

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
