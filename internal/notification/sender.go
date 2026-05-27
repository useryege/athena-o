package notification

import (
	"context"
	"fmt"
	"strconv"

	utiltelegram "github.com/useryege/athena/util/telegram"
)

type Sender interface {
	Send(ctx context.Context, text string) (string, error)
}

type TelegramSender struct {
	client utiltelegram.Client
}

func NewTelegramSender(client utiltelegram.Client) *TelegramSender {
	return &TelegramSender{client: client}
}

func (s *TelegramSender) Send(ctx context.Context, text string) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("telegram client is required")
	}
	resp, err := s.client.SendMessage(ctx, utiltelegram.SendMessageRequest{Text: text})
	if err != nil {
		return "", err
	}
	return strconv.Itoa(resp.MessageID), nil
}
