package notification

import (
	"context"
	"errors"
	"github.com/useryege/athena/internal/notification/delivery"
	notificationstore "github.com/useryege/athena/internal/notification/store"
	"sync"
	"time"
)

type notificationSource struct {
	service *Service
	kind    string
	mu      sync.Mutex
	items   map[delivery.WorkRef]notificationstore.DispatchItem
}

func (s *Service) workSources() []WorkSource {
	sources := make([]WorkSource, 0, 3)
	for _, kind := range []string{"account", "system", "reply"} {
		sources = append(sources, &notificationSource{service: s, kind: kind, items: make(map[delivery.WorkRef]notificationstore.DispatchItem)})
	}
	return sources
}
func (s *notificationSource) Ready(ctx context.Context, now time.Time) ([]delivery.Candidate, error) {
	items, err := s.service.store.DispatchItems(ctx, s.kind)
	if err != nil {
		return nil, err
	}
	candidates := make([]delivery.Candidate, 0, len(items))
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = make(map[delivery.WorkRef]notificationstore.DispatchItem, len(items))
	for _, item := range items {
		if s.kind == "system" {
			chat, err := s.service.sender.SystemChatID(item.SystemChat)
			if err != nil {
				return nil, err
			}
			item.Candidate.ChatID = chat
		}
		s.items[item.Candidate.Ref] = item
		candidates = append(candidates, item.Candidate)
	}
	return candidates, nil
}
func (s *notificationSource) Dispatch(ctx context.Context, c delivery.Candidate, onStarted func(time.Time)) error {
	s.mu.Lock()
	item, ok := s.items[c.Ref]
	s.mu.Unlock()
	if !ok {
		return ErrDispatchDeferred
	}
	text := item.Body
	if s.kind != "reply" {
		text = renderNotificationMessage(sendNotificationParams{source: item.Source, severity: item.Severity, title: item.Title, body: item.Body, link: item.Link}).Text
	}
	outcome, err := s.service.sendPermittedNotification(ctx, c, SendRequest{TelegramChatID: c.ChatID, MessageThreadID: item.ThreadID, Text: text}, onStarted)
	if err != nil {
		if errors.Is(err, notificationstore.ErrDeliveryNotEligible) {
			return ErrDispatchDeferred
		}
		return err
	}
	if outcome.Code == "recipient_unreachable" && (s.kind == "account" || (s.kind == "reply" && item.BindingRevision > 0)) {
		if err = s.service.store.MarkTelegramBindingUnreachable(ctx, c.OwnerID, c.ChatID, item.BindingRevision, outcome.Code); err != nil {
			return err
		}
	}
	if outcome.RetryAfter > 0 {
		return &RateLimitError{RetryAfter: outcome.RetryAfter}
	}
	return nil
}
