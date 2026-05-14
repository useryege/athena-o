package store

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
)

func TestUpdateProjectSourceCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	projectID := uuid.New()
	sourceCode := "contract C {}"

	mock.ExpectExec("UPDATE project").
		WithArgs(projectID, sourceCode).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateProjectSourceCode(context.Background(), projectID, sourceCode); err != nil {
		t.Fatalf("update project source code: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListProjectMetasIncludesSourceCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	projectID := uuid.New()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000A1")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000B2")
	txHash := common.HexToHash("0x1234")
	sourceCode := "contract X {}"

	rows := sqlmock.NewRows([]string{
		"project_id", "block_number", "block_time", "contract", "creator", "tx_hash", "tx_index", "source_code",
	}).AddRow(projectID, int64(100), int64(200), contract.Bytes(), creator.Bytes(), txHash.Bytes(), int64(3), sourceCode)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	metas, err := store.ListProjectMetas(context.Background())
	if err != nil {
		t.Fatalf("list project metas: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("metas len = %d, want 1", len(metas))
	}
	if metas[0].SourceCode != sourceCode {
		t.Fatalf("source code = %q, want %q", metas[0].SourceCode, sourceCode)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListProjectMetasNullSourceCodeReturnsEmptyString(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	projectID := uuid.New()
	contract := common.HexToAddress("0x00000000000000000000000000000000000000C3")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000D4")
	txHash := common.HexToHash("0x5678")

	rows := sqlmock.NewRows([]string{
		"project_id", "block_number", "block_time", "contract", "creator", "tx_hash", "tx_index", "source_code",
	}).AddRow(projectID, int64(101), int64(201), contract.Bytes(), creator.Bytes(), txHash.Bytes(), int64(4), nil)

	mock.ExpectQuery("SELECT").WillReturnRows(rows)

	metas, err := store.ListProjectMetas(context.Background())
	if err != nil {
		t.Fatalf("list project metas: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("metas len = %d, want 1", len(metas))
	}
	if metas[0].SourceCode != "" {
		t.Fatalf("source code = %q, want empty", metas[0].SourceCode)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
