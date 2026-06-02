package telegram

import (
	"context"
	"io"
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

func TestEnsureBotProfileUpdatesDriftAndUploadsPhoto(t *testing.T) {
	var calls []string
	var gotName string
	var gotDescription string
	var gotShortDescription string
	var gotPhoto string
	var gotPhotoData string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/bottoken/getMe":
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":42,"is_bot":true,"first_name":"Old","username":"athena_bot"}}`))
		case "/bottoken/getMyName":
			requireEmptyRequestBody(t, r)
			_, _ = w.Write([]byte(`{"ok":true,"result":{"name":"Old name"}}`))
		case "/bottoken/setMyName":
			parseMultipartForm(t, r)
			gotName = r.FormValue("name")
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		case "/bottoken/getMyDescription":
			requireEmptyRequestBody(t, r)
			_, _ = w.Write([]byte(`{"ok":true,"result":{"description":"Old description"}}`))
		case "/bottoken/setMyDescription":
			parseMultipartForm(t, r)
			gotDescription = r.FormValue("description")
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		case "/bottoken/getMyShortDescription":
			requireEmptyRequestBody(t, r)
			_, _ = w.Write([]byte(`{"ok":true,"result":{"short_description":"Old short"}}`))
		case "/bottoken/setMyShortDescription":
			parseMultipartForm(t, r)
			gotShortDescription = r.FormValue("short_description")
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		case "/bottoken/setMyProfilePhoto":
			parseMultipartForm(t, r)
			gotPhoto = r.FormValue("photo")
			file, _, err := r.FormFile("telegram-bot-avatar.jpg")
			if err != nil {
				t.Fatalf("FormFile: %v", err)
			}
			defer file.Close()
			data, err := io.ReadAll(file)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			gotPhotoData = string(data)
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BotToken: "token", ChatID: "chat-1", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	result, err := client.EnsureBotProfile(context.Background(), BotProfileConfig{
		Name:             "ATHENA",
		Description:      "ATHENA notification bot for operational alerts and system updates.",
		ShortDescription: "ATHENA operational alerts",
		ProfilePhoto: SetMyProfilePhotoRequest{
			Filename: "telegram-bot-avatar.jpg",
			Data:     []byte("photo-bytes"),
		},
	})
	if err != nil {
		t.Fatalf("EnsureBotProfile: %v", err)
	}
	if result.Bot.ID != 42 || result.Bot.Username != "athena_bot" || !result.Bot.IsBot {
		t.Fatalf("bot identity = %#v", result.Bot)
	}
	if !result.NameUpdated || !result.DescriptionUpdated || !result.ShortDescriptionUpdated || !result.ProfilePhotoUpdated {
		t.Fatalf("sync result = %#v, want all updates", result)
	}
	if gotName != "ATHENA" {
		t.Fatalf("set name = %q", gotName)
	}
	if gotDescription != "ATHENA notification bot for operational alerts and system updates." {
		t.Fatalf("set description = %q", gotDescription)
	}
	if gotShortDescription != "ATHENA operational alerts" {
		t.Fatalf("set short description = %q", gotShortDescription)
	}
	if !strings.Contains(gotPhoto, `"type":"static"`) || !strings.Contains(gotPhoto, `"photo":"attach://telegram-bot-avatar.jpg"`) {
		t.Fatalf("photo form value = %q, want static attach photo", gotPhoto)
	}
	if gotPhotoData != "photo-bytes" {
		t.Fatalf("photo data = %q", gotPhotoData)
	}
	wantCalls := []string{
		"/bottoken/getMe",
		"/bottoken/getMyName",
		"/bottoken/setMyName",
		"/bottoken/getMyDescription",
		"/bottoken/setMyDescription",
		"/bottoken/getMyShortDescription",
		"/bottoken/setMyShortDescription",
		"/bottoken/setMyProfilePhoto",
	}
	if strings.Join(calls, ",") != strings.Join(wantCalls, ",") {
		t.Fatalf("calls = %v, want %v", calls, wantCalls)
	}
}

func TestEnsureBotProfileSkipsMatchingTextAndAlwaysUploadsPhoto(t *testing.T) {
	var setNameCalls int
	var setPhotoCalls int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/bottoken/getMe":
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":42,"is_bot":true,"first_name":"ATHENA","username":"athena_bot"}}`))
		case "/bottoken/getMyName":
			requireEmptyRequestBody(t, r)
			_, _ = w.Write([]byte(`{"ok":true,"result":{"name":"ATHENA"}}`))
		case "/bottoken/setMyName":
			setNameCalls++
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		case "/bottoken/getMyDescription":
			requireEmptyRequestBody(t, r)
			_, _ = w.Write([]byte(`{"ok":true,"result":{"description":"ATHENA notification bot for operational alerts and system updates."}}`))
		case "/bottoken/getMyShortDescription":
			requireEmptyRequestBody(t, r)
			_, _ = w.Write([]byte(`{"ok":true,"result":{"short_description":"ATHENA operational alerts"}}`))
		case "/bottoken/setMyProfilePhoto":
			setPhotoCalls++
			parseMultipartForm(t, r)
			_, _ = w.Write([]byte(`{"ok":true,"result":true}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client, err := NewClient(Config{BotToken: "token", ChatID: "chat-1", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	result, err := client.EnsureBotProfile(context.Background(), defaultTestBotProfileConfig())
	if err != nil {
		t.Fatalf("EnsureBotProfile: %v", err)
	}
	if result.NameUpdated || result.DescriptionUpdated || result.ShortDescriptionUpdated || !result.ProfilePhotoUpdated {
		t.Fatalf("sync result = %#v, want only photo update", result)
	}
	if setNameCalls != 0 {
		t.Fatalf("setMyName calls = %d, want 0", setNameCalls)
	}
	if setPhotoCalls != 1 {
		t.Fatalf("setMyProfilePhoto calls = %d, want 1", setPhotoCalls)
	}
}

func TestEnsureBotProfilePropagatesAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/bottoken/getMe" {
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":42,"is_bot":true,"first_name":"ATHENA","username":"athena_bot"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: name unavailable"}`))
	}))
	defer server.Close()

	client, err := NewClient(Config{BotToken: "token", ChatID: "chat-1", BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.EnsureBotProfile(context.Background(), defaultTestBotProfileConfig())
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "name unavailable") {
		t.Fatalf("error = %v, want name unavailable", err)
	}
}

func TestEnsureBotProfileValidatesConfig(t *testing.T) {
	client, err := NewClient(Config{BotToken: "token", ChatID: "chat-1"})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	tests := []struct {
		name   string
		config BotProfileConfig
		want   string
	}{
		{name: "missing name", config: BotProfileConfig{ProfilePhoto: SetMyProfilePhotoRequest{Filename: "avatar.jpg", Data: []byte("x")}}, want: "name is required"},
		{name: "long name", config: BotProfileConfig{Name: strings.Repeat("a", 65), ProfilePhoto: SetMyProfilePhotoRequest{Filename: "avatar.jpg", Data: []byte("x")}}, want: "name must be at most 64"},
		{name: "long short description", config: BotProfileConfig{Name: "ATHENA", ShortDescription: strings.Repeat("a", 121), ProfilePhoto: SetMyProfilePhotoRequest{Filename: "avatar.jpg", Data: []byte("x")}}, want: "short description must be at most 120"},
		{name: "long description", config: BotProfileConfig{Name: "ATHENA", Description: strings.Repeat("a", 513), ProfilePhoto: SetMyProfilePhotoRequest{Filename: "avatar.jpg", Data: []byte("x")}}, want: "description must be at most 512"},
		{name: "missing photo", config: BotProfileConfig{Name: "ATHENA", ProfilePhoto: SetMyProfilePhotoRequest{Filename: "avatar.jpg"}}, want: "photo data is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.EnsureBotProfile(context.Background(), tt.config)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want contains %q", err, tt.want)
			}
		})
	}
}

func defaultTestBotProfileConfig() BotProfileConfig {
	return BotProfileConfig{
		Name:             "ATHENA",
		Description:      "ATHENA notification bot for operational alerts and system updates.",
		ShortDescription: "ATHENA operational alerts",
		ProfilePhoto: SetMyProfilePhotoRequest{
			Filename: "telegram-bot-avatar.jpg",
			Data:     []byte("photo-bytes"),
		},
	}
}

func parseMultipartForm(t *testing.T, r *http.Request) {
	t.Helper()
	if err := r.ParseMultipartForm(1024 * 1024); err != nil {
		t.Fatalf("ParseMultipartForm: %v", err)
	}
}

func requireEmptyRequestBody(t *testing.T, r *http.Request) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("ReadAll request body: %v", err)
	}
	if len(body) != 0 {
		t.Fatalf("%s request body length = %d, want 0; body prefix = %q", r.URL.Path, len(body), string(body[:min(len(body), 80)]))
	}
}
