package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientRequiresConfig(t *testing.T) {
	_, err := NewClient(Config{})
	if err == nil || !strings.Contains(err.Error(), "bot token is required") {
		t.Fatalf("NewClient error = %v, want bot token error", err)
	}

	_, err = NewClient(Config{BotToken: "token"})
	if err == nil || !strings.Contains(err.Error(), "chat id is required") {
		t.Fatalf("NewClient error = %v, want chat id error", err)
	}
}

func TestConfigWithDefaults(t *testing.T) {
	config := Config{BotToken: " token ", ChatID: " chat ", BaseURL: "https://api.telegram.org/", Timeout: -1}.WithDefaults()
	if config.BotToken != "token" {
		t.Fatalf("BotToken = %q, want trimmed token", config.BotToken)
	}
	if config.ChatID != "chat" {
		t.Fatalf("ChatID = %q, want trimmed chat", config.ChatID)
	}
	if config.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", config.BaseURL, DefaultBaseURL)
	}
	if config.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %s, want %s", config.Timeout, DefaultTimeout)
	}
}

func TestSendMessage(t *testing.T) {
	var gotPath string
	var gotChatID string
	var gotText string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := r.ParseMultipartForm(1024 * 1024); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		gotChatID = r.FormValue("chat_id")
		gotText = r.FormValue("text")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":123,"date":1,"chat":{"id":1,"type":"private"}}}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BotToken: "token", ChatID: "chat-1", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	resp, err := client.SendMessage(context.Background(), SendMessageRequest{Text: " hello "})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if resp.MessageID != 123 {
		t.Fatalf("MessageID = %d, want 123", resp.MessageID)
	}
	if gotPath != "/bottoken/sendMessage" {
		t.Fatalf("path = %q, want /bottoken/sendMessage", gotPath)
	}
	if gotChatID != "chat-1" {
		t.Fatalf("chat_id = %q, want chat-1", gotChatID)
	}
	if gotText != "hello" {
		t.Fatalf("text = %q, want hello", gotText)
	}
}

func TestSendMessageErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       string
	}{
		{name: "api error", statusCode: http.StatusOK, body: `{"ok":false,"error_code":400,"description":"Bad Request: chat not found"}`, want: "bad request"},
		{name: "non json", statusCode: http.StatusInternalServerError, body: `server unavailable`, want: "decode response body"},
		{name: "invalid result", statusCode: http.StatusOK, body: `{"ok":true,"result":{`, want: "decode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := NewClient(Config{BotToken: "token", ChatID: "chat-1", BaseURL: server.URL})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			_, err = client.SendMessage(context.Background(), SendMessageRequest{Text: "hello"})
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.want)) {
				t.Fatalf("error = %v, want contains %q", err, tt.want)
			}
		})
	}
}

func TestSendMessageRequiresText(t *testing.T) {
	client, err := NewClient(Config{BotToken: "token", ChatID: "chat-1"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.SendMessage(context.Background(), SendMessageRequest{Text: " "})
	if err == nil || !strings.Contains(err.Error(), "text is required") {
		t.Fatalf("error = %v, want text required", err)
	}
}

func TestSendMessageContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client, err := NewClient(Config{BotToken: "token", ChatID: "chat-1", BaseURL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.SendMessage(ctx, SendMessageRequest{Text: "hello"})
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("error = %v, want context canceled", err)
	}
}
