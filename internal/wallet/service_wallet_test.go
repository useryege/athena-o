package wallet

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/wallet/apiclient"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	walletsqlc "github.com/useryege/athena/internal/wallet/store/sqlc"
	utilcrypto "github.com/useryege/athena/util/crypto"
)

type fakeWalletQuerier struct {
	getWalletResult walletsqlc.WalletPrivateKey
}

func (f *fakeWalletQuerier) CountWallets(context.Context, walletsqlc.CountWalletsParams) (int64, error) {
	return 0, nil
}

func (f *fakeWalletQuerier) CreateWallet(context.Context, walletsqlc.CreateWalletParams) (walletsqlc.WalletPrivateKey, error) {
	return walletsqlc.WalletPrivateKey{}, nil
}

func (f *fakeWalletQuerier) GetWallet(context.Context, int64) (walletsqlc.WalletPrivateKey, error) {
	return f.getWalletResult, nil
}

func (f *fakeWalletQuerier) ListWallets(context.Context, walletsqlc.ListWalletsParams) ([]walletsqlc.ListWalletsRow, error) {
	return nil, nil
}

func (f *fakeWalletQuerier) UpdateWalletAlias(context.Context, walletsqlc.UpdateWalletAliasParams) (walletsqlc.UpdateWalletAliasRow, error) {
	return walletsqlc.UpdateWalletAliasRow{}, nil
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
