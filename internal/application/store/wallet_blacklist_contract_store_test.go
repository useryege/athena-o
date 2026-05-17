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

func TestAddWalletBlacklistContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("INSERT INTO wallet_blacklist_contract").
		WithArgs(contract.Bytes(), "seed").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.AddWalletBlacklistContract(context.Background(), WalletBlacklistContract{
		Contract: contract,
		Note:     "seed",
	})
	if err != nil {
		t.Fatalf("add wallet blacklist contract: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestAddWalletBlacklistContractDuplicateReturnsExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("INSERT INTO wallet_blacklist_contract").
		WithArgs(contract.Bytes(), nil).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	err = store.AddWalletBlacklistContract(context.Background(), WalletBlacklistContract{
		Contract: contract,
	})
	if !errors.Is(err, ErrWalletBlacklistContractAlreadyExists) {
		t.Fatalf("err = %v, want %v", err, ErrWalletBlacklistContractAlreadyExists)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListWalletBlacklistContracts(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	createdAt := time.Now().UTC().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{"contract", "note", "created_at"}).
		AddRow(contract.Bytes(), "seed", createdAt)
	mock.ExpectQuery("SELECT contract, note, created_at").WillReturnRows(rows)

	items, err := store.ListWalletBlacklistContracts(context.Background())
	if err != nil {
		t.Fatalf("list wallet blacklist contracts: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	if items[0].Contract != contract {
		t.Fatalf("contract = %s, want %s", items[0].Contract.Hex(), contract.Hex())
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

func TestUpdateWalletBlacklistContractNote(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("UPDATE wallet_blacklist_contract").
		WithArgs(contract.Bytes(), "updated").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateWalletBlacklistContractNote(context.Background(), contract, "updated"); err != nil {
		t.Fatalf("update note: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestUpdateWalletBlacklistContractNoteNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("UPDATE wallet_blacklist_contract").
		WithArgs(contract.Bytes(), nil).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.UpdateWalletBlacklistContractNote(context.Background(), contract, "")
	if !errors.Is(err, ErrWalletBlacklistContractNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrWalletBlacklistContractNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestDeleteWalletBlacklistContract(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("DELETE FROM wallet_blacklist_contract").
		WithArgs(contract.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.DeleteWalletBlacklistContract(context.Background(), contract); err != nil {
		t.Fatalf("delete wallet blacklist contract: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestDeleteWalletBlacklistContractNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("DELETE FROM wallet_blacklist_contract").
		WithArgs(contract.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.DeleteWalletBlacklistContract(context.Background(), contract)
	if !errors.Is(err, ErrWalletBlacklistContractNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrWalletBlacklistContractNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
