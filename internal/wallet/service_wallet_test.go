package wallet

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/useryege/athena/internal/wallet/apiclient"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	utilcrypto "github.com/useryege/athena/util/crypto"
)

func TestGetWalletRevealsSecretsOnlyWhenRequested(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

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
	rows := sqlmock.NewRows([]string{
		"id", "chain", "address", "address_key", "alias", "private_key_ciphertext", "mnemonic_ciphertext", "source", "derivation_path", "created_at", "updated_at",
	}).AddRow(int64(1), "ETH", "0xabc", "0xabc", "main", privateKeyCiphertext, mnemonicCiphertext, "mnemonic", evmDerivationPath, createdAt, createdAt)
	mock.ExpectQuery("SELECT id, chain, address, address_key, alias, private_key_ciphertext, mnemonic_ciphertext, source, derivation_path, created_at, updated_at").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	service := NewService(walletstore.NewSQLStore(db), key)
	resp, err := service.GetWallet(context.Background(), &apiclient.GetWalletRequest{Id: 1, RevealSecrets: true})
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if resp.GetItem().PrivateKey != "0xabc" || resp.GetItem().Mnemonic != testMnemonic {
		t.Fatalf("secrets = %q/%q", resp.GetItem().PrivateKey, resp.GetItem().Mnemonic)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
