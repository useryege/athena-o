package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	notificationsqlc "github.com/useryege/athena/internal/notification/store/sqlc"
)

type fakeNotificationQuerier struct {
	countDeliveriesErr error
	createDeliveryErr  error
	getDeliveryErr     error
	markSentErr        error
	markFailedErr      error
	claimDeliveriesErr error
	scheduleRetryErr   error

	countDeliveriesParams notificationsqlc.CountDeliveriesParams
	listDeliveriesParams  notificationsqlc.ListDeliveriesParams
	createDeliveryParams  notificationsqlc.CreateDeliveryParams
	markSentParams        notificationsqlc.MarkDeliverySentParams
	markFailedParams      notificationsqlc.MarkDeliveryFailedParams
	claimDeliveriesParams notificationsqlc.ClaimPendingDeliveriesParams
	scheduleRetryParams   notificationsqlc.ScheduleDeliveryRetryParams

	countDeliveriesResult int64
	createDeliveryResult  notificationsqlc.CreateDeliveryRow
	getDeliveryResult     notificationsqlc.GetDeliveryRow
	listDeliveriesResult  []notificationsqlc.ListDeliveriesRow
	claimDeliveriesResult []notificationsqlc.ClaimPendingDeliveriesRow
}

func (f *fakeNotificationQuerier) ClaimPendingDeliveries(_ context.Context, arg notificationsqlc.ClaimPendingDeliveriesParams) ([]notificationsqlc.ClaimPendingDeliveriesRow, error) {
	f.claimDeliveriesParams = arg
	return f.claimDeliveriesResult, f.claimDeliveriesErr
}

func (f *fakeNotificationQuerier) CountDeliveries(_ context.Context, arg notificationsqlc.CountDeliveriesParams) (int64, error) {
	f.countDeliveriesParams = arg
	return f.countDeliveriesResult, f.countDeliveriesErr
}

func (f *fakeNotificationQuerier) CreateDelivery(_ context.Context, arg notificationsqlc.CreateDeliveryParams) (notificationsqlc.CreateDeliveryRow, error) {
	f.createDeliveryParams = arg
	return f.createDeliveryResult, f.createDeliveryErr
}

func (f *fakeNotificationQuerier) GetDelivery(context.Context, int64) (notificationsqlc.GetDeliveryRow, error) {
	return f.getDeliveryResult, f.getDeliveryErr
}

func (f *fakeNotificationQuerier) ListDeliveries(_ context.Context, arg notificationsqlc.ListDeliveriesParams) ([]notificationsqlc.ListDeliveriesRow, error) {
	f.listDeliveriesParams = arg
	return f.listDeliveriesResult, nil
}

func (f *fakeNotificationQuerier) MarkDeliveryFailed(_ context.Context, arg notificationsqlc.MarkDeliveryFailedParams) error {
	f.markFailedParams = arg
	return f.markFailedErr
}

func (f *fakeNotificationQuerier) MarkDeliverySent(_ context.Context, arg notificationsqlc.MarkDeliverySentParams) error {
	f.markSentParams = arg
	return f.markSentErr
}

func (f *fakeNotificationQuerier) ScheduleDeliveryRetry(_ context.Context, arg notificationsqlc.ScheduleDeliveryRetryParams) error {
	f.scheduleRetryParams = arg
	return f.scheduleRetryErr
}

func TestCreateDeliveryUsesQuerier(t *testing.T) {
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	querier := &fakeNotificationQuerier{
		createDeliveryResult: notificationsqlc.CreateDeliveryRow{
			ID:        7,
			Source:    "worm",
			Severity:  "warning",
			Title:     "Scan finished",
			Body:      "Contract risk changed",
			Link:      "https://example.com",
			Channel:   "telegram",
			Status:    "pending",
			Topic:     "token",
			CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		},
	}

	item, err := NewSQLStoreWithQuerier(querier).CreateDelivery(context.Background(), CreateDeliveryRequest{
		Source:   "worm",
		Severity: "warning",
		Title:    "Scan finished",
		Body:     "Contract risk changed",
		Link:     "https://example.com",
		Channel:  "telegram",
		Status:   "pending",
		Topic:    "token",
	})
	if err != nil {
		t.Fatalf("CreateDelivery: %v", err)
	}
	if item.ID != 7 || item.Title != "Scan finished" || item.Topic != "token" {
		t.Fatalf("item = %#v, want generated row mapping", item)
	}
	if querier.createDeliveryParams.Title.String != "Scan finished" || !querier.createDeliveryParams.Title.Valid || querier.createDeliveryParams.Topic != "token" {
		t.Fatalf("create params = %#v, want pgtype text", querier.createDeliveryParams)
	}
}

func TestListDeliveriesUsesQuerierFiltersAndPagination(t *testing.T) {
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	querier := &fakeNotificationQuerier{
		countDeliveriesResult: 1,
		listDeliveriesResult: []notificationsqlc.ListDeliveriesRow{{
			ID:        9,
			Source:    "application",
			Severity:  "error",
			Title:     "",
			Body:      "Deploy failed",
			Link:      "",
			Channel:   "telegram",
			Status:    "failed",
			Topic:     "poly",
			CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		}},
	}

	items, total, err := NewSQLStoreWithQuerier(querier).ListDeliveries(context.Background(), ListDeliveriesOptions{
		Page:     2,
		PageSize: 5,
		Status:   " failed ",
		Severity: " error ",
		Topic:    " poly ",
		Source:   " application ",
		Keyword:  " deploy ",
	})
	if err != nil {
		t.Fatalf("ListDeliveries: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != 9 || items[0].Topic != "poly" {
		t.Fatalf("items/total = %#v/%d, want one mapped delivery", items, total)
	}
	if querier.countDeliveriesParams.Status.String != "failed" || querier.countDeliveriesParams.Topic.String != "poly" || querier.countDeliveriesParams.Keyword.String != "%deploy%" {
		t.Fatalf("count params = %#v, want normalized filters", querier.countDeliveriesParams)
	}
	if querier.listDeliveriesParams.Limit != 5 || querier.listDeliveriesParams.Offset != 5 {
		t.Fatalf("list params = %#v, want page 2 offset", querier.listDeliveriesParams)
	}
}

func TestClaimPendingDeliveriesUsesQuerier(t *testing.T) {
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	querier := &fakeNotificationQuerier{
		claimDeliveriesResult: []notificationsqlc.ClaimPendingDeliveriesRow{{
			ID:        11,
			Source:    "worm",
			Severity:  "warning",
			Title:     "Scan finished",
			Body:      "Contract risk changed",
			Channel:   "telegram",
			Status:    "pending",
			Topic:     "token",
			CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
			Attempts:  2,
		}},
	}

	items, err := NewSQLStoreWithQuerier(querier).ClaimPendingDeliveries(context.Background(), ClaimDeliveriesOptions{
		Limit:       10,
		LockedBy:    "worker-1",
		LockTimeout: 2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("ClaimPendingDeliveries: %v", err)
	}
	if len(items) != 1 || items[0].ID != 11 || items[0].Attempts != 2 || items[0].Topic != "token" {
		t.Fatalf("items = %#v, want claimed delivery", items)
	}
	if querier.claimDeliveriesParams.Limit != 10 || querier.claimDeliveriesParams.LockedBy.String != "worker-1" || querier.claimDeliveriesParams.LockTimeout.Microseconds != (2*time.Minute).Microseconds() {
		t.Fatalf("claim params = %#v, want limit/worker/timeout", querier.claimDeliveriesParams)
	}
}

func TestGetDeliveryAndStatusUpdatesUseQuerier(t *testing.T) {
	querier := &fakeNotificationQuerier{getDeliveryErr: pgx.ErrNoRows}
	if _, err := NewSQLStoreWithQuerier(querier).GetDelivery(context.Background(), 404); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("GetDelivery error = %v, want pgx.ErrNoRows", err)
	}

	querier = &fakeNotificationQuerier{}
	store := NewSQLStoreWithQuerier(querier)
	if err := store.MarkDeliverySent(context.Background(), 7, "123"); err != nil {
		t.Fatalf("MarkDeliverySent: %v", err)
	}
	if querier.markSentParams.ID != 7 || querier.markSentParams.ProviderMessageID.String != "123" {
		t.Fatalf("sent params = %#v", querier.markSentParams)
	}
	if err := store.MarkDeliveryFailed(context.Background(), 7, "telegram unavailable"); err != nil {
		t.Fatalf("MarkDeliveryFailed: %v", err)
	}
	if querier.markFailedParams.ID != 7 || querier.markFailedParams.ErrorMessage.String != "telegram unavailable" {
		t.Fatalf("failed params = %#v", querier.markFailedParams)
	}
	nextAttemptAt := time.Date(2026, time.May, 30, 12, 1, 0, 0, time.UTC)
	if err := store.ScheduleDeliveryRetry(context.Background(), 7, "rate limited", nextAttemptAt); err != nil {
		t.Fatalf("ScheduleDeliveryRetry: %v", err)
	}
	if querier.scheduleRetryParams.ID != 7 || querier.scheduleRetryParams.ErrorMessage.String != "rate limited" || !querier.scheduleRetryParams.NextAttemptAt.Time.Equal(nextAttemptAt) {
		t.Fatalf("retry params = %#v", querier.scheduleRetryParams)
	}
}
