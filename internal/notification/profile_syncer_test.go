package notification

import (
	"context"
	"errors"
	"testing"

	utiltelegram "github.com/useryege/athena/util/telegram"
)

type fakeTelegramClient struct {
	botSyncCalls  int
	chatSyncCalls int
	chatSyncErr   error
}

func (f *fakeTelegramClient) GetMe(context.Context) (*utiltelegram.BotIdentity, error) {
	return &utiltelegram.BotIdentity{}, nil
}

func (f *fakeTelegramClient) GetMyName(context.Context) (string, error) {
	return "", nil
}

func (f *fakeTelegramClient) SetMyName(context.Context, string) error {
	return nil
}

func (f *fakeTelegramClient) GetMyDescription(context.Context) (string, error) {
	return "", nil
}

func (f *fakeTelegramClient) SetMyDescription(context.Context, string) error {
	return nil
}

func (f *fakeTelegramClient) GetMyShortDescription(context.Context) (string, error) {
	return "", nil
}

func (f *fakeTelegramClient) SetMyShortDescription(context.Context, string) error {
	return nil
}

func (f *fakeTelegramClient) SetMyProfilePhoto(context.Context, utiltelegram.SetMyProfilePhotoRequest) error {
	return nil
}

func (f *fakeTelegramClient) EnsureBotProfile(context.Context, utiltelegram.BotProfileConfig) (*utiltelegram.BotProfileSyncResult, error) {
	f.botSyncCalls++
	return &utiltelegram.BotProfileSyncResult{Bot: utiltelegram.BotIdentity{ID: 42, IsBot: true, Username: "athena_bot"}}, nil
}

func (f *fakeTelegramClient) GetChat(context.Context) (*utiltelegram.ChatInfo, error) {
	return &utiltelegram.ChatInfo{}, nil
}

func (f *fakeTelegramClient) GetChatMember(context.Context, int64) (*utiltelegram.ChatMemberInfo, error) {
	return &utiltelegram.ChatMemberInfo{}, nil
}

func (f *fakeTelegramClient) SetChatTitle(context.Context, string) error {
	return nil
}

func (f *fakeTelegramClient) SetChatDescription(context.Context, string) error {
	return nil
}

func (f *fakeTelegramClient) SetChatPhoto(context.Context, utiltelegram.SetChatPhotoRequest) error {
	return nil
}

func (f *fakeTelegramClient) EnsureChatProfile(context.Context, utiltelegram.ChatProfileConfig) (*utiltelegram.ChatProfileSyncResult, error) {
	f.chatSyncCalls++
	if f.chatSyncErr != nil {
		return nil, f.chatSyncErr
	}
	return &utiltelegram.ChatProfileSyncResult{ChatID: -1001, ChatType: "supergroup", PhotoUpdated: true}, nil
}

func (f *fakeTelegramClient) SendMessage(context.Context, utiltelegram.SendMessageRequest) (*utiltelegram.SendMessageResponse, error) {
	return &utiltelegram.SendMessageResponse{}, nil
}

func (f *fakeTelegramClient) CreateForumTopic(context.Context, utiltelegram.CreateForumTopicRequest) (*utiltelegram.ForumTopic, error) {
	return &utiltelegram.ForumTopic{}, nil
}

func TestTelegramProfileSyncerSyncsBotAndChatProfiles(t *testing.T) {
	client := &fakeTelegramClient{}
	syncer := NewTelegramProfileSyncer(client, utiltelegram.BotProfileConfig{}, utiltelegram.ChatProfileConfig{})

	if err := syncer.SyncProfile(context.Background()); err != nil {
		t.Fatalf("SyncProfile: %v", err)
	}
	if client.botSyncCalls != 1 || client.chatSyncCalls != 1 {
		t.Fatalf("sync calls bot=%d chat=%d, want 1 each", client.botSyncCalls, client.chatSyncCalls)
	}
}

func TestTelegramProfileSyncerReturnsChatSyncFailure(t *testing.T) {
	client := &fakeTelegramClient{chatSyncErr: errors.New("chat profile unavailable")}
	syncer := NewTelegramProfileSyncer(client, utiltelegram.BotProfileConfig{}, utiltelegram.ChatProfileConfig{})

	err := syncer.SyncProfile(context.Background())
	if err == nil || !errors.Is(err, client.chatSyncErr) {
		t.Fatalf("SyncProfile error = %v, want chat sync error", err)
	}
	if client.botSyncCalls != 1 || client.chatSyncCalls != 1 {
		t.Fatalf("sync calls bot=%d chat=%d, want 1 each", client.botSyncCalls, client.chatSyncCalls)
	}
}
