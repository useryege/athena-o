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

func TestAddBytecodeBlacklistContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	codeHash := common.HexToHash("0x1234")

	mock.ExpectExec("INSERT INTO bytecode_blacklist_contract").
		WithArgs(contract.Bytes(), codeHash.Bytes(), "seed").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.AddBytecodeBlacklistContract(context.Background(), BytecodeBlacklistContract{
		Contract: contract,
		CodeHash: codeHash,
		Note:     "seed",
	})
	if err != nil {
		t.Fatalf("add bytecode blacklist contract: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestAddBytecodeBlacklistContractDuplicateReturnsExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	codeHash := common.HexToHash("0x1234")

	mock.ExpectExec("INSERT INTO bytecode_blacklist_contract").
		WithArgs(contract.Bytes(), codeHash.Bytes(), nil).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	err = store.AddBytecodeBlacklistContract(context.Background(), BytecodeBlacklistContract{
		Contract: contract,
		CodeHash: codeHash,
	})
	if !errors.Is(err, ErrBytecodeBlacklistContractAlreadyExists) {
		t.Fatalf("err = %v, want %v", err, ErrBytecodeBlacklistContractAlreadyExists)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListBytecodeBlacklistContracts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	codeHash := common.HexToHash("0x1234")
	createdAt := time.Now().UTC().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{"contract", "code_hash", "note", "created_at"}).
		AddRow(contract.Bytes(), codeHash.Bytes(), "seed", createdAt)
	mock.ExpectQuery("SELECT contract, code_hash, note, created_at").WillReturnRows(rows)

	items, err := store.ListBytecodeBlacklistContracts(context.Background())
	if err != nil {
		t.Fatalf("list bytecode blacklist contracts: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	if items[0].Contract != contract {
		t.Fatalf("contract = %s, want %s", items[0].Contract.Hex(), contract.Hex())
	}
	if items[0].CodeHash != codeHash {
		t.Fatalf("code hash = %s, want %s", items[0].CodeHash.Hex(), codeHash.Hex())
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

func TestUpdateBytecodeBlacklistContractNote(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("UPDATE bytecode_blacklist_contract").
		WithArgs(contract.Bytes(), "updated").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateBytecodeBlacklistContractNote(context.Background(), contract, "updated"); err != nil {
		t.Fatalf("update note: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestUpdateBytecodeBlacklistContractNoteNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("UPDATE bytecode_blacklist_contract").
		WithArgs(contract.Bytes(), nil).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.UpdateBytecodeBlacklistContractNote(context.Background(), contract, "")
	if !errors.Is(err, ErrBytecodeBlacklistContractNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrBytecodeBlacklistContractNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestDeleteBytecodeBlacklistContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("DELETE FROM bytecode_blacklist_contract").
		WithArgs(contract.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.DeleteBytecodeBlacklistContract(context.Background(), contract); err != nil {
		t.Fatalf("delete bytecode blacklist contract: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestDeleteBytecodeBlacklistContractNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("DELETE FROM bytecode_blacklist_contract").
		WithArgs(contract.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.DeleteBytecodeBlacklistContract(context.Background(), contract)
	if !errors.Is(err, ErrBytecodeBlacklistContractNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrBytecodeBlacklistContractNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
