package notification

import (
	"strings"

	"github.com/useryege/athena/internal/notification/apiclient"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	TelegramChatTest = "test"
	TelegramChatProd = "prod"
)

var supportedTelegramChats = []string{TelegramChatTest, TelegramChatProd}

func telegramChatFromEnum(value apiclient.TelegramChat) (string, error) {
	switch value {
	case apiclient.TelegramChat_TELEGRAM_CHAT_TEST:
		return TelegramChatTest, nil
	case apiclient.TelegramChat_TELEGRAM_CHAT_PROD:
		return TelegramChatProd, nil
	case apiclient.TelegramChat_TELEGRAM_CHAT_UNSPECIFIED:
		return "", status.Error(codes.InvalidArgument, "telegram_chat is required")
	default:
		return "", status.Error(codes.InvalidArgument, "telegram_chat is invalid")
	}
}

func normalizeTelegramChatFilter(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", nil
	}
	switch value {
	case TelegramChatTest, TelegramChatProd:
		return value, nil
	default:
		return "", status.Error(codes.InvalidArgument, "telegram_chat filter is invalid")
	}
}
