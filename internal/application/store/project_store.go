package store

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/common"
)

func (s *SQLStore) SaveProjectMeta(ctx context.Context, meta ProjectMeta) error {
	txHash := meta.TxHash
	if meta.Tx != nil {
		txHash = meta.Tx.Hash()
	}
	if txHash == (common.Hash{}) {
		return errors.New("project meta transaction is nil")
	}
	if meta.BlockNumber > math.MaxInt64 {
		return fmt.Errorf("project meta block number %d exceeds postgres BIGINT", meta.BlockNumber)
	}
	if meta.BlockTime > math.MaxInt64 {
		return fmt.Errorf("project meta block time %d exceeds postgres BIGINT", meta.BlockTime)
	}
	if meta.TxIndex > math.MaxInt64 {
		return fmt.Errorf("project meta tx index %d exceeds postgres BIGINT", meta.TxIndex)
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO project (
  project_id,
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT DO NOTHING
`, meta.ProjectID, int64(meta.BlockNumber), int64(meta.BlockTime), meta.Contract.Bytes(), meta.Creator.Bytes(), txHash.Bytes(), int64(meta.TxIndex))
	if err != nil {
		return fmt.Errorf("save project meta: %w", err)
	}
	return nil
}

func (s *SQLStore) ListProjectMetas(ctx context.Context) ([]ProjectMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  project_id,
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index
FROM project
ORDER BY block_number, tx_index, id
`)
	if err != nil {
		return nil, fmt.Errorf("list project metas: %w", err)
	}
	defer rows.Close()

	metas := make([]ProjectMeta, 0)
	for rows.Next() {
		var meta ProjectMeta
		var blockNumber int64
		var blockTime int64
		var contract []byte
		var creator []byte
		var txHash []byte
		var txIndex int64
		if err := rows.Scan(&meta.ProjectID, &blockNumber, &blockTime, &contract, &creator, &txHash, &txIndex); err != nil {
			return nil, fmt.Errorf("scan project meta: %w", err)
		}
		if blockNumber < 0 {
			return nil, fmt.Errorf("project meta block number %d is negative", blockNumber)
		}
		if blockTime < 0 {
			return nil, fmt.Errorf("project meta block time %d is negative", blockTime)
		}
		if txIndex < 0 {
			return nil, fmt.Errorf("project meta tx index %d is negative", txIndex)
		}
		meta.BlockNumber = uint64(blockNumber)
		meta.BlockTime = uint64(blockTime)
		meta.Contract = common.BytesToAddress(contract)
		meta.Creator = common.BytesToAddress(creator)
		meta.TxHash = common.BytesToHash(txHash)
		meta.TxIndex = uint64(txIndex)
		metas = append(metas, meta)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project metas: %w", err)
	}
	return metas, nil
}
