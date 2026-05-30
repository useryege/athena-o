package store

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

func (s *SQLStore) ReplaceProjectGenesisWallets(ctx context.Context, contract common.Address, items []ProjectGenesisWallet) error {
	if s.pool == nil {
		return fmt.Errorf("application postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace project genesis wallets tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := appsqlc.New(tx)
	if err := queries.DeleteProjectGenesisWalletsByContract(ctx, contract.Bytes()); err != nil {
		return fmt.Errorf("delete project genesis wallets: %w", err)
	}

	for _, item := range items {
		if item.NetAmount == nil || item.NetAmount.Sign() <= 0 {
			return fmt.Errorf("project genesis wallet %s net amount must be positive", item.Wallet.Hex())
		}
		if item.RatioBPS < 0 {
			return fmt.Errorf("project genesis wallet %s ratio bps must be non-negative", item.Wallet.Hex())
		}
		if item.RankIndex < 0 {
			return fmt.Errorf("project genesis wallet %s rank index must be non-negative", item.Wallet.Hex())
		}
		if item.TotalSupply == nil || item.TotalSupply.Sign() < 0 {
			return fmt.Errorf("project genesis wallet %s total supply must be non-negative", item.Wallet.Hex())
		}
		if item.SourceBlockNumber > math.MaxInt64 {
			return fmt.Errorf("project genesis wallet %s source block number %d exceeds postgres BIGINT", item.Wallet.Hex(), item.SourceBlockNumber)
		}

		err = queries.InsertProjectGenesisWallet(ctx, appsqlc.InsertProjectGenesisWalletParams{
			ProjectContract:   contract.Bytes(),
			Wallet:            item.Wallet.Bytes(),
			Column3:           numericFromBigInt(item.NetAmount),
			RatioBps:          item.RatioBPS,
			RankIndex:         item.RankIndex,
			Column6:           numericFromBigInt(item.TotalSupply),
			SourceTxHash:      item.SourceTxHash.Bytes(),
			SourceBlockNumber: int64(item.SourceBlockNumber),
		})
		if err != nil {
			return fmt.Errorf("insert project genesis wallet %s: %w", item.Wallet.Hex(), err)
		}
	}

	if err := queries.MarkProjectComponentSuccessNow(ctx, appsqlc.MarkProjectComponentSuccessNowParams{
		ProjectContract: contract.Bytes(),
		Component:       ProjectComponentGenesisWallet,
	}); err != nil {
		return fmt.Errorf("update project genesis wallet component state: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replace project genesis wallets tx: %w", err)
	}
	return nil
}

func (s *SQLStore) ListProjectGenesisWalletsByContract(ctx context.Context, contract common.Address) ([]ProjectGenesisWallet, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectGenesisWalletsByContract(ctx, contract.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project genesis wallets by contract: %w", err)
	}

	items := make([]ProjectGenesisWallet, 0, len(rows))
	for _, row := range rows {
		item, err := projectGenesisWalletFromFields(row.ID, row.ProjectContract, row.Wallet, row.NetAmount, row.RatioBps, row.RankIndex, row.TotalSupply, row.SourceTxHash, row.SourceBlockNumber, row.CreatedAt.Time)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) ListProjectGenesisWalletsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address][]ProjectGenesisWallet, error) {
	result := make(map[common.Address][]ProjectGenesisWallet)
	uniqueContracts := uniqueNonZeroAddresses(contracts)
	if len(uniqueContracts) == 0 {
		return result, nil
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectGenesisWalletsByContracts(ctx, addressesToBytes(uniqueContracts))
	if err != nil {
		return nil, fmt.Errorf("list project genesis wallets by contracts: %w", err)
	}

	for _, row := range rows {
		item, err := projectGenesisWalletFromFields(row.ID, row.ProjectContract, row.Wallet, row.NetAmount, row.RatioBps, row.RankIndex, row.TotalSupply, row.SourceTxHash, row.SourceBlockNumber, row.CreatedAt.Time)
		if err != nil {
			return nil, err
		}
		result[item.ProjectContract] = append(result[item.ProjectContract], item)
	}
	return result, nil
}

func (s *SQLStore) ListProjectGenesisWalletsByWallet(ctx context.Context, wallet common.Address) ([]ProjectGenesisWallet, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListProjectGenesisWalletsByWallet(ctx, wallet.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project genesis wallets by wallet: %w", err)
	}

	items := make([]ProjectGenesisWallet, 0, len(rows))
	for _, row := range rows {
		item, err := projectGenesisWalletFromFields(row.ID, row.ProjectContract, row.Wallet, row.NetAmount, row.RatioBps, row.RankIndex, row.TotalSupply, row.SourceTxHash, row.SourceBlockNumber, row.CreatedAt.Time)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func projectGenesisWalletFromFields(id int64, projectContract []byte, wallet []byte, netAmountText string, ratioBps int64, rankIndex int32, totalSupplyText string, sourceTxHash []byte, sourceBlockNumber int64, createdAt time.Time) (ProjectGenesisWallet, error) {
	if sourceBlockNumber < 0 {
		return ProjectGenesisWallet{}, fmt.Errorf("project genesis wallet source block number %d is negative", sourceBlockNumber)
	}

	netAmount, ok := new(big.Int).SetString(netAmountText, 10)
	if !ok {
		return ProjectGenesisWallet{}, fmt.Errorf("parse project genesis wallet net amount %q", netAmountText)
	}
	totalSupply, ok := new(big.Int).SetString(totalSupplyText, 10)
	if !ok {
		return ProjectGenesisWallet{}, fmt.Errorf("parse project genesis wallet total supply %q", totalSupplyText)
	}

	return ProjectGenesisWallet{
		ID:                id,
		ProjectContract:   common.BytesToAddress(projectContract),
		Wallet:            common.BytesToAddress(wallet),
		NetAmount:         netAmount,
		RatioBPS:          ratioBps,
		RankIndex:         rankIndex,
		TotalSupply:       totalSupply,
		SourceTxHash:      common.BytesToHash(sourceTxHash),
		SourceBlockNumber: uint64(sourceBlockNumber),
		CreatedAt:         createdAt,
	}, nil
}

func numericFromBigInt(value *big.Int) pgtype.Numeric {
	if value == nil {
		return pgtype.Numeric{}
	}
	return pgtype.Numeric{Int: new(big.Int).Set(value), Exp: 0, Valid: true}
}
