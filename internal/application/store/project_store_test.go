package store

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
)

func TestUpdateProjectSourceCode(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x0000000000000000000000000000000000000001")
	sourceCode := "contract C {}"

	mock.ExpectExec("UPDATE project").
		WithArgs(contract.Bytes(), sourceCode).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateProjectSourceCode(context.Background(), contract, sourceCode); err != nil {
		t.Fatalf("update project source code: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestUpdateProjectSourceQualityReport(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x0000000000000000000000000000000000000001")
	report := "## Quality Report"

	mock.ExpectExec("UPDATE project").
		WithArgs(contract.Bytes(), report).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateProjectSourceQualityReport(context.Background(), contract, report); err != nil {
		t.Fatalf("update project source quality report: %v", err)
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
	contract := common.HexToAddress("0x00000000000000000000000000000000000000A1")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000B2")
	txHash := common.HexToHash("0x1234")
	sourceCode := "contract X {}"
	report := "## Report"
	reportedAt := time.Date(2026, 5, 22, 8, 0, 0, 0, time.UTC)

	rows := sqlmock.NewRows([]string{
		"block_number", "block_time", "contract", "creator", "tx_hash", "tx_index", "source_code", "source_quality_report", "source_quality_reported_at",
	}).AddRow(int64(100), int64(200), contract.Bytes(), creator.Bytes(), txHash.Bytes(), int64(3), sourceCode, report, reportedAt)

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
	if metas[0].SourceQualityReport != report {
		t.Fatalf("source quality report = %q, want %q", metas[0].SourceQualityReport, report)
	}
	if !metas[0].SourceQualityReportedAt.Equal(reportedAt) {
		t.Fatalf("source quality reported at = %s, want %s", metas[0].SourceQualityReportedAt, reportedAt)
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
	contract := common.HexToAddress("0x00000000000000000000000000000000000000C3")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000D4")
	txHash := common.HexToHash("0x5678")

	rows := sqlmock.NewRows([]string{
		"block_number", "block_time", "contract", "creator", "tx_hash", "tx_index", "source_code", "source_quality_report", "source_quality_reported_at",
	}).AddRow(int64(101), int64(201), contract.Bytes(), creator.Bytes(), txHash.Bytes(), int64(4), nil, nil, nil)

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
	if metas[0].SourceQualityReport != "" {
		t.Fatalf("source quality report = %q, want empty", metas[0].SourceQualityReport)
	}
	if !metas[0].SourceQualityReportedAt.IsZero() {
		t.Fatalf("source quality reported at = %s, want zero", metas[0].SourceQualityReportedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListProjectMetasByCreator(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	creator := common.HexToAddress("0x00000000000000000000000000000000000000B2")
	contractA := common.HexToAddress("0x00000000000000000000000000000000000000A1")
	contractB := common.HexToAddress("0x00000000000000000000000000000000000000A2")
	txHashA := common.HexToHash("0x1234")
	txHashB := common.HexToHash("0x5678")

	rows := sqlmock.NewRows([]string{
		"block_number", "block_time", "contract", "creator", "tx_hash", "tx_index", "source_code", "source_quality_report", "source_quality_reported_at", "is_archived", "archived_at",
	}).AddRow(int64(100), int64(200), contractA.Bytes(), creator.Bytes(), txHashA.Bytes(), int64(1), "contract A {}", "", nil, false, nil).
		AddRow(int64(101), int64(201), contractB.Bytes(), creator.Bytes(), txHashB.Bytes(), int64(2), "contract B {}", "report", time.Now(), true, nil)

	mock.ExpectQuery("SELECT").
		WithArgs(creator.Bytes()).
		WillReturnRows(rows)

	metas, err := store.ListProjectMetasByCreator(context.Background(), creator)
	if err != nil {
		t.Fatalf("list project metas by creator: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("metas len = %d, want 2", len(metas))
	}
	if metas[0].Contract != contractA || metas[1].Contract != contractB {
		t.Fatalf("contracts = [%s, %s], want [%s, %s]", metas[0].Contract.Hex(), metas[1].Contract.Hex(), contractA.Hex(), contractB.Hex())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
