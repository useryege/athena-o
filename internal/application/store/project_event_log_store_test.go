package store

import (
	"context"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
)

func TestAddProjectEventLogDefaultsEmptyPayload(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x0000000000000000000000000000000000000301")
	occurredAt := time.Date(2026, 5, 28, 1, 2, 3, 0, time.UTC)

	mock.ExpectExec("INSERT INTO project_event_log").
		WithArgs(contract.Bytes(), int16(5), occurredAt, nil, "{}", "empty-payload").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.AddProjectEventLog(context.Background(), ProjectEventLog{
		Contract:       contract,
		EventType:      5,
		OccurredAt:     occurredAt,
		IdempotencyKey: "empty-payload",
	}); err != nil {
		t.Fatalf("add project event log: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestAddProjectEventLogRejectsInvalidJSONPayload(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	err = store.AddProjectEventLog(context.Background(), ProjectEventLog{
		Contract:       common.HexToAddress("0x0000000000000000000000000000000000000302"),
		EventType:      5,
		Payload:        "{",
		IdempotencyKey: "invalid-json",
	})
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
	if !strings.Contains(err.Error(), "not valid JSON") {
		t.Fatalf("error = %v, want invalid JSON message", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestAddProjectEventLogUsesIdempotentInsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x0000000000000000000000000000000000000303")
	occurredAt := time.Date(2026, 5, 28, 2, 3, 4, 0, time.UTC)
	insertPattern := regexp.QuoteMeta("ON CONFLICT (contract, idempotency_key) DO NOTHING")
	for i := 0; i < 2; i++ {
		mock.ExpectExec(insertPattern).
			WithArgs(contract.Bytes(), int16(5), occurredAt, "same event", `{"ok":true}`, "same-key").
			WillReturnResult(sqlmock.NewResult(0, int64(1-i)))
	}

	item := ProjectEventLog{
		Contract:       contract,
		EventType:      5,
		OccurredAt:     occurredAt,
		Message:        "same event",
		Payload:        `{"ok":true}`,
		IdempotencyKey: "same-key",
	}
	if err := store.AddProjectEventLog(context.Background(), item); err != nil {
		t.Fatalf("add first project event log: %v", err)
	}
	if err := store.AddProjectEventLog(context.Background(), item); err != nil {
		t.Fatalf("add duplicate project event log: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListProjectEventLogsByContractOrdersTimeline(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x0000000000000000000000000000000000000304")
	older := time.Date(2026, 5, 28, 1, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 5, 28, 2, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id",
		"contract",
		"event_type",
		"occurred_at",
		"message",
		"payload",
		"idempotency_key",
		"created_at",
	}).
		AddRow(int64(2), contract.Bytes(), int16(5), newer, "newer", []byte(`{"order":2}`), "key-2", newer).
		AddRow(int64(1), contract.Bytes(), int16(5), older, nil, []byte(`{"order":1}`), "key-1", older)

	mock.ExpectQuery("ORDER BY occurred_at DESC, id DESC").
		WithArgs(contract.Bytes()).
		WillReturnRows(rows)

	items, err := store.ListProjectEventLogsByContract(context.Background(), contract)
	if err != nil {
		t.Fatalf("list project event logs: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %d, want 2", len(items))
	}
	if items[0].ID != 2 || items[0].Payload != `{"order":2}` {
		t.Fatalf("first item = %+v, want newest item", items[0])
	}
	if items[1].ID != 1 || items[1].Message != "" {
		t.Fatalf("second item = %+v, want older item with empty message", items[1])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
