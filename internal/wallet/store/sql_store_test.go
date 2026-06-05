package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	walletsqlc "github.com/useryege/athena/internal/wallet/store/sqlc"
)

type fakeWalletQuerier struct {
	countWalletsErr error
	createWalletErr error
	getWalletErr    error
	updateAliasErr  error

	countWalletsParams walletsqlc.CountWalletsParams
	createWalletParams walletsqlc.CreateWalletParams
	listWalletsParams  walletsqlc.ListWalletsParams
	updateAliasParams  walletsqlc.UpdateWalletAliasParams

	countWalletsResult int64
	createWalletResult walletsqlc.WalletPrivateKey
	getWalletResult    walletsqlc.WalletPrivateKey
	listWalletsResult  []walletsqlc.ListWalletsRow
	updateAliasResult  walletsqlc.UpdateWalletAliasRow
}

func (f *fakeWalletQuerier) CountWallets(_ context.Context, arg walletsqlc.CountWalletsParams) (int64, error) {
	f.countWalletsParams = arg
	return f.countWalletsResult, f.countWalletsErr
}

func (f *fakeWalletQuerier) CreateWallet(_ context.Context, arg walletsqlc.CreateWalletParams) (walletsqlc.WalletPrivateKey, error) {
	f.createWalletParams = arg
	return f.createWalletResult, f.createWalletErr
}

func (f *fakeWalletQuerier) GetWallet(_ context.Context, _ int64) (walletsqlc.WalletPrivateKey, error) {
	return f.getWalletResult, f.getWalletErr
}

func (f *fakeWalletQuerier) ListWallets(_ context.Context, arg walletsqlc.ListWalletsParams) ([]walletsqlc.ListWalletsRow, error) {
	f.listWalletsParams = arg
	return f.listWalletsResult, nil
}

func (f *fakeWalletQuerier) UpdateWalletAlias(_ context.Context, arg walletsqlc.UpdateWalletAliasParams) (walletsqlc.UpdateWalletAliasRow, error) {
	f.updateAliasParams = arg
	return f.updateAliasResult, f.updateAliasErr
}

func TestCreateWalletUsesQuerierAndMapsUniqueViolation(t *testing.T) {
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	querier := &fakeWalletQuerier{
		createWalletResult: walletsqlc.WalletPrivateKey{
			ID:                   7,
			Chain:                "ETH",
			Address:              "0xabc",
			AddressKey:           "0xabc",
			Alias:                "main",
			PrivateKeyCiphertext: []byte("cipher"),
			MnemonicCiphertext:   []byte("mnemonic"),
			Source:               "mnemonic",
			DerivationPath:       "m/44'/60'/0'/0/0",
			CreatedAt:            pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:            pgtype.Timestamptz{Time: now, Valid: true},
		},
	}

	record, err := NewSQLStoreWithQuerier(querier).CreateWallet(context.Background(), CreateWalletRecordRequest{
		Chain:                "ETH",
		Address:              "0xabc",
		AddressKey:           "0xabc",
		Alias:                "main",
		PrivateKeyCiphertext: []byte("cipher"),
		Source:               "mnemonic",
		DerivationPath:       "m/44'/60'/0'/0/0",
	})
	if err != nil {
		t.Fatalf("CreateWallet: %v", err)
	}
	if record.ID != 7 || string(record.PrivateKeyCiphertext) != "cipher" {
		t.Fatalf("record = %#v, want generated row mapping", record)
	}
	if querier.createWalletParams.MnemonicCiphertext != nil {
		t.Fatalf("empty mnemonic ciphertext = %#v, want nil", querier.createWalletParams.MnemonicCiphertext)
	}

	querier.createWalletErr = &pgconn.PgError{Code: "23505"}
	if _, err := NewSQLStoreWithQuerier(querier).CreateWallet(context.Background(), CreateWalletRecordRequest{}); !errors.Is(err, ErrWalletAlreadyExists) {
		t.Fatalf("duplicate error = %v, want ErrWalletAlreadyExists", err)
	}
}

func TestListWalletsUsesQuerierFiltersAndPagination(t *testing.T) {
	now := time.Date(2026, time.May, 30, 12, 0, 0, 0, time.UTC)
	querier := &fakeWalletQuerier{
		countWalletsResult: 1,
		listWalletsResult: []walletsqlc.ListWalletsRow{{
			ID:             1,
			Chain:          "ETH",
			Address:        "0xabc",
			Alias:          "main",
			Source:         "mnemonic",
			DerivationPath: "m/44'/60'/0'/0/0",
			CreatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:      pgtype.Timestamptz{Time: now, Valid: true},
		}},
	}

	items, total, err := NewSQLStoreWithQuerier(querier).ListWallets(context.Background(), ListWalletsOptions{
		Chain:    " ETH ",
		Query:    " abc ",
		Page:     2,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListWallets: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Address != "0xabc" {
		t.Fatalf("items/total = %#v/%d, want one mapped wallet", items, total)
	}
	if querier.countWalletsParams.Chain.String != "ETH" || querier.countWalletsParams.Query.String != "%abc%" {
		t.Fatalf("count params = %#v, want normalized filters", querier.countWalletsParams)
	}
	if querier.listWalletsParams.Limit != 10 || querier.listWalletsParams.Offset != 10 {
		t.Fatalf("list params = %#v, want page 2 offset", querier.listWalletsParams)
	}
}

func TestWalletStoreMapsNotFound(t *testing.T) {
	querier := &fakeWalletQuerier{getWalletErr: pgx.ErrNoRows}
	if _, err := NewSQLStoreWithQuerier(querier).GetWallet(context.Background(), 404); !errors.Is(err, ErrWalletNotFound) {
		t.Fatalf("GetWallet error = %v, want ErrWalletNotFound", err)
	}
}
