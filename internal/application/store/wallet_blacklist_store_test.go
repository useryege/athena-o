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

func TestAddWalletBlacklistEntry(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("INSERT INTO wallet_blacklist").
		WithArgs(wallet.Bytes(), "seed").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store.AddWalletBlacklistEntry(context.Background(), WalletBlacklistEntry{
		Wallet: wallet,
		Note:   "seed",
	})
	if err != nil {
		t.Fatalf("add wallet blacklist entry: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestAddWalletBlacklistEntryDuplicateReturnsExists(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("INSERT INTO wallet_blacklist").
		WithArgs(wallet.Bytes(), nil).
		WillReturnError(&pgconn.PgError{Code: "23505"})

	err = store.AddWalletBlacklistEntry(context.Background(), WalletBlacklistEntry{
		Wallet: wallet,
	})
	if !errors.Is(err, ErrWalletBlacklistEntryAlreadyExists) {
		t.Fatalf("err = %v, want %v", err, ErrWalletBlacklistEntryAlreadyExists)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListWalletBlacklistEntries(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	createdAt := time.Now().UTC().Truncate(time.Microsecond)

	rows := sqlmock.NewRows([]string{"wallet", "note", "created_at"}).
		AddRow(wallet.Bytes(), "seed", createdAt)
	mock.ExpectQuery("SELECT wallet, note, created_at").WillReturnRows(rows)

	items, err := store.ListWalletBlacklistEntries(context.Background())
	if err != nil {
		t.Fatalf("list wallet blacklist entries: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items len = %d, want 1", len(items))
	}
	if items[0].Wallet != wallet {
		t.Fatalf("wallet = %s, want %s", items[0].Wallet.Hex(), wallet.Hex())
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

func TestUpdateWalletBlacklistEntryNote(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("UPDATE wallet_blacklist").
		WithArgs(wallet.Bytes(), "updated").
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.UpdateWalletBlacklistEntryNote(context.Background(), wallet, "updated"); err != nil {
		t.Fatalf("update note: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestUpdateWalletBlacklistEntryNoteNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("UPDATE wallet_blacklist").
		WithArgs(wallet.Bytes(), nil).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.UpdateWalletBlacklistEntryNote(context.Background(), wallet, "")
	if !errors.Is(err, ErrWalletBlacklistEntryNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrWalletBlacklistEntryNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestDeleteWalletBlacklistEntry(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("DELETE FROM wallet_blacklist").
		WithArgs(wallet.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	if err := store.DeleteWalletBlacklistEntry(context.Background(), wallet); err != nil {
		t.Fatalf("delete wallet blacklist entry: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestDeleteWalletBlacklistEntryNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")

	mock.ExpectExec("DELETE FROM wallet_blacklist").
		WithArgs(wallet.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = store.DeleteWalletBlacklistEntry(context.Background(), wallet)
	if !errors.Is(err, ErrWalletBlacklistEntryNotFound) {
		t.Fatalf("err = %v, want %v", err, ErrWalletBlacklistEntryNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
