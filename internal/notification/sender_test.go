package notification

import (
	"context"
	"strings"
	"testing"

	utiltelegram "github.com/useryege/athena/util/telegram"
)

type fakeTelegramSenderClient struct {
	utiltelegram.Client
	request       utiltelegram.SendMessageRequest
	topicRequests []utiltelegram.CreateForumTopicRequest
	topicIDs      []int
	topicErr      error
}

func (f *fakeTelegramSenderClient) SendMessage(_ context.Context, request utiltelegram.SendMessageRequest) (*utiltelegram.SendMessageResponse, error) {
	f.request = request
	return &utiltelegram.SendMessageResponse{MessageID: 123}, nil
}

func (f *fakeTelegramSenderClient) CreateForumTopic(_ context.Context, request utiltelegram.CreateForumTopicRequest) (*utiltelegram.ForumTopic, error) {
	f.topicRequests = append(f.topicRequests, request)
	if f.topicErr != nil {
		return nil, f.topicErr
	}
	topicID := 100 + len(f.topicRequests)
	if len(f.topicIDs) >= len(f.topicRequests) {
		topicID = f.topicIDs[len(f.topicRequests)-1]
	}
	return &utiltelegram.ForumTopic{MessageThreadID: topicID, Name: request.Name}, nil
}

func TestTelegramSenderRoutesTopicToThreadID(t *testing.T) {
	client := &fakeTelegramSenderClient{}
	sender, err := NewTelegramSender(client, map[string]int{
		NotificationTopicToken:     111,
		NotificationTopicPolyMover: 222,
	})
	if err != nil {
		t.Fatalf("NewTelegramSender: %v", err)
	}

	messageID, err := sender.Send(context.Background(), SendRequest{Topic: NotificationTopicPolyMover, Text: "hello"})
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

func TestTelegramSenderProvisionTopicsCreatesThreadsAtomically(t *testing.T) {
	client := &fakeTelegramSenderClient{topicIDs: []int{111, 222, 333}}
	sender, err := NewTelegramSender(client, nil)
	if err != nil {
		t.Fatalf("NewTelegramSender: %v", err)
	}
	if err := sender.ProvisionTopics(context.Background(), DefaultTopicConfigs()); err != nil {
		t.Fatalf("ProvisionTopics: %v", err)
	}
	if len(client.topicRequests) != 3 ||
		client.topicRequests[0].Name != "[TOKEN] 代币通知" ||
		client.topicRequests[1].Name != "[POLY] 市场异动" ||
		client.topicRequests[2].Name != "[POLY] 开赛通知" {
		t.Fatalf("topic requests = %#v, want default topics", client.topicRequests)
	}
	if _, err := sender.Send(context.Background(), SendRequest{Topic: " poly-mover ", Text: "hello"}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if client.request.MessageThreadID != 222 {
		t.Fatalf("thread id = %d, want provisioned poly mover thread", client.request.MessageThreadID)
	}
}

func TestTelegramSenderRejectsUnknownTopic(t *testing.T) {
	sender, err := NewTelegramSender(&fakeTelegramSenderClient{}, map[string]int{NotificationTopicToken: 111})
	if err != nil {
		t.Fatalf("NewTelegramSender: %v", err)
	}
	_, err = sender.Send(context.Background(), SendRequest{Topic: NotificationTopicPolyKickoff, Text: "hello"})
	if err == nil || !strings.Contains(err.Error(), "not initialized") {
		t.Fatalf("Send error = %v, want uninitialized topic", err)
	}
}
