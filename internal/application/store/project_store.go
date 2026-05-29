package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
)

func (s *SQLStore) SaveProjectBase(ctx context.Context, base ProjectBase) error {
	txHash := base.TxHash
	if base.Tx != nil {
		txHash = base.Tx.Hash()
	}
	if txHash == (common.Hash{}) {
		return errors.New("project base transaction is nil")
	}
	if err := validateProjectBaseNumbers(base.BlockNumber, base.BlockTime, base.TxIndex); err != nil {
		return err
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
`, int64(base.BlockNumber), int64(base.BlockTime), base.Contract.Bytes(), base.Creator.Bytes(), txHash.Bytes(), int64(base.TxIndex))
	if err != nil {
		return fmt.Errorf("save project base: %w", err)
	}
	return nil
}

func (s *SQLStore) SaveProjectMeta(ctx context.Context, meta ProjectMeta) error {
	return s.SaveProjectBase(ctx, ProjectBase{
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract,
		Creator:     meta.Creator,
		Tx:          meta.Tx,
		TxHash:      meta.TxHash,
		TxIndex:     meta.TxIndex,
	})
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
		return 0, false, fmt.Errorf("project base block number %d is negative", maxBlock.Int64)
	}
	return uint64(maxBlock.Int64), true, nil
}

func (s *SQLStore) ListProjectBases(ctx context.Context) ([]ProjectBase, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
ORDER BY block_number, tx_index, id
`)
	if err != nil {
		return nil, fmt.Errorf("list project bases: %w", err)
	}
	defer rows.Close()
	return scanProjectBases(rows)
}

func (s *SQLStore) ListProjectBasesPage(ctx context.Context, page int32, pageSize int32) ([]ProjectBase, int64, int32, int32, error) {
	page, pageSize = normalizePage(page, pageSize)
	var total int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*)::bigint FROM project`).Scan(&total); err != nil {
		return nil, 0, page, pageSize, fmt.Errorf("count project bases: %w", err)
	}
	offset := int64(page-1) * int64(pageSize)
	rows, err := s.db.QueryContext(ctx, `
SELECT block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
ORDER BY block_number, tx_index, id
LIMIT $1 OFFSET $2
`, int64(pageSize), offset)
	if err != nil {
		return nil, 0, page, pageSize, fmt.Errorf("list project bases page: %w", err)
	}
	defer rows.Close()
	items, err := scanProjectBases(rows)
	return items, total, page, pageSize, err
}

func (s *SQLStore) GetProjectBaseByContract(ctx context.Context, contract common.Address) (*ProjectBase, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE contract = $1
`, contract.Bytes())
	base, err := scanProjectBaseRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &base, nil
}

func (s *SQLStore) ListProjectBasesByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectBase, error) {
	if err := validateProjectOrderNumbers(blockNumber, txIndex); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT block_number, block_time, contract, creator, tx_hash, tx_index, created_at
FROM project
WHERE creator = $1
  AND (block_number < $2 OR (block_number = $2 AND tx_index < $3))
ORDER BY block_number, tx_index, id
`, creator.Bytes(), int64(blockNumber), int64(txIndex))
	if err != nil {
		return nil, fmt.Errorf("list project bases by creator before: %w", err)
	}
	defer rows.Close()
	return scanProjectBases(rows)
}

func (s *SQLStore) ListProjectMetas(ctx context.Context) ([]ProjectMeta, error) {
	rows, err := s.db.QueryContext(ctx, projectMetaSelectSQL+`
ORDER BY p.block_number, p.tx_index, p.id
`)
	if err != nil {
		return nil, fmt.Errorf("list project metas: %w", err)
	}
	defer rows.Close()
	return scanProjectMetas(rows)
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
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(projectMetaSelectSQL+`
WHERE cs.weth_pair IN (%s) OR cs.usdt_pair IN (%s)
ORDER BY p.block_number, p.tx_index, p.id
`, inClause, inClause), args...)
	if err != nil {
		return nil, fmt.Errorf("list project metas by pair addresses: %w", err)
	}
	defer rows.Close()
	return scanProjectMetas(rows)
}

func (s *SQLStore) ListProjectMetasByCreator(ctx context.Context, creator common.Address) ([]ProjectMeta, error) {
	rows, err := s.db.QueryContext(ctx, projectMetaSelectSQL+`
WHERE p.creator = $1
ORDER BY p.block_number, p.tx_index, p.id
`, creator.Bytes())
	if err != nil {
		return nil, fmt.Errorf("list project metas by creator: %w", err)
	}
	defer rows.Close()
	return scanProjectMetas(rows)
}

func (s *SQLStore) ListProjectMetasByCreatorBefore(ctx context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]ProjectMeta, error) {
	if err := validateProjectOrderNumbers(blockNumber, txIndex); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, projectMetaSelectSQL+`
WHERE p.creator = $1
  AND (p.block_number < $2 OR (p.block_number = $2 AND p.tx_index < $3))
ORDER BY p.block_number, p.tx_index, p.id
`, creator.Bytes(), int64(blockNumber), int64(txIndex))
	if err != nil {
		return nil, fmt.Errorf("list project metas by creator before: %w", err)
	}
	defer rows.Close()
	return scanProjectMetas(rows)
}

func (s *SQLStore) GetProjectMetaByContract(ctx context.Context, contract common.Address) (*ProjectMeta, error) {
	row := s.db.QueryRowContext(ctx, projectMetaSelectSQL+`
WHERE p.contract = $1
`, contract.Bytes())
	meta, err := scanProjectMetaRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &meta, nil
}

func (s *SQLStore) UpsertProjectChainState(ctx context.Context, item ProjectChainState) error {
	payload := item.RawChainState
	if len(payload) == 0 {
		data, err := json.Marshal(item.ChainState)
		if err != nil {
			return fmt.Errorf("marshal project chain state: %w", err)
		}
		payload = data
	}
	if item.WethPair == (common.Address{}) {
		item.WethPair = item.ChainState.WethPair.ContractAddress
	}
	if item.UsdtPair == (common.Address{}) {
		item.UsdtPair = item.ChainState.UsdtPair.ContractAddress
	}
	if item.TokenName == "" {
		item.TokenName = item.ChainState.Token.Name
	}
	if item.TokenSymbol == "" {
		item.TokenSymbol = item.ChainState.Token.Symbol
	}
	fetchedAt := item.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO project_chain_state (
  project_contract,
  chain_state,
  weth_pair,
  usdt_pair,
  token_name,
  token_symbol,
  fetched_at
) VALUES ($1, $2::jsonb, $3, $4, $5, $6, $7)
ON CONFLICT (project_contract) DO UPDATE
SET chain_state = EXCLUDED.chain_state,
  weth_pair = EXCLUDED.weth_pair,
  usdt_pair = EXCLUDED.usdt_pair,
  token_name = EXCLUDED.token_name,
  token_symbol = EXCLUDED.token_symbol,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now()
`, item.ProjectContract.Bytes(), string(payload), item.WethPair.Bytes(), item.UsdtPair.Bytes(), nullableText(item.TokenName), nullableText(item.TokenSymbol), fetchedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert project chain state: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectChainState(ctx context.Context, contract common.Address) (*ProjectChainState, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT project_contract, chain_state, weth_pair, usdt_pair, token_name, token_symbol, fetched_at, updated_at
FROM project_chain_state
WHERE project_contract = $1
`, contract.Bytes())
	item, err := scanProjectChainStateRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) ListProjectChainStatesByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]ProjectChainState, error) {
	unique := uniqueNonZeroAddresses(contracts)
	result := make(map[common.Address]ProjectChainState, len(unique))
	if len(unique) == 0 {
		return result, nil
	}
	clause, args := addressInClause(unique)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT project_contract, chain_state, weth_pair, usdt_pair, token_name, token_symbol, fetched_at, updated_at
FROM project_chain_state
WHERE project_contract IN (%s)
`, clause), args...)
	if err != nil {
		return nil, fmt.Errorf("list project chain states by contracts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanProjectChainStateRow(rows)
		if err != nil {
			return nil, err
		}
		result[item.ProjectContract] = item
	}
	return result, rows.Err()
}

func (s *SQLStore) ListProjectChainStatesByPairAddresses(ctx context.Context, pairs []common.Address) ([]ProjectChainState, error) {
	unique := uniqueNonZeroAddresses(pairs)
	if len(unique) == 0 {
		return nil, nil
	}
	clause, args := addressInClause(unique)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT project_contract, chain_state, weth_pair, usdt_pair, token_name, token_symbol, fetched_at, updated_at
FROM project_chain_state
WHERE weth_pair IN (%s) OR usdt_pair IN (%s)
`, clause, clause), args...)
	if err != nil {
		return nil, fmt.Errorf("list project chain states by pair addresses: %w", err)
	}
	defer rows.Close()
	items := make([]ProjectChainState, 0)
	for rows.Next() {
		item, err := scanProjectChainStateRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *SQLStore) UpsertProjectSimulationResult(ctx context.Context, item ProjectSimulationResult) error {
	fetchedAt := item.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	r := item.Result
	_, err := s.db.ExecContext(ctx, `
INSERT INTO project_simulation_result (
  project_contract,
  can_mint_from_dead_via_transfer_from,
  can_mint_from_zero_via_transfer_from,
  can_mint_from_weth_pair_via_transfer_from,
  can_mint_from_usdt_pair_via_transfer_from,
  can_mint_via_transfer_to_weth_pair,
  can_mint_via_transfer_to_usdt_pair,
  fetched_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (project_contract) DO UPDATE
SET can_mint_from_dead_via_transfer_from = EXCLUDED.can_mint_from_dead_via_transfer_from,
  can_mint_from_zero_via_transfer_from = EXCLUDED.can_mint_from_zero_via_transfer_from,
  can_mint_from_weth_pair_via_transfer_from = EXCLUDED.can_mint_from_weth_pair_via_transfer_from,
  can_mint_from_usdt_pair_via_transfer_from = EXCLUDED.can_mint_from_usdt_pair_via_transfer_from,
  can_mint_via_transfer_to_weth_pair = EXCLUDED.can_mint_via_transfer_to_weth_pair,
  can_mint_via_transfer_to_usdt_pair = EXCLUDED.can_mint_via_transfer_to_usdt_pair,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now()
`, item.ProjectContract.Bytes(), r.CanMintFromDeadViaTransferFrom, r.CanMintFromZeroViaTransferFrom, r.CanMintFromWethPairViaTransferFrom, r.CanMintFromUsdtPairViaTransferFrom, r.CanMintViaTransferToWethPair, r.CanMintViaTransferToUsdtPair, fetchedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert project simulation result: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateProjectCreatorResult(ctx context.Context, contract common.Address, result SimulateResult) error {
	return s.UpsertProjectSimulationResult(ctx, ProjectSimulationResult{ProjectContract: contract, Result: result})
}

func (s *SQLStore) GetProjectSimulationResult(ctx context.Context, contract common.Address) (*ProjectSimulationResult, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT project_contract,
  can_mint_from_dead_via_transfer_from,
  can_mint_from_zero_via_transfer_from,
  can_mint_from_weth_pair_via_transfer_from,
  can_mint_from_usdt_pair_via_transfer_from,
  can_mint_via_transfer_to_weth_pair,
  can_mint_via_transfer_to_usdt_pair,
  fetched_at,
  updated_at
FROM project_simulation_result
WHERE project_contract = $1
`, contract.Bytes())
	item, err := scanProjectSimulationResultRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) UpsertProjectPolicyReport(ctx context.Context, item ProjectPolicyReport) error {
	evaluatedAt := item.EvaluatedAt
	if evaluatedAt.IsZero() {
		evaluatedAt = time.Now().UTC()
	}
	r := item.Report
	_, err := s.db.ExecContext(ctx, `
INSERT INTO project_policy_report (
  project_contract,
  is_policy_evaluated,
  is_blacklisted_creator_wallet,
  is_blacklisted_genesis_wallet,
  is_blacklisted_bytecode,
  has_mint_risk,
  evaluated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (project_contract) DO UPDATE
SET is_policy_evaluated = EXCLUDED.is_policy_evaluated,
  is_blacklisted_creator_wallet = EXCLUDED.is_blacklisted_creator_wallet,
  is_blacklisted_genesis_wallet = EXCLUDED.is_blacklisted_genesis_wallet,
  is_blacklisted_bytecode = EXCLUDED.is_blacklisted_bytecode,
  has_mint_risk = EXCLUDED.has_mint_risk,
  evaluated_at = EXCLUDED.evaluated_at,
  updated_at = now()
`, item.ProjectContract.Bytes(), r.IsPolicyEvaluated, r.IsBlacklistedCreatorWallet, r.IsBlacklistedGenesisWallet, r.IsBlacklistedBytecode, r.HasMintRisk, evaluatedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert project policy report: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateProjectReport(ctx context.Context, contract common.Address, report ProjectReport) error {
	return s.UpsertProjectPolicyReport(ctx, ProjectPolicyReport{ProjectContract: contract, Report: report})
}

func (s *SQLStore) GetProjectPolicyReport(ctx context.Context, contract common.Address) (*ProjectPolicyReport, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT project_contract, is_policy_evaluated, is_blacklisted_creator_wallet,
  is_blacklisted_genesis_wallet, is_blacklisted_bytecode, has_mint_risk,
  evaluated_at, updated_at
FROM project_policy_report
WHERE project_contract = $1
`, contract.Bytes())
	item, err := scanProjectPolicyReportRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) ListProjectPolicyReportsByContracts(ctx context.Context, contracts []common.Address) (map[common.Address]ProjectPolicyReport, error) {
	unique := uniqueNonZeroAddresses(contracts)
	result := make(map[common.Address]ProjectPolicyReport, len(unique))
	if len(unique) == 0 {
		return result, nil
	}
	clause, args := addressInClause(unique)
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
SELECT project_contract, is_policy_evaluated, is_blacklisted_creator_wallet,
  is_blacklisted_genesis_wallet, is_blacklisted_bytecode, has_mint_risk,
  evaluated_at, updated_at
FROM project_policy_report
WHERE project_contract IN (%s)
`, clause), args...)
	if err != nil {
		return nil, fmt.Errorf("list project policy reports by contracts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanProjectPolicyReportRow(rows)
		if err != nil {
			return nil, err
		}
		result[item.ProjectContract] = item
	}
	return result, rows.Err()
}

func (s *SQLStore) UpsertProjectBytecodeFact(ctx context.Context, item ProjectBytecodeFact) error {
	fetchedAt := item.FetchedAt
	if fetchedAt.IsZero() {
		fetchedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO project_bytecode_fact (
  project_contract,
  code_hash,
  is_bytecode_blacklisted,
  fetched_at
) VALUES ($1, $2, $3, $4)
ON CONFLICT (project_contract) DO UPDATE
SET code_hash = EXCLUDED.code_hash,
  is_bytecode_blacklisted = EXCLUDED.is_bytecode_blacklisted,
  fetched_at = EXCLUDED.fetched_at,
  updated_at = now()
`, item.ProjectContract.Bytes(), nullableHashBytes(item.CodeHash), item.IsBytecodeBlacklisted, fetchedAt.UTC())
	if err != nil {
		return fmt.Errorf("upsert project bytecode fact: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectBytecodeFact(ctx context.Context, contract common.Address) (*ProjectBytecodeFact, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT project_contract, code_hash, is_bytecode_blacklisted, fetched_at, updated_at
FROM project_bytecode_fact
WHERE project_contract = $1
`, contract.Bytes())
	item, err := scanProjectBytecodeFactRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) UpsertProjectComponentState(ctx context.Context, item ProjectComponentState) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO project_component_state (
  project_contract,
  component,
  status,
  last_attempt_at,
  last_success_at,
  next_run_at,
  last_error
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (project_contract, component) DO UPDATE
SET status = EXCLUDED.status,
  last_attempt_at = EXCLUDED.last_attempt_at,
  last_success_at = EXCLUDED.last_success_at,
  next_run_at = EXCLUDED.next_run_at,
  last_error = EXCLUDED.last_error,
  updated_at = now()
`, item.ProjectContract.Bytes(), item.Component, item.Status, nullableTime(item.LastAttemptAt), nullableTime(item.LastSuccessAt), nullableTime(item.NextRunAt), nullableText(item.LastError))
	if err != nil {
		return fmt.Errorf("upsert project component state: %w", err)
	}
	return nil
}

func (s *SQLStore) GetProjectComponentState(ctx context.Context, contract common.Address, component string) (*ProjectComponentState, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT project_contract, component, status, last_attempt_at, last_success_at, next_run_at, last_error, updated_at
FROM project_component_state
WHERE project_contract = $1 AND component = $2
`, contract.Bytes(), component)
	item, err := scanProjectComponentStateRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

const projectMetaSelectSQL = `
SELECT
  p.block_number,
  p.block_time,
  p.contract,
  p.creator,
  COALESCE(cs.weth_pair, decode(repeat('00', 20), 'hex')) AS weth_pair,
  COALESCE(cs.usdt_pair, decode(repeat('00', 20), 'hex')) AS usdt_pair,
  COALESCE(cs.fetched_at, p.created_at) AS fetch_at,
  p.tx_hash,
  p.tx_index,
  COALESCE(sr.can_mint_from_dead_via_transfer_from, false),
  COALESCE(sr.can_mint_from_zero_via_transfer_from, false),
  COALESCE(sr.can_mint_from_weth_pair_via_transfer_from, false),
  COALESCE(sr.can_mint_from_usdt_pair_via_transfer_from, false),
  COALESCE(sr.can_mint_via_transfer_to_weth_pair, false),
  COALESCE(sr.can_mint_via_transfer_to_usdt_pair, false),
  COALESCE(pr.is_policy_evaluated, false),
  COALESCE(pr.is_blacklisted_creator_wallet, false),
  COALESCE(pr.is_blacklisted_genesis_wallet, false),
  COALESCE(pr.is_blacklisted_bytecode, false),
  COALESCE(pr.has_mint_risk, false),
  gw.last_success_at,
  ch.last_success_at
FROM project p
LEFT JOIN project_chain_state cs ON cs.project_contract = p.contract
LEFT JOIN project_simulation_result sr ON sr.project_contract = p.contract
LEFT JOIN project_policy_report pr ON pr.project_contract = p.contract
LEFT JOIN project_component_state gw ON gw.project_contract = p.contract AND gw.component = 'genesis_wallet'
LEFT JOIN project_component_state ch ON ch.project_contract = p.contract AND ch.component = 'creator_history'
`

type rowScanner interface {
	Scan(dest ...any) error
}

type rowsScanner interface {
	rowScanner
	Next() bool
	Err() error
}

func scanProjectBases(rows rowsScanner) ([]ProjectBase, error) {
	items := make([]ProjectBase, 0)
	for rows.Next() {
		item, err := scanProjectBaseRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project bases: %w", err)
	}
	return items, nil
}

func scanProjectBaseRow(scanner rowScanner) (ProjectBase, error) {
	var base ProjectBase
	var blockNumber, blockTime, txIndex int64
	var contract, creator, txHash []byte
	if err := scanner.Scan(&blockNumber, &blockTime, &contract, &creator, &txHash, &txIndex, &base.CreatedAt); err != nil {
		return ProjectBase{}, fmt.Errorf("scan project base: %w", err)
	}
	if blockNumber < 0 || blockTime < 0 || txIndex < 0 {
		return ProjectBase{}, fmt.Errorf("project base has negative order fields")
	}
	base.BlockNumber = uint64(blockNumber)
	base.BlockTime = uint64(blockTime)
	base.Contract = common.BytesToAddress(contract)
	base.Creator = common.BytesToAddress(creator)
	base.TxHash = common.BytesToHash(txHash)
	base.TxIndex = uint64(txIndex)
	return base, nil
}

func scanProjectMetas(rows rowsScanner) ([]ProjectMeta, error) {
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

func scanProjectMetaRow(scanner rowScanner) (ProjectMeta, error) {
	var meta ProjectMeta
	var blockNumber, blockTime, txIndex int64
	var contract, creator, wethPair, usdtPair, txHash []byte
	var fetchAt time.Time
	var genesisWalletsFetchedAt, creatorHistoricalProjectsFetchedAt sql.NullTime

	if err := scanner.Scan(&blockNumber, &blockTime, &contract, &creator, &wethPair, &usdtPair, &fetchAt, &txHash, &txIndex, &meta.CreatorResult.CanMintFromDeadViaTransferFrom, &meta.CreatorResult.CanMintFromZeroViaTransferFrom, &meta.CreatorResult.CanMintFromWethPairViaTransferFrom, &meta.CreatorResult.CanMintFromUsdtPairViaTransferFrom, &meta.CreatorResult.CanMintViaTransferToWethPair, &meta.CreatorResult.CanMintViaTransferToUsdtPair, &meta.Report.IsPolicyEvaluated, &meta.Report.IsBlacklistedCreatorWallet, &meta.Report.IsBlacklistedGenesisWallet, &meta.Report.IsBlacklistedBytecode, &meta.Report.HasMintRisk, &genesisWalletsFetchedAt, &creatorHistoricalProjectsFetchedAt); err != nil {
		return ProjectMeta{}, fmt.Errorf("scan project meta: %w", err)
	}
	if blockNumber < 0 || blockTime < 0 || txIndex < 0 {
		return ProjectMeta{}, fmt.Errorf("project meta has negative order fields")
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
	if genesisWalletsFetchedAt.Valid {
		meta.GenesisWalletsFetchedAt = genesisWalletsFetchedAt.Time
	}
	if creatorHistoricalProjectsFetchedAt.Valid {
		meta.CreatorHistoricalProjectsFetchedAt = creatorHistoricalProjectsFetchedAt.Time
	}
	return meta, nil
}

func scanProjectChainStateRow(scanner rowScanner) (ProjectChainState, error) {
	var item ProjectChainState
	var contract, wethPair, usdtPair []byte
	var raw []byte
	var tokenName, tokenSymbol sql.NullString
	if err := scanner.Scan(&contract, &raw, &wethPair, &usdtPair, &tokenName, &tokenSymbol, &item.FetchedAt, &item.UpdatedAt); err != nil {
		return ProjectChainState{}, fmt.Errorf("scan project chain state: %w", err)
	}
	item.ProjectContract = common.BytesToAddress(contract)
	item.RawChainState = append(json.RawMessage(nil), raw...)
	_ = json.Unmarshal(raw, &item.ChainState)
	item.WethPair = common.BytesToAddress(wethPair)
	item.UsdtPair = common.BytesToAddress(usdtPair)
	item.TokenName = tokenName.String
	item.TokenSymbol = tokenSymbol.String
	return item, nil
}

func scanProjectSimulationResultRow(scanner rowScanner) (ProjectSimulationResult, error) {
	var item ProjectSimulationResult
	var contract []byte
	if err := scanner.Scan(&contract, &item.Result.CanMintFromDeadViaTransferFrom, &item.Result.CanMintFromZeroViaTransferFrom, &item.Result.CanMintFromWethPairViaTransferFrom, &item.Result.CanMintFromUsdtPairViaTransferFrom, &item.Result.CanMintViaTransferToWethPair, &item.Result.CanMintViaTransferToUsdtPair, &item.FetchedAt, &item.UpdatedAt); err != nil {
		return ProjectSimulationResult{}, fmt.Errorf("scan project simulation result: %w", err)
	}
	item.ProjectContract = common.BytesToAddress(contract)
	return item, nil
}

func scanProjectPolicyReportRow(scanner rowScanner) (ProjectPolicyReport, error) {
	var item ProjectPolicyReport
	var contract []byte
	if err := scanner.Scan(&contract, &item.Report.IsPolicyEvaluated, &item.Report.IsBlacklistedCreatorWallet, &item.Report.IsBlacklistedGenesisWallet, &item.Report.IsBlacklistedBytecode, &item.Report.HasMintRisk, &item.EvaluatedAt, &item.UpdatedAt); err != nil {
		return ProjectPolicyReport{}, fmt.Errorf("scan project policy report: %w", err)
	}
	item.ProjectContract = common.BytesToAddress(contract)
	return item, nil
}

func scanProjectBytecodeFactRow(scanner rowScanner) (ProjectBytecodeFact, error) {
	var item ProjectBytecodeFact
	var contract []byte
	var codeHash []byte
	if err := scanner.Scan(&contract, &codeHash, &item.IsBytecodeBlacklisted, &item.FetchedAt, &item.UpdatedAt); err != nil {
		return ProjectBytecodeFact{}, fmt.Errorf("scan project bytecode fact: %w", err)
	}
	item.ProjectContract = common.BytesToAddress(contract)
	if len(codeHash) > 0 {
		item.CodeHash = common.BytesToHash(codeHash)
	}
	return item, nil
}

func scanProjectComponentStateRow(scanner rowScanner) (ProjectComponentState, error) {
	var item ProjectComponentState
	var contract []byte
	var lastAttemptAt, lastSuccessAt, nextRunAt sql.NullTime
	var lastError sql.NullString
	if err := scanner.Scan(&contract, &item.Component, &item.Status, &lastAttemptAt, &lastSuccessAt, &nextRunAt, &lastError, &item.UpdatedAt); err != nil {
		return ProjectComponentState{}, fmt.Errorf("scan project component state: %w", err)
	}
	item.ProjectContract = common.BytesToAddress(contract)
	if lastAttemptAt.Valid {
		item.LastAttemptAt = lastAttemptAt.Time
	}
	if lastSuccessAt.Valid {
		item.LastSuccessAt = lastSuccessAt.Time
	}
	if nextRunAt.Valid {
		item.NextRunAt = nextRunAt.Time
	}
	item.LastError = lastError.String
	return item, nil
}

func validateProjectBaseNumbers(blockNumber, blockTime, txIndex uint64) error {
	if blockTime > math.MaxInt64 {
		return fmt.Errorf("project base block time %d exceeds postgres BIGINT", blockTime)
	}
	return validateProjectOrderNumbers(blockNumber, txIndex)
}

func validateProjectOrderNumbers(blockNumber, txIndex uint64) error {
	if blockNumber > math.MaxInt64 {
		return fmt.Errorf("project base block number %d exceeds postgres BIGINT", blockNumber)
	}
	if txIndex > math.MaxInt64 {
		return fmt.Errorf("project base tx index %d exceeds postgres BIGINT", txIndex)
	}
	return nil
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

func addressInClause(items []common.Address) (string, []any) {
	placeholders := make([]string, 0, len(items))
	args := make([]any, 0, len(items))
	for i, item := range items {
		placeholders = append(placeholders, fmt.Sprintf("$%d", i+1))
		args = append(args, item.Bytes())
	}
	return strings.Join(placeholders, ", "), args
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
