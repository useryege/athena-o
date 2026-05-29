package wallet

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/wallet/apiclient"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	walletsqlc "github.com/useryege/athena/internal/wallet/store/sqlc"
	utilcrypto "github.com/useryege/athena/util/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeWalletQuerier struct {
	getWalletResult      walletsqlc.WalletPrivateKey
	addBlacklistErr      error
	updateRowsAffected   int64
	deleteRowsAffected   int64
	addBlacklistParams   walletsqlc.AddWalletBlacklistEntryParams
	updateBlacklistParam walletsqlc.UpdateWalletBlacklistEntryNoteParams
	deleteBlacklistParam []byte
}

func (f *fakeWalletQuerier) AddWalletBlacklistEntry(_ context.Context, arg walletsqlc.AddWalletBlacklistEntryParams) error {
	f.addBlacklistParams = arg
	return f.addBlacklistErr
}

func (f *fakeWalletQuerier) CountWallets(context.Context, walletsqlc.CountWalletsParams) (int64, error) {
	return 0, nil
}

func (f *fakeWalletQuerier) CreateWallet(context.Context, walletsqlc.CreateWalletParams) (walletsqlc.WalletPrivateKey, error) {
	return walletsqlc.WalletPrivateKey{}, nil
}

func (f *fakeWalletQuerier) DeleteWalletBlacklistEntry(_ context.Context, wallet []byte) (int64, error) {
	f.deleteBlacklistParam = wallet
	return f.deleteRowsAffected, nil
}

func (f *fakeWalletQuerier) GetWallet(context.Context, int64) (walletsqlc.WalletPrivateKey, error) {
	return f.getWalletResult, nil
}

func (f *fakeWalletQuerier) ListWalletBlacklistEntries(context.Context) ([]walletsqlc.WalletBlacklist, error) {
	return nil, nil
}

func (f *fakeWalletQuerier) ListWallets(context.Context, walletsqlc.ListWalletsParams) ([]walletsqlc.ListWalletsRow, error) {
	return nil, nil
}

func (f *fakeWalletQuerier) UpdateWalletAlias(context.Context, walletsqlc.UpdateWalletAliasParams) (walletsqlc.UpdateWalletAliasRow, error) {
	return walletsqlc.UpdateWalletAliasRow{}, nil
}

func (f *fakeWalletQuerier) UpdateWalletBlacklistEntryNote(_ context.Context, arg walletsqlc.UpdateWalletBlacklistEntryNoteParams) (int64, error) {
	f.updateBlacklistParam = arg
	return f.updateRowsAffected, nil
}

func TestGetWalletRevealsSecretsOnlyWhenRequested(t *testing.T) {
	key := testWalletEncryptionKey(t)
	privateKeyCiphertext, err := utilcrypto.Encrypt([]byte("0xabc"), key)
	if err != nil {
		t.Fatalf("encrypt private key: %v", err)
	}
	mnemonicCiphertext, err := utilcrypto.Encrypt([]byte(testMnemonic), key)
	if err != nil {
		t.Fatalf("encrypt mnemonic: %v", err)
	}
	createdAt := time.Now().UTC().Truncate(time.Second)
	store := walletstore.NewSQLStoreWithQuerier(&fakeWalletQuerier{
		getWalletResult: walletsqlc.WalletPrivateKey{
			ID:                   1,
			Chain:                "ETH",
			Address:              "0xabc",
			AddressKey:           "0xabc",
			Alias:                "main",
			PrivateKeyCiphertext: privateKeyCiphertext,
			MnemonicCiphertext:   mnemonicCiphertext,
			Source:               "mnemonic",
			DerivationPath:       evmDerivationPath,
			CreatedAt:            pgtype.Timestamptz{Time: createdAt, Valid: true},
			UpdatedAt:            pgtype.Timestamptz{Time: createdAt, Valid: true},
		},
	})

	resp, err := NewService(store, key).GetWallet(context.Background(), &apiclient.GetWalletRequest{Id: 1, RevealSecrets: true})
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if resp.GetItem().PrivateKey != "0xabc" || resp.GetItem().Mnemonic != testMnemonic {
		t.Fatalf("secrets = %q/%q", resp.GetItem().PrivateKey, resp.GetItem().Mnemonic)
	}
}

func TestWalletBlacklistServiceValidationAndErrors(t *testing.T) {
	service := NewService(walletstore.NewSQLStore(nil), testWalletEncryptionKey(t))
	if _, err := service.AddWalletBlacklistEntry(context.Background(), &apiclient.AddWalletBlacklistEntryRequest{Wallet: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid wallet error = %v, want InvalidArgument", err)
	}

	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	querier := &fakeWalletQuerier{
		addBlacklistErr:    &pgconn.PgError{Code: "23505"},
		updateRowsAffected: 0,
		deleteRowsAffected: 0,
	}
	service = NewService(walletstore.NewSQLStoreWithQuerier(querier), testWalletEncryptionKey(t))
	if _, err := service.AddWalletBlacklistEntry(context.Background(), &apiclient.AddWalletBlacklistEntryRequest{Wallet: wallet.Hex(), Note: "seed"}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate wallet error = %v, want AlreadyExists", err)
	}
	if _, err := service.UpdateWalletBlacklistEntryNote(context.Background(), &apiclient.UpdateWalletBlacklistEntryNoteRequest{Wallet: wallet.Hex()}); status.Code(err) != codes.NotFound {
		t.Fatalf("update missing wallet error = %v, want NotFound", err)
	}
	if _, err := service.DeleteWalletBlacklistEntry(context.Background(), &apiclient.DeleteWalletBlacklistEntryRequest{Wallet: wallet.Hex()}); status.Code(err) != codes.NotFound {
		t.Fatalf("delete missing wallet error = %v, want NotFound", err)
	}
}
