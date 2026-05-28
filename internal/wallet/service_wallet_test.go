package wallet

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/useryege/athena/internal/wallet/apiclient"
	walletstore "github.com/useryege/athena/internal/wallet/store"
	utilcrypto "github.com/useryege/athena/util/crypto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func TestWalletBlacklistServiceValidationAndErrors(t *testing.T) {
	service := NewService(walletstore.NewSQLStore(nil), testWalletEncryptionKey(t))
	if _, err := service.AddWalletBlacklistEntry(context.Background(), &apiclient.AddWalletBlacklistEntryRequest{Wallet: "bad"}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid wallet error = %v, want InvalidArgument", err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new: %v", err)
	}
	defer db.Close()

	service = NewService(walletstore.NewSQLStore(db), testWalletEncryptionKey(t))
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	mock.ExpectExec("INSERT INTO wallet_blacklist").
		WithArgs(wallet.Bytes(), "seed").
		WillReturnError(&pgconn.PgError{Code: "23505"})
	if _, err := service.AddWalletBlacklistEntry(context.Background(), &apiclient.AddWalletBlacklistEntryRequest{Wallet: wallet.Hex(), Note: "seed"}); status.Code(err) != codes.AlreadyExists {
		t.Fatalf("duplicate wallet error = %v, want AlreadyExists", err)
	}

	mock.ExpectExec("UPDATE wallet_blacklist").
		WithArgs(wallet.Bytes(), nil).
		WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := service.UpdateWalletBlacklistEntryNote(context.Background(), &apiclient.UpdateWalletBlacklistEntryNoteRequest{Wallet: wallet.Hex()}); status.Code(err) != codes.NotFound {
		t.Fatalf("update missing wallet error = %v, want NotFound", err)
	}

	mock.ExpectExec("DELETE FROM wallet_blacklist").
		WithArgs(wallet.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	if _, err := service.DeleteWalletBlacklistEntry(context.Background(), &apiclient.DeleteWalletBlacklistEntryRequest{Wallet: wallet.Hex()}); status.Code(err) != codes.NotFound {
		t.Fatalf("delete missing wallet error = %v, want NotFound", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations were not met: %v", err)
	}
}
