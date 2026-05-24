package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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
	if meta.SourceCode != "" && meta.SourceCodeHash == (common.Hash{}) {
		meta.SourceCodeHash = crypto.Keccak256Hash([]byte(meta.SourceCode))
	}

	_, err := s.db.ExecContext(ctx, `
INSERT INTO project (
  block_number,
  block_time,
  contract,
  creator,
  weth_pair,
  usdt_pair,
  fetch_at,
  tx_hash,
  tx_index,
  source_code,
  source_code_hash,
  source_code_fetched_at,
  code_bin_hash,
  code_bin_hash_fetched_at,
  source_quality_report,
  source_quality_report_fetched_at,
  creator_result_can_mint_from_dead_via_transfer_from,
  creator_result_can_mint_from_zero_via_transfer_from,
  creator_result_can_mint_from_weth_pair_via_transfer_from,
  creator_result_can_mint_from_usdt_pair_via_transfer_from,
  creator_result_can_mint_via_transfer_to_weth_pair,
  creator_result_can_mint_via_transfer_to_usdt_pair,
  report_is_policy_evaluated,
  report_is_blacklisted_creator_wallet,
  report_is_blacklisted_genesis_wallet,
  report_is_blacklisted_bytecode,
  report_is_blacklisted_source_code,
  report_has_mint_risk,
  genesis_wallets_fetched_at,
  creator_historical_projects_fetched_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26, $27, $28, $29, $30)
ON CONFLICT DO NOTHING
`, int64(meta.BlockNumber), int64(meta.BlockTime), meta.Contract.Bytes(), meta.Creator.Bytes(), meta.WethPair.Bytes(), meta.UsdtPair.Bytes(), meta.FetchAt, txHash.Bytes(), int64(meta.TxIndex), nullableText(meta.SourceCode), nullableHashBytes(meta.SourceCodeHash), nullableTime(meta.SourceCodeFetchedAt), nullableHashBytes(meta.CodeBinHash), nullableTime(meta.CodeBinHashFetchedAt), nullableText(meta.SourceQualityReport), nullableTime(meta.SourceQualityReportFetchedAt), meta.CreatorResult.CanMintFromDeadViaTransferFrom, meta.CreatorResult.CanMintFromZeroViaTransferFrom, meta.CreatorResult.CanMintFromWethPairViaTransferFrom, meta.CreatorResult.CanMintFromUsdtPairViaTransferFrom, meta.CreatorResult.CanMintViaTransferToWethPair, meta.CreatorResult.CanMintViaTransferToUsdtPair, meta.Report.IsPolicyEvaluated, meta.Report.IsBlacklistedCreatorWallet, meta.Report.IsBlacklistedGenesisWallet, meta.Report.IsBlacklistedBytecode, meta.Report.IsBlacklistedSourceCode, meta.Report.HasMintRisk, nullableTime(meta.GenesisWalletsFetchedAt), nullableTime(meta.CreatorHistoricalProjectsFetchedAt))
	if err != nil {
		return fmt.Errorf("save project meta: %w", err)
	}
	return nil
}

func (s *SQLStore) GetMaxProjectBlockNumber(ctx context.Context) (uint64, bool, error) {
	var maxBlock sql.NullInt64
	if err := s.db.QueryRowContext(ctx, `
SELECT MAX(block_number)
FROM project
`).Scan(&maxBlock); err != nil {
		return 0, false, fmt.Errorf("get max project block number: %w", err)
	}
	if !maxBlock.Valid {
		return 0, false, nil
	}
	if maxBlock.Int64 < 0 {
		return 0, false, fmt.Errorf("project meta block number %d is negative", maxBlock.Int64)
	}
	return uint64(maxBlock.Int64), true, nil
}

func (s *SQLStore) ListProjectMetas(ctx context.Context) ([]ProjectMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  block_number,
  block_time,
  contract,
  creator,
  weth_pair,
  usdt_pair,
  fetch_at,
  tx_hash,
  tx_index,
  source_code,
  source_code_hash,
  source_code_fetched_at,
  code_bin_hash,
  code_bin_hash_fetched_at,
  source_quality_report,
  source_quality_report_fetched_at,
  creator_result_can_mint_from_dead_via_transfer_from,
  creator_result_can_mint_from_zero_via_transfer_from,
  creator_result_can_mint_from_weth_pair_via_transfer_from,
  creator_result_can_mint_from_usdt_pair_via_transfer_from,
  creator_result_can_mint_via_transfer_to_weth_pair,
  creator_result_can_mint_via_transfer_to_usdt_pair,
  report_is_policy_evaluated,
  report_is_blacklisted_creator_wallet,
  report_is_blacklisted_genesis_wallet,
  report_is_blacklisted_bytecode,
  report_is_blacklisted_source_code,
  report_has_mint_risk,
  genesis_wallets_fetched_at,
  creator_historical_projects_fetched_at
FROM project
ORDER BY block_number, tx_index, id
`)
	if err != nil {
		return nil, fmt.Errorf("list project metas: %w", err)
	}
	defer rows.Close()

	metas := make([]ProjectMeta, 0)
	for rows.Next() {
		meta, err := scanProjectMetaRow(rows)
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
	return s.ListProjectMetas(ctx)
}

func (s *SQLStore) ListProjectMetasByPairAddresses(ctx context.Context, pairs []common.Address) ([]ProjectMeta, error) {
	uniquePairs := uniqueNonZeroAddresses(pairs)
	if len(uniquePairs) == 0 {
		return nil, nil
	}

	placeholders := make([]string, 0, len(uniquePairs))
	args := make([]any, 0, len(uniquePairs))
	for i, pair := range uniquePairs {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, pair.Bytes())
	}
	inClause := strings.Join(placeholders, ", ")
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT
  block_number,
  block_time,
  contract,
  creator,
  weth_pair,
  usdt_pair,
  fetch_at,
  tx_hash,
  tx_index,
  source_code,
  source_code_hash,
  source_code_fetched_at,
  code_bin_hash,
  code_bin_hash_fetched_at,
  source_quality_report,
  source_quality_report_fetched_at,
  creator_result_can_mint_from_dead_via_transfer_from,
  creator_result_can_mint_from_zero_via_transfer_from,
  creator_result_can_mint_from_weth_pair_via_transfer_from,
  creator_result_can_mint_from_usdt_pair_via_transfer_from,
  creator_result_can_mint_via_transfer_to_weth_pair,
  creator_result_can_mint_via_transfer_to_usdt_pair,
  report_is_policy_evaluated,
  report_is_blacklisted_creator_wallet,
  report_is_blacklisted_genesis_wallet,
  report_is_blacklisted_bytecode,
  report_is_blacklisted_source_code,
  report_has_mint_risk,
  genesis_wallets_fetched_at,
  creator_historical_projects_fetched_at
FROM project
WHERE weth_pair IN (%s) OR usdt_pair IN (%s)
ORDER BY block_number, tx_index, id
`, inClause, inClause), args...)
	if err != nil {
		return nil, fmt.Errorf("list project metas by pair addresses: %w", err)
	}
	defer rows.Close()

	metas := make([]ProjectMeta, 0)
	for rows.Next() {
		meta, err := scanProjectMetaRow(rows)
		if err != nil {
			return nil, err
		}
		metas = append(metas, meta)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project metas by pair addresses: %w", err)
	}
	return metas, nil
}

func (s *SQLStore) ListProjectMetasByCreator(ctx context.Context, creator common.Address) ([]ProjectMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
  block_number,
  block_time,
  contract,
  creator,
  weth_pair,
  usdt_pair,
  fetch_at,
  tx_hash,
  tx_index,
  source_code,
  source_code_hash,
  source_code_fetched_at,
  code_bin_hash,
  code_bin_hash_fetched_at,
  source_quality_report,
  source_quality_report_fetched_at,
  creator_result_can_mint_from_dead_via_transfer_from,
  creator_result_can_mint_from_zero_via_transfer_from,
  creator_result_can_mint_from_weth_pair_via_transfer_from,
  creator_result_can_mint_from_usdt_pair_via_transfer_from,
  creator_result_can_mint_via_transfer_to_weth_pair,
  creator_result_can_mint_via_transfer_to_usdt_pair,
  report_is_policy_evaluated,
  report_is_blacklisted_creator_wallet,
  report_is_blacklisted_genesis_wallet,
  report_is_blacklisted_bytecode,
  report_is_blacklisted_source_code,
  report_has_mint_risk,
  genesis_wallets_fetched_at,
  creator_historical_projects_fetched_at
FROM project
WHERE creator = $1
ORDER BY block_number, tx_index, id
`, creator.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project metas by creator: %w", err)
	}
	defer rows.Close()

	metas := make([]ProjectMeta, 0)
	for rows.Next() {
		meta, err := scanProjectMetaRow(rows)
		if err != nil {
			return nil, err
		}
		metas = append(metas, meta)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project metas by creator: %w", err)
	}
	return metas, nil
}

func uniqueNonZeroAddresses(items []common.Address) []common.Address {
	seen := make(map[common.Address]struct{}, len(items))
	unique := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item == (common.Address{}) {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		unique = append(unique, item)
	}
	return unique
}

func (s *SQLStore) ListProjectMetasByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectMeta, error) {
	if blockNumber > math.MaxInt64 {
		return nil, fmt.Errorf("project meta block number %d exceeds postgres BIGINT", blockNumber)
	}
	if txIndex > math.MaxInt64 {
		return nil, fmt.Errorf("project meta tx index %d exceeds postgres BIGINT", txIndex)
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT
  block_number,
  block_time,
  contract,
  creator,
  weth_pair,
  usdt_pair,
  fetch_at,
  tx_hash,
  tx_index,
  source_code,
  source_code_hash,
  source_code_fetched_at,
  code_bin_hash,
  code_bin_hash_fetched_at,
  source_quality_report,
  source_quality_report_fetched_at,
  creator_result_can_mint_from_dead_via_transfer_from,
  creator_result_can_mint_from_zero_via_transfer_from,
  creator_result_can_mint_from_weth_pair_via_transfer_from,
  creator_result_can_mint_from_usdt_pair_via_transfer_from,
  creator_result_can_mint_via_transfer_to_weth_pair,
  creator_result_can_mint_via_transfer_to_usdt_pair,
  report_is_policy_evaluated,
  report_is_blacklisted_creator_wallet,
  report_is_blacklisted_genesis_wallet,
  report_is_blacklisted_bytecode,
  report_is_blacklisted_source_code,
  report_has_mint_risk,
  genesis_wallets_fetched_at,
  creator_historical_projects_fetched_at
FROM project
WHERE creator = $1
  AND (block_number < $2 OR (block_number = $2 AND tx_index < $3))
ORDER BY block_number, tx_index, id
`, creator.Bytes(), int64(blockNumber), int64(txIndex))
	if err != nil {
		return nil, fmt.Errorf("list project metas by creator before: %w", err)
	}
	defer rows.Close()

	metas := make([]ProjectMeta, 0)
	for rows.Next() {
		meta, err := scanProjectMetaRow(rows)
		if err != nil {
			return nil, err
		}
		metas = append(metas, meta)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project metas by creator before: %w", err)
	}
	return metas, nil
}

func (s *SQLStore) UpdateProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error {
	sourceCodeHash := common.Hash{}
	if sourceCode != "" {
		sourceCodeHash = crypto.Keccak256Hash([]byte(sourceCode))
	}
	_, err := s.db.ExecContext(ctx, `
UPDATE project
SET source_code = $2, source_code_hash = $3, source_code_fetched_at = now()
WHERE contract = $1
`, contract.Bytes(), sourceCode, nullableHashBytes(sourceCodeHash))
	if err != nil {
		return fmt.Errorf("update project source code: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateProjectCodeBinHash(ctx context.Context, contract common.Address, codeBinHash common.Hash) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE project
SET code_bin_hash = $2, code_bin_hash_fetched_at = now()
WHERE contract = $1
`, contract.Bytes(), nullableHashBytes(codeBinHash))
	if err != nil {
		return fmt.Errorf("update project code bin hash: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateProjectSourceQualityReport(ctx context.Context, contract common.Address, report string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE project
SET source_quality_report = $2, source_quality_report_fetched_at = now()
WHERE contract = $1
`, contract.Bytes(), report)
	if err != nil {
		return fmt.Errorf("update project source quality report: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateProjectCreatorResult(ctx context.Context, contract common.Address, result SimulateResult) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE project
SET creator_result_can_mint_from_dead_via_transfer_from = $2,
  creator_result_can_mint_from_zero_via_transfer_from = $3,
  creator_result_can_mint_from_weth_pair_via_transfer_from = $4,
  creator_result_can_mint_from_usdt_pair_via_transfer_from = $5,
  creator_result_can_mint_via_transfer_to_weth_pair = $6,
  creator_result_can_mint_via_transfer_to_usdt_pair = $7
WHERE contract = $1
`, contract.Bytes(), result.CanMintFromDeadViaTransferFrom, result.CanMintFromZeroViaTransferFrom, result.CanMintFromWethPairViaTransferFrom, result.CanMintFromUsdtPairViaTransferFrom, result.CanMintViaTransferToWethPair, result.CanMintViaTransferToUsdtPair)
	if err != nil {
		return fmt.Errorf("update project creator result: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateProjectReport(ctx context.Context, contract common.Address, report ProjectReport) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE project
SET report_is_policy_evaluated = $2,
  report_is_blacklisted_creator_wallet = $3,
  report_is_blacklisted_genesis_wallet = $4,
  report_is_blacklisted_bytecode = $5,
  report_is_blacklisted_source_code = $6,
  report_has_mint_risk = $7
WHERE contract = $1
`, contract.Bytes(), report.IsPolicyEvaluated, report.IsBlacklistedCreatorWallet, report.IsBlacklistedGenesisWallet, report.IsBlacklistedBytecode, report.IsBlacklistedSourceCode, report.HasMintRisk)
	if err != nil {
		return fmt.Errorf("update project report: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectMetaByContract(ctx context.Context, contract common.Address) (*ProjectMeta, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
  block_number,
  block_time,
  contract,
  creator,
  weth_pair,
  usdt_pair,
  fetch_at,
  tx_hash,
  tx_index,
  source_code,
  source_code_hash,
  source_code_fetched_at,
  code_bin_hash,
  code_bin_hash_fetched_at,
  source_quality_report,
  source_quality_report_fetched_at,
  creator_result_can_mint_from_dead_via_transfer_from,
  creator_result_can_mint_from_zero_via_transfer_from,
  creator_result_can_mint_from_weth_pair_via_transfer_from,
  creator_result_can_mint_from_usdt_pair_via_transfer_from,
  creator_result_can_mint_via_transfer_to_weth_pair,
  creator_result_can_mint_via_transfer_to_usdt_pair,
  report_is_policy_evaluated,
  report_is_blacklisted_creator_wallet,
  report_is_blacklisted_genesis_wallet,
  report_is_blacklisted_bytecode,
  report_is_blacklisted_source_code,
  report_has_mint_risk,
  genesis_wallets_fetched_at,
  creator_historical_projects_fetched_at
FROM project
WHERE contract = $1
`, contract.Bytes())
	meta, err := scanProjectMetaRow(row)
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

func scanProjectMetaRow(scanner rowScanner) (ProjectMeta, error) {
	var meta ProjectMeta
	var blockNumber int64
	var blockTime int64
	var contract []byte
	var creator []byte
	var wethPair []byte
	var usdtPair []byte
	var fetchAt time.Time
	var txHash []byte
	var txIndex int64
	var sourceCode sql.NullString
	var sourceCodeHash []byte
	var sourceCodeFetchedAt sql.NullTime
	var codeBinHash []byte
	var codeBinHashFetchedAt sql.NullTime
	var sourceQualityReport sql.NullString
	var sourceQualityReportFetchedAt sql.NullTime
	var genesisWalletsFetchedAt sql.NullTime
	var creatorHistoricalProjectsFetchedAt sql.NullTime

	if err := scanner.Scan(&blockNumber, &blockTime, &contract, &creator, &wethPair, &usdtPair, &fetchAt, &txHash, &txIndex, &sourceCode, &sourceCodeHash, &sourceCodeFetchedAt, &codeBinHash, &codeBinHashFetchedAt, &sourceQualityReport, &sourceQualityReportFetchedAt, &meta.CreatorResult.CanMintFromDeadViaTransferFrom, &meta.CreatorResult.CanMintFromZeroViaTransferFrom, &meta.CreatorResult.CanMintFromWethPairViaTransferFrom, &meta.CreatorResult.CanMintFromUsdtPairViaTransferFrom, &meta.CreatorResult.CanMintViaTransferToWethPair, &meta.CreatorResult.CanMintViaTransferToUsdtPair, &meta.Report.IsPolicyEvaluated, &meta.Report.IsBlacklistedCreatorWallet, &meta.Report.IsBlacklistedGenesisWallet, &meta.Report.IsBlacklistedBytecode, &meta.Report.IsBlacklistedSourceCode, &meta.Report.HasMintRisk, &genesisWalletsFetchedAt, &creatorHistoricalProjectsFetchedAt); err != nil {
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
	meta.WethPair = common.BytesToAddress(wethPair)
	meta.UsdtPair = common.BytesToAddress(usdtPair)
	meta.FetchAt = fetchAt
	meta.TxHash = common.BytesToHash(txHash)
	meta.TxIndex = uint64(txIndex)
	meta.SourceCode = sourceCode.String
	meta.SourceCodeHash = common.BytesToHash(sourceCodeHash)
	if sourceCodeFetchedAt.Valid {
		meta.SourceCodeFetchedAt = sourceCodeFetchedAt.Time
	}
	meta.CodeBinHash = common.BytesToHash(codeBinHash)
	if codeBinHashFetchedAt.Valid {
		meta.CodeBinHashFetchedAt = codeBinHashFetchedAt.Time
	}
	meta.SourceQualityReport = sourceQualityReport.String
	if sourceQualityReportFetchedAt.Valid {
		meta.SourceQualityReportFetchedAt = sourceQualityReportFetchedAt.Time
	}
	if genesisWalletsFetchedAt.Valid {
		meta.GenesisWalletsFetchedAt = genesisWalletsFetchedAt.Time
	}
	if creatorHistoricalProjectsFetchedAt.Valid {
		meta.CreatorHistoricalProjectsFetchedAt = creatorHistoricalProjectsFetchedAt.Time
	}
	return meta, nil
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullableHashBytes(value common.Hash) any {
	if value == (common.Hash{}) {
		return nil
	}
	return value.Bytes()
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value.UTC()
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
