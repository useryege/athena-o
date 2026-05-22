package deepseek

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClientRequiresAPIKey(t *testing.T) {
	_, err := NewClient(Config{})
	if err == nil || !strings.Contains(err.Error(), "api key is required") {
		t.Fatalf("NewClient error = %v, want api key error", err)
	}
}

func TestConfigWithDefaults(t *testing.T) {
	config := Config{}.WithDefaults()
	if config.BaseURL != DefaultBaseURL {
		t.Fatalf("BaseURL = %q, want %q", config.BaseURL, DefaultBaseURL)
	}
	if config.Model != DefaultModel {
		t.Fatalf("Model = %q, want %q", config.Model, DefaultModel)
	}
	if config.Timeout != DefaultTimeout {
		t.Fatalf("Timeout = %s, want %s", config.Timeout, DefaultTimeout)
	}
	if config.MaxTokens != DefaultMaxTokens {
		t.Fatalf("MaxTokens = %d, want %d", config.MaxTokens, DefaultMaxTokens)
	}
	if config.Temperature != DefaultTemperature {
		t.Fatalf("Temperature = %f, want %f", config.Temperature, DefaultTemperature)
	}
}

func TestCreateChatCompletion(t *testing.T) {
	var gotPath string
	var gotAuthorization string
	var gotContentType string
	var gotRequest ChatCompletionRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuthorization = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","model":"deepseek-v4-flash","choices":[{"message":{"role":"assistant","content":"  report text  "}}]}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{
		BaseURL:     server.URL,
		APIKey:      "secret",
		Model:       "deepseek-v4-flash",
		MaxTokens:   123,
		Temperature: 0.4,
	})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	resp, err := client.CreateChatCompletion(context.Background(), ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("CreateChatCompletion: %v", err)
	}
	if resp.Content != "report text" {
		t.Fatalf("Content = %q, want %q", resp.Content, "report text")
	}
	if gotPath != "/chat/completions" {
		t.Fatalf("path = %q, want /chat/completions", gotPath)
	}
	if gotAuthorization != "Bearer secret" {
		t.Fatalf("Authorization = %q, want bearer token", gotAuthorization)
	}
	if gotContentType != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", gotContentType)
	}
	if gotRequest.Model != "deepseek-v4-flash" {
		t.Fatalf("request model = %q", gotRequest.Model)
	}
	if gotRequest.MaxTokens != 123 {
		t.Fatalf("request max tokens = %d", gotRequest.MaxTokens)
	}
	if gotRequest.Temperature == nil || *gotRequest.Temperature != 0.4 {
		t.Fatalf("request temperature = %v", gotRequest.Temperature)
	}
}

func TestCreateChatCompletionErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		want       string
	}{
		{name: "http error", statusCode: http.StatusUnauthorized, body: `unauthorized`, want: "401"},
		{name: "no choices", statusCode: http.StatusOK, body: `{"choices":[]}`, want: "no choices"},
		{name: "empty content", statusCode: http.StatusOK, body: `{"choices":[{"message":{"content":"   "}}]}`, want: "content is empty"},
		{name: "invalid json", statusCode: http.StatusOK, body: `{`, want: "decode"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := NewClient(Config{BaseURL: server.URL, APIKey: "secret"})
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			_, err = client.CreateChatCompletion(context.Background(), ChatCompletionRequest{
				Messages: []ChatMessage{{Role: "user", Content: "hello"}},
			})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want contains %q", err, tt.want)
			}
		})
	}
}

func TestCreateChatCompletionContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client, err := NewClient(Config{BaseURL: server.URL, APIKey: "secret", Timeout: time.Second})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = client.CreateChatCompletion(ctx, ChatCompletionRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hello"}},
	})
	if err == nil || !strings.Contains(err.Error(), "context canceled") {
		t.Fatalf("error = %v, want context canceled", err)
	}
}
