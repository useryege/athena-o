package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestAddSourcecodeBlacklistContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	sourceHash := common.HexToHash("0x1234")

	mock.ExpectExec("INSERT INTO sourcecode_blacklist_contract").
		WithArgs(contract.Bytes(), sourceHash.Bytes(), "seed").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.AddSourcecodeBlacklistContract(context.Background(), SourcecodeBlacklistContract{
		Contract:   contract,
		SourceHash: sourceHash,
		Note:       "seed",
	})
	if err != nil {
		t.Fatalf("add sourcecode blacklist contract: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestAddSourcecodeBlacklistContractDuplicateReturnsExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	sourceHash := common.HexToHash("0x1234")

	mock.ExpectExec("INSERT INTO sourcecode_blacklist_contract").
		WithArgs(contract.Bytes(), sourceHash.Bytes(), nil).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	err = store.AddSourcecodeBlacklistContract(context.Background(), SourcecodeBlacklistContract{
		Contract:   contract,
		SourceHash: sourceHash,
	})
	if !errors.Is(err, ErrSourcecodeBlacklistContractAlreadyExists) {
		t.Fatalf("err = %v, want %v", err, ErrSourcecodeBlacklistContractAlreadyExists)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListSourcecodeBlacklistContracts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	sourceHash := common.HexToHash("0x1234")
	createdAt := time.Now().UTC().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{"contract", "source_hash", "note", "created_at"}).
		AddRow(contract.Bytes(), sourceHash.Bytes(), "seed", createdAt)
	mock.ExpectQuery("SELECT contract, source_hash, note, created_at").WillReturnRows(rows)

	items, err := store.ListSourcecodeBlacklistContracts(context.Background())
	if err != nil {
		t.Fatalf("list sourcecode blacklist contracts: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	if items[0].Contract != contract {
		t.Fatalf("contract = %s, want %s", items[0].Contract.Hex(), contract.Hex())
	}
	if items[0].SourceHash != sourceHash {
		t.Fatalf("source hash = %s, want %s", items[0].SourceHash.Hex(), sourceHash.Hex())
	}
	if items[0].Note != "seed" {
		t.Fatalf("note = %q, want %q", items[0].Note, "seed")
	}
	if !items[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %v, want %v", items[0].CreatedAt, createdAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestUpdateSourcecodeBlacklistContractNoteNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("UPDATE sourcecode_blacklist_contract").
		WithArgs(contract.Bytes(), nil).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.UpdateSourcecodeBlacklistContractNote(context.Background(), contract, "")
	if !errors.Is(err, ErrSourcecodeBlacklistContractNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrSourcecodeBlacklistContractNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestDeleteSourcecodeBlacklistContractNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("DELETE FROM sourcecode_blacklist_contract").
		WithArgs(contract.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.DeleteSourcecodeBlacklistContract(context.Background(), contract)
	if !errors.Is(err, ErrSourcecodeBlacklistContractNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrSourcecodeBlacklistContractNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
