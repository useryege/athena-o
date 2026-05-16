package store

import (
	"context"
	"database/sql"
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
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index
) VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT DO NOTHING
`, int64(meta.BlockNumber), int64(meta.BlockTime), meta.Contract.Bytes(), meta.Creator.Bytes(), txHash.Bytes(), int64(meta.TxIndex))
	if err != nil {
		return fmt.Errorf("save project meta: %w", err)
	}
	return nil
}

func (s *SQLStore) ListProjectMetas(ctx context.Context) ([]ProjectMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index,
  source_code
FROM project
WHERE is_archived = FALSE
ORDER BY block_number, tx_index, id
`)
	if err != nil {
		return nil, fmt.Errorf("list project metas: %w", err)
	}
	defer rows.Close()

	metas := make([]ProjectMeta, 0)
	for rows.Next() {
		meta, err := scanProjectMetaRow(rows, false)
		if err != nil {
			return nil, err
		}
		metas = append(metas, meta)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project metas: %w", err)
	}
	return metas, nil
}

func (s *SQLStore) ListAllProjectMetas(ctx context.Context) ([]ProjectMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index,
  source_code,
  is_archived,
  archived_at
FROM project
ORDER BY block_number, tx_index, id
`)
	if err != nil {
		return nil, fmt.Errorf("list all project metas: %w", err)
	}
	defer rows.Close()

	metas := make([]ProjectMeta, 0)
	for rows.Next() {
		meta, err := scanProjectMetaRow(rows, true)
		if err != nil {
			return nil, err
		}
		metas = append(metas, meta)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate all project metas: %w", err)
	}
	return metas, nil
}

func (s *SQLStore) UpdateProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE project
SET source_code = $2
WHERE contract = $1
`, contract.Bytes(), sourceCode)
	if err != nil {
		return fmt.Errorf("update project source code: %w", err)
	}
	return nil
}

func (s *SQLStore) ArchiveProjectByContract(ctx context.Context, contract common.Address) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE project
SET is_archived = TRUE, archived_at = now()
WHERE contract = $1
`, contract.Bytes())
	if err != nil {
		return fmt.Errorf("archive project: %w", err)
	}
	return nil
}

func (s *SQLStore) UnarchiveProjectByContract(ctx context.Context, contract common.Address) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE project
SET is_archived = FALSE, archived_at = NULL
WHERE contract = $1
`, contract.Bytes())
	if err != nil {
		return fmt.Errorf("unarchive project: %w", err)
	}
	return nil
}

func (s *SQLStore) ListArchivedProjectMetas(ctx context.Context, page int32, pageSize int32) ([]ProjectMeta, int64, int32, int32, error) {
	page, pageSize = normalizePage(page, pageSize)

	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM project WHERE is_archived = TRUE`).Scan(&total); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("count archived project metas: %w", err)
	}

	offset := int64(page-1) * int64(pageSize)
	rows, err := s.db.QueryContext(ctx, `
SELECT
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index,
  source_code,
  is_archived,
  archived_at
FROM project
WHERE is_archived = TRUE
ORDER BY archived_at DESC, id DESC
LIMIT $1 OFFSET $2
`, pageSize, offset)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("list archived project metas: %w", err)
	}
	defer rows.Close()

	metas := make([]ProjectMeta, 0)
	for rows.Next() {
		meta, err := scanProjectMetaRow(rows, true)
		if err != nil {
			return nil, 0, 0, 0, err
		}
		metas = append(metas, meta)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("iterate archived project metas: %w", err)
	}
	return metas, total, page, pageSize, nil
}

func (s *SQLStore) GetArchivedProjectMetaByContract(ctx context.Context, contract common.Address) (*ProjectMeta, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index,
  source_code,
  is_archived,
  archived_at
FROM project
WHERE contract = $1 AND is_archived = TRUE
`, contract.Bytes())
	meta, err := scanProjectMetaRow(row, true)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &meta, nil
}

func (s *SQLStore) GetProjectMetaByContract(ctx context.Context, contract common.Address) (*ProjectMeta, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
  block_number,
  block_time,
  contract,
  creator,
  tx_hash,
  tx_index,
  source_code,
  is_archived,
  archived_at
FROM project
WHERE contract = $1
`, contract.Bytes())
	meta, err := scanProjectMetaRow(row, true)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &meta, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanProjectMetaRow(scanner rowScanner, withArchiveFields bool) (ProjectMeta, error) {
	var meta ProjectMeta
	var blockNumber int64
	var blockTime int64
	var contract []byte
	var creator []byte
	var txHash []byte
	var txIndex int64
	var sourceCode sql.NullString
	var isArchived bool
	var archivedAt sql.NullTime

	var err error
	if withArchiveFields {
		err = scanner.Scan(&blockNumber, &blockTime, &contract, &creator, &txHash, &txIndex, &sourceCode, &isArchived, &archivedAt)
	} else {
		err = scanner.Scan(&blockNumber, &blockTime, &contract, &creator, &txHash, &txIndex, &sourceCode)
	}
	if err != nil {
		return ProjectMeta{}, fmt.Errorf("scan project meta: %w", err)
	}
	if blockNumber < 0 {
		return ProjectMeta{}, fmt.Errorf("project meta block number %d is negative", blockNumber)
	}
	if blockTime < 0 {
		return ProjectMeta{}, fmt.Errorf("project meta block time %d is negative", blockTime)
	}
	if txIndex < 0 {
		return ProjectMeta{}, fmt.Errorf("project meta tx index %d is negative", txIndex)
	}

	meta.BlockNumber = uint64(blockNumber)
	meta.BlockTime = uint64(blockTime)
	meta.Contract = common.BytesToAddress(contract)
	meta.Creator = common.BytesToAddress(creator)
	meta.TxHash = common.BytesToHash(txHash)
	meta.TxIndex = uint64(txIndex)
	meta.SourceCode = sourceCode.String
	meta.IsArchived = isArchived
	if archivedAt.Valid {
		meta.ArchivedAt = archivedAt.Time
	}
	return meta, nil
}

func normalizePage(page int32, pageSize int32) (int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}
