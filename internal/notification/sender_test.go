package notification

import (
	"context"
	"strings"
	"testing"

	utiltelegram "github.com/useryege/athena/util/telegram"
)

type fakeTelegramSenderClient struct {
	utiltelegram.Client
	request utiltelegram.SendMessageRequest
}

func (f *fakeTelegramSenderClient) SendMessage(_ context.Context, request utiltelegram.SendMessageRequest) (*utiltelegram.SendMessageResponse, error) {
	f.request = request
	return &utiltelegram.SendMessageResponse{MessageID: 123}, nil
}

func TestTelegramSenderRoutesTopicToThreadID(t *testing.T) {
	client := &fakeTelegramSenderClient{}
	sender, err := NewTelegramSender(client, map[string]int{
		NotificationTopicToken: 111,
		NotificationTopicPoly:  222,
	})
	if err != nil {
		t.Fatalf("NewTelegramSender: %v", err)
	}

	messageID, err := sender.Send(context.Background(), SendRequest{Topic: NotificationTopicPoly, Text: "hello"})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if messageID != "123" {
		t.Fatalf("messageID = %q, want 123", messageID)
	}
	if client.request.MessageThreadID != 222 || client.request.Text != "hello" {
		t.Fatalf("telegram request = %#v, want poly thread", client.request)
	}
}

func TestTelegramSenderRequiresTopicThreads(t *testing.T) {
	_, err := NewTelegramSender(&fakeTelegramSenderClient{}, map[string]int{NotificationTopicToken: 111})
	if err == nil || !strings.Contains(err.Error(), "poly") {
		t.Fatalf("NewTelegramSender error = %v, want missing poly thread", err)
	}
}
