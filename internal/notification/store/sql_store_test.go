package store

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestListDeliveriesBuildsFiltersAndPagination(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	createdAt := time.Date(2026, time.May, 27, 12, 0, 0, 0, time.UTC)
	sentAt := time.Date(2026, time.May, 27, 12, 1, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM notification_deliveries WHERE status = \$1 AND severity = \$2 AND source = \$3 AND \(title ILIKE \$4 OR body ILIKE \$4 OR error_message ILIKE \$4 OR provider_message_id ILIKE \$4\)`).
		WithArgs("sent", "warning", "worm", "%scan%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	mock.ExpectQuery(`(?s)SELECT id, source, severity, COALESCE\(title, ''\), body, COALESCE\(link, ''\), channel, status, provider_message_id, error_message, created_at, sent_at.*LIMIT \$5 OFFSET \$6`).
		WithArgs("sent", "warning", "worm", "%scan%", 20, 20).
		WillReturnRows(sqlmock.NewRows([]string{
			"id",
			"source",
			"severity",
			"title",
			"body",
			"link",
			"channel",
			"status",
			"provider_message_id",
			"error_message",
			"created_at",
			"sent_at",
		}).AddRow(int64(7), "worm", "warning", "scan", "body", "", "telegram", "sent", "123", nil, createdAt, sentAt))

	items, total, err := NewSQLStore(db).ListDeliveries(context.Background(), ListDeliveriesOptions{
		Page:     2,
		PageSize: 20,
		Status:   "sent",
		Severity: "warning",
		Source:   "worm",
		Keyword:  "scan",
	})
	if err != nil {
		t.Fatalf("ListDeliveries: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].ID != 7 || items[0].ProviderMessageID != "123" {
		t.Fatalf("result = total %d items %#v, want one mapped delivery", total, items)
	}
	if items[0].CreatedAt != createdAt.Format(time.RFC3339) || items[0].SentAt != sentAt.Format(time.RFC3339) {
		t.Fatalf("timestamps = %q/%q, want RFC3339", items[0].CreatedAt, items[0].SentAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestGetDeliveryNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT id, source, severity, COALESCE\(title, ''\), body, COALESCE\(link, ''\), channel, status, provider_message_id, error_message, created_at, sent_at`).
		WithArgs(int64(404)).
		WillReturnError(sql.ErrNoRows)

	_, err = NewSQLStore(db).GetDelivery(context.Background(), 404)
	if err != sql.ErrNoRows {
		t.Fatalf("GetDelivery error = %v, want sql.ErrNoRows", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
