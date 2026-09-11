package delivery

import (
	"bytes"
	"testing"
)

func TestFrozenPayloadPreservesTextAndBindsFormat(t *testing.T) {
	text := "  <A>&🙂 https://example.test/path?q=a&x=b\n"
	plain, err := EncodePayload(Payload{Format: "plain", Text: text})
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodePayload(plain)
	if err != nil || got.Text != text || got.Format != "plain" {
		t.Fatalf("frozen text changed: %#v %v", got, err)
	}
	html, err := EncodePayload(Payload{Format: "html", Text: text})
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(PayloadDigest(plain), PayloadDigest(html)) {
		t.Fatal("format not bound to digest")
	}
	for _, format := range []string{"", "markdown"} {
		if _, err := EncodePayload(Payload{Format: format, Text: text}); err == nil {
			t.Fatalf("accepted format %q", format)
		}
	}
	if _, err := DecodePayload([]byte(`{"format":"plain","text":"ok","extra":true}`)); err == nil {
		t.Fatal("accepted ambiguous payload")
	}
}
