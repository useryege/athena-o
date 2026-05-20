package store

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

func (s *SQLStore) ReplaceProjectGenesisWallets(ctx context.Context, contract common.Address, items []ProjectGenesisWallet) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace project genesis wallets tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, `
DELETE FROM project_genesis_wallet
WHERE project_contract = $1
`, contract.Bytes()); err != nil {
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

		if _, err := tx.ExecContext(ctx, `
INSERT INTO project_genesis_wallet (
  project_contract,
  wallet,
  net_amount,
  ratio_bps,
  rank_index,
  total_supply,
  source_tx_hash,
  source_block_number
) VALUES ($1, $2, $3::numeric, $4, $5, $6::numeric, $7, $8)
`, contract.Bytes(), item.Wallet.Bytes(), item.NetAmount.String(), item.RatioBPS, item.RankIndex, item.TotalSupply.String(), item.SourceTxHash.Bytes(), int64(item.SourceBlockNumber)); err != nil {
			return fmt.Errorf("insert project genesis wallet %s: %w", item.Wallet.Hex(), err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace project genesis wallets tx: %w", err)
	}
	return nil
}

func (s *SQLStore) ListProjectGenesisWalletsByContract(ctx context.Context, contract common.Address) ([]ProjectGenesisWallet, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  id,
  project_contract,
  wallet,
  net_amount::text,
  ratio_bps,
  rank_index,
  total_supply::text,
  source_tx_hash,
  source_block_number,
  created_at
FROM project_genesis_wallet
WHERE project_contract = $1
ORDER BY rank_index ASC, id ASC
`, contract.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project genesis wallets by contract: %w", err)
	}
	defer rows.Close()

	items := make([]ProjectGenesisWallet, 0)
	for rows.Next() {
		item, err := scanProjectGenesisWalletRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project genesis wallets by contract: %w", err)
	}
	return items, nil
}

func (s *SQLStore) ListProjectGenesisWalletsByWallet(ctx context.Context, wallet common.Address) ([]ProjectGenesisWallet, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  id,
  project_contract,
  wallet,
  net_amount::text,
  ratio_bps,
  rank_index,
  total_supply::text,
  source_tx_hash,
  source_block_number,
  created_at
FROM project_genesis_wallet
WHERE wallet = $1
ORDER BY ratio_bps DESC, project_contract ASC, id ASC
`, wallet.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project genesis wallets by wallet: %w", err)
	}
	defer rows.Close()

	items := make([]ProjectGenesisWallet, 0)
	for rows.Next() {
		item, err := scanProjectGenesisWalletRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project genesis wallets by wallet: %w", err)
	}
	return items, nil
}

func scanProjectGenesisWalletRow(scanner rowScanner) (ProjectGenesisWallet, error) {
	var (
		item              ProjectGenesisWallet
		projectContract   []byte
		wallet            []byte
		netAmountText     string
		totalSupplyText   string
		sourceTxHash      []byte
		sourceBlockNumber int64
	)
	if err := scanner.Scan(
		&item.ID,
		&projectContract,
		&wallet,
		&netAmountText,
		&item.RatioBPS,
		&item.RankIndex,
		&totalSupplyText,
		&sourceTxHash,
		&sourceBlockNumber,
		&item.CreatedAt,
	); err != nil {
		return ProjectGenesisWallet{}, fmt.Errorf("scan project genesis wallet: %w", err)
	}
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

	item.ProjectContract = common.BytesToAddress(projectContract)
	item.Wallet = common.BytesToAddress(wallet)
	item.NetAmount = netAmount
	item.TotalSupply = totalSupply
	item.SourceTxHash = common.BytesToHash(sourceTxHash)
	item.SourceBlockNumber = uint64(sourceBlockNumber)
	return item, nil
}
