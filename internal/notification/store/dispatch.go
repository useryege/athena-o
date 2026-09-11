package store

import (
	"context"
	"fmt"
	"github.com/useryege/athena/internal/notification/delivery"
)

// DispatchItem keeps pending future work visible; there is no global LIMIT biased toward one owner.
type DispatchItem struct {
	Candidate                                       delivery.Candidate
	Source, Severity, Title, Body, Link, SystemChat string
	ThreadID                                        int
	BindingRevision                                 int64
}

func (s *SQLStore) DispatchItems(ctx context.Context, kind string) ([]DispatchItem, error) {
	var items []DispatchItem
	switch kind {
	case "account":
		rows, err := s.queries.ListDispatchAccounts(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			items = append(items, DispatchItem{Candidate: delivery.Candidate{Ref: delivery.WorkRef{Kind: kind, ID: r.ID}, OwnerID: uuidString(r.AccountID), ChatID: r.TelegramChatID, NotBefore: r.NextAttemptAt.Time}, Source: r.Source, Severity: r.Severity, Title: r.Title.String, Body: r.Body, Link: r.Link.String, BindingRevision: r.BindingRevision})
		}
	case "system":
		rows, err := s.queries.ListDispatchSystems(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			items = append(items, DispatchItem{Candidate: delivery.Candidate{Ref: delivery.WorkRef{Kind: kind, ID: r.ID}, OwnerID: "system:" + r.TelegramChat, Group: true, NotBefore: r.NextAttemptAt.Time}, Source: r.Source, Severity: r.Severity, Title: r.Title.String, Body: r.Body, Link: r.Link.String, SystemChat: r.TelegramChat, ThreadID: int(r.MessageThreadID)})
		}
	case "reply":
		rows, err := s.queries.ListDispatchReplies(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			owner := uuidString(r.AccountID)
			if owner == "" {
				owner = fmt.Sprintf("reply:%d", r.TelegramChatID)
			}
			items = append(items, DispatchItem{Candidate: delivery.Candidate{Ref: delivery.WorkRef{Kind: kind, ID: r.ID}, OwnerID: owner, ChatID: r.TelegramChatID, NotBefore: r.NextAttemptAt.Time}, Body: r.Body, BindingRevision: r.BindingRevision.Int64})
		}
	default:
		return nil, fmt.Errorf("unsupported notification source %q", kind)
	}
	return items, nil
}
