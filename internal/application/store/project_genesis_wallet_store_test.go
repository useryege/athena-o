package store

import (
	"context"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestReplaceProjectGenesisWallets(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewSQLStore(db)
	projectContract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	txHash := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	walletA := common.HexToAddress("0x2222222222222222222222222222222222222222")
	walletB := common.HexToAddress("0x3333333333333333333333333333333333333333")

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM project_genesis_wallet").
		WithArgs(projectContract.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectExec("INSERT INTO project_genesis_wallet").
		WithArgs(projectContract.Bytes(), walletA.Bytes(), "100", int64(5000), int32(0), "200", txHash.Bytes(), int64(1000)).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO project_genesis_wallet").
		WithArgs(projectContract.Bytes(), walletB.Bytes(), "50", int64(2500), int32(1), "200", txHash.Bytes(), int64(1000)).
		WillReturnResult(sqlmock.NewResult(2, 1))
	mock.ExpectExec("INSERT INTO project_component_state").
		WithArgs(projectContract.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err = store.ReplaceProjectGenesisWallets(context.Background(), projectContract, []ProjectGenesisWallet{
		{
			ProjectContract:   projectContract,
			Wallet:            walletA,
			NetAmount:         mustBigInt("100"),
			RatioBPS:          5000,
			RankIndex:         0,
			TotalSupply:       mustBigInt("200"),
			SourceTxHash:      txHash,
			SourceBlockNumber: 1000,
		},
		{
			ProjectContract:   projectContract,
			Wallet:            walletB,
			NetAmount:         mustBigInt("50"),
			RatioBPS:          2500,
			RankIndex:         1,
			TotalSupply:       mustBigInt("200"),
			SourceTxHash:      txHash,
			SourceBlockNumber: 1000,
		},
	})
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestReplaceProjectGenesisWalletsRejectsNonPositiveNetAmount(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewSQLStore(db)
	projectContract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	txHash := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	wallet := common.HexToAddress("0x2222222222222222222222222222222222222222")

	mock.ExpectBegin()
	mock.ExpectExec("DELETE FROM project_genesis_wallet").
		WithArgs(projectContract.Bytes()).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectRollback()

	err = store.ReplaceProjectGenesisWallets(context.Background(), projectContract, []ProjectGenesisWallet{
		{
			ProjectContract:   projectContract,
			Wallet:            wallet,
			NetAmount:         mustBigInt("0"),
			RatioBPS:          0,
			RankIndex:         0,
			TotalSupply:       mustBigInt("100"),
			SourceTxHash:      txHash,
			SourceBlockNumber: 1000,
		},
	})
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "net amount must be positive"))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestListProjectGenesisWalletsByContractAndWallet(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	store := NewSQLStore(db)
	projectContract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	walletA := common.HexToAddress("0x2222222222222222222222222222222222222222")
	walletB := common.HexToAddress("0x3333333333333333333333333333333333333333")
	txHash := common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	now := time.Now().UTC()

	rowsByContract := sqlmock.NewRows([]string{
		"id", "project_contract", "wallet", "net_amount", "ratio_bps", "rank_index", "total_supply", "source_tx_hash", "source_block_number", "created_at",
	}).
		AddRow(int64(1), projectContract.Bytes(), walletA.Bytes(), "100", int64(5000), int32(0), "200", txHash.Bytes(), int64(1000), now).
		AddRow(int64(2), projectContract.Bytes(), walletB.Bytes(), "50", int64(2500), int32(1), "200", txHash.Bytes(), int64(1000), now)
	mock.ExpectQuery("SELECT(.|\\n)*FROM project_genesis_wallet(.|\\n)*WHERE project_contract = \\$1").
		WithArgs(projectContract.Bytes()).
		WillReturnRows(rowsByContract)

	items, err := store.ListProjectGenesisWalletsByContract(context.Background(), projectContract)
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, walletA, items[0].Wallet)
	require.Equal(t, "100", items[0].NetAmount.String())
	require.EqualValues(t, 0, items[0].RankIndex)

	rowsByWallet := sqlmock.NewRows([]string{
		"id", "project_contract", "wallet", "net_amount", "ratio_bps", "rank_index", "total_supply", "source_tx_hash", "source_block_number", "created_at",
	}).
		AddRow(int64(3), projectContract.Bytes(), walletA.Bytes(), "100", int64(5000), int32(0), "200", txHash.Bytes(), int64(1000), now)
	mock.ExpectQuery("SELECT(.|\\n)*FROM project_genesis_wallet(.|\\n)*WHERE wallet = \\$1").
		WithArgs(walletA.Bytes()).
		WillReturnRows(rowsByWallet)

	byWallet, err := store.ListProjectGenesisWalletsByWallet(context.Background(), walletA)
	require.NoError(t, err)
	require.Len(t, byWallet, 1)
	require.Equal(t, projectContract, byWallet[0].ProjectContract)
	require.Equal(t, int64(5000), byWallet[0].RatioBPS)
	require.NoError(t, mock.ExpectationsWereMet())
}

func mustBigInt(value string) *big.Int {
	result, ok := new(big.Int).SetString(value, 10)
	if !ok {
		panic("invalid big int literal: " + value)
	}
	return result
}
