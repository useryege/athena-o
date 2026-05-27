package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestCreateWallet(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	createdAt := time.Now().UTC().Truncate(time.Second)
	rows := sqlmock.NewRows([]string{
		"id", "chain", "address", "address_key", "alias", "private_key_ciphertext", "mnemonic_ciphertext", "source", "derivation_path", "created_at", "updated_at",
	}).AddRow(int64(1), "ETH", "0xabc", "0xabc", "main", []byte("secret"), nil, "private_key", "", createdAt, createdAt)
	mock.ExpectQuery("INSERT INTO wallet_private_keys").
		WithArgs("ETH", "0xabc", "0xabc", "main", []byte("secret"), nil, "private_key", "").
		WillReturnRows(rows)

	item, err := store.CreateWallet(context.Background(), CreateWalletRecordRequest{
		Chain:                "ETH",
		Address:              "0xabc",
		AddressKey:           "0xabc",
		Alias:                "main",
		PrivateKeyCiphertext: []byte("secret"),
		Source:               "private_key",
	})
	if err != nil {
		t.Fatalf("create wallet: %v", err)
	}
	if item.ID != 1 || item.Address != "0xabc" || item.Alias != "main" {
		t.Fatalf("item = %#v", item)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestCreateWalletDuplicate(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	mock.ExpectQuery("INSERT INTO wallet_private_keys").
		WillReturnError(&pgconn.PgError{Code: "23505"})

	_, err = store.CreateWallet(context.Background(), CreateWalletRecordRequest{})
	if !errors.Is(err, ErrWalletAlreadyExists) {
		t.Fatalf("err = %v, want %v", err, ErrWalletAlreadyExists)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}

func TestListWallets(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	store := NewSQLStore(db)
	createdAt := time.Date(2026, 5, 28, 1, 2, 3, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM wallet_private_keys").
		WithArgs("ETH", "%main%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(int64(1)))
	rows := sqlmock.NewRows([]string{"id", "chain", "address", "alias", "source", "derivation_path", "created_at", "updated_at"}).
		AddRow(int64(1), "ETH", "0xabc", "main", "private_key", "", createdAt, updatedAt)
	mock.ExpectQuery("SELECT id, chain, address, alias, source, derivation_path, created_at, updated_at").
		WithArgs("ETH", "%main%", 20, 0).
		WillReturnRows(rows)

	items, total, err := store.ListWallets(context.Background(), ListWalletsOptions{Chain: "ETH", Query: "main", Page: 1, PageSize: 20})
	if err != nil {
		t.Fatalf("list wallets: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("total/items = %d/%d, want 1/1", total, len(items))
	}
	if items[0].CreatedAt != "2026-05-28T01:02:03Z" || items[0].UpdatedAt != "2026-05-28T01:03:03Z" {
		t.Fatalf("timestamps = %q/%q", items[0].CreatedAt, items[0].UpdatedAt)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
