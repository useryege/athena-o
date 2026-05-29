package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrBytecodeBlacklistAlreadyExists = errors.New("bytecode blacklist entry already exists")
	ErrBytecodeBlacklistNotFound      = errors.New("bytecode blacklist entry not found")
)

type Bytecode struct {
	CodeHash                     common.Hash
	RuntimeBytecode              []byte
	SourceCode                   string
	SourceCodeHash               common.Hash
	SourceCodeFetchedAt          time.Time
	SourceCodeOrigin             string
	SourceQualityReport          string
	SourceQualityReportFetchedAt time.Time
	SourceQualityReportOrigin    string
	SourceQualityPromptVersion   int64
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

type ContractBytecodeDeployment struct {
	ChainID     int64
	Contract    common.Address
	CodeHash    common.Hash
	FirstSeenAt time.Time
	UpdatedAt   time.Time
}

type BytecodeBlacklistEntry struct {
	CodeHash       common.Hash
	Note           string
	SourceChainID  int64
	SourceContract common.Address
	CreatedAt      time.Time
}

type BytecodeListRecord struct {
	CodeHash              common.Hash
	RuntimeBytecodeSize   int64
	DeploymentCount       int64
	IsOpenSource          bool
	IsBytecodeBlacklisted bool
	CreatedAt             time.Time
	UpdatedAt             time.Time
	Total                 int64
}

type BytecodeDetailRecord struct {
	Bytecode
	RuntimeBytecodeSize   int64
	DeploymentCount       int64
	IsBytecodeBlacklisted bool
}

type BytecodeDeploymentRecord struct {
	ContractBytecodeDeployment
	Total int64
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *SQLStore) UpsertBytecode(ctx context.Context, codeHash common.Hash, runtimeBytecode []byte) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO bytecode (code_hash, runtime_bytecode)
VALUES ($1, $2)
ON CONFLICT (code_hash) DO UPDATE
SET runtime_bytecode = EXCLUDED.runtime_bytecode,
  updated_at = now()
`, codeHash.Bytes(), runtimeBytecode)
	if err != nil {
		return fmt.Errorf("upsert bytecode: %w", err)
	}
	return nil
}

func (s *SQLStore) UpsertContractBytecodeDeployment(ctx context.Context, item ContractBytecodeDeployment) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO contract_bytecode_deployment (chain_id, contract, code_hash)
VALUES ($1, $2, $3)
ON CONFLICT (chain_id, contract) DO UPDATE
SET code_hash = EXCLUDED.code_hash,
  updated_at = now()
`, item.ChainID, item.Contract.Bytes(), item.CodeHash.Bytes())
	if err != nil {
		return fmt.Errorf("upsert contract bytecode deployment: %w", err)
	}
	return nil
}

func (s *SQLStore) GetBytecode(ctx context.Context, codeHash common.Hash) (*Bytecode, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT code_hash, runtime_bytecode, source_code, source_code_hash, source_code_fetched_at,
  source_code_origin, source_quality_report, source_quality_report_fetched_at,
  source_quality_report_origin, source_quality_prompt_version, created_at, updated_at
FROM bytecode
WHERE code_hash = $1
`, codeHash.Bytes())
	item, err := scanBytecode(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) UpdateBytecodeSourceCode(ctx context.Context, codeHash common.Hash, sourceCode string, sourceCodeHash common.Hash, origin string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE bytecode
SET source_code = $2,
  source_code_hash = $3,
  source_code_fetched_at = now(),
  source_code_origin = $4,
  updated_at = now()
WHERE code_hash = $1
`, codeHash.Bytes(), sourceCode, nullableHashBytes(sourceCodeHash), nullableTrimmedText(origin))
	if err != nil {
		return fmt.Errorf("update bytecode source code: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateBytecodeSourceQualityReport(ctx context.Context, codeHash common.Hash, report string, origin string, promptVersion int64) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE bytecode
SET source_quality_report = $2,
  source_quality_report_fetched_at = now(),
  source_quality_report_origin = $3,
  source_quality_prompt_version = $4,
  updated_at = now()
WHERE code_hash = $1
`, codeHash.Bytes(), report, nullableTrimmedText(origin), promptVersion)
	if err != nil {
		return fmt.Errorf("update bytecode source quality report: %w", err)
	}
	return nil
}

func (s *SQLStore) ListBytecodes(ctx context.Context, codeHash *common.Hash, limit, offset int64) ([]BytecodeListRecord, int64, error) {
	rows, err := s.db.QueryContext(ctx, `
WITH deployment_counts AS (
  SELECT code_hash, COUNT(*)::bigint AS deployment_count
  FROM contract_bytecode_deployment
  GROUP BY code_hash
),
filtered AS (
  SELECT b.code_hash,
    length(b.runtime_bytecode)::bigint AS runtime_bytecode_size,
    COALESCE(dc.deployment_count, 0)::bigint AS deployment_count,
    COALESCE(btrim(b.source_code), '') <> '' AS is_open_source,
    bl.code_hash IS NOT NULL AS is_bytecode_blacklisted,
    b.created_at,
    b.updated_at
  FROM bytecode b
  LEFT JOIN deployment_counts dc ON dc.code_hash = b.code_hash
  LEFT JOIN bytecode_blacklist bl ON bl.code_hash = b.code_hash
  WHERE ($1::bytea IS NULL OR b.code_hash = $1)
)
SELECT code_hash, runtime_bytecode_size, deployment_count, is_open_source,
  is_bytecode_blacklisted, created_at, updated_at, COUNT(*) OVER()::bigint AS total
FROM filtered
ORDER BY updated_at DESC, code_hash
LIMIT $2 OFFSET $3
`, nullableHashPtrBytes(codeHash), limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list bytecodes: %w", err)
	}
	defer rows.Close()

	items := make([]BytecodeListRecord, 0)
	var total int64
	for rows.Next() {
		item, err := scanBytecodeListRecord(rows)
		if err != nil {
			return nil, 0, err
		}
		if total == 0 {
			total = item.Total
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate bytecodes: %w", err)
	}
	return items, total, nil
}

func (s *SQLStore) GetBytecodeDetail(ctx context.Context, codeHash common.Hash) (*BytecodeDetailRecord, error) {
	row := s.db.QueryRowContext(ctx, `
WITH deployment_counts AS (
  SELECT code_hash, COUNT(*)::bigint AS deployment_count
  FROM contract_bytecode_deployment
  WHERE code_hash = $1
  GROUP BY code_hash
)
SELECT b.code_hash, b.runtime_bytecode, b.source_code, b.source_code_hash, b.source_code_fetched_at,
  b.source_code_origin, b.source_quality_report, b.source_quality_report_fetched_at,
  b.source_quality_report_origin, b.source_quality_prompt_version, b.created_at, b.updated_at,
  length(b.runtime_bytecode)::bigint AS runtime_bytecode_size,
  COALESCE(dc.deployment_count, 0)::bigint AS deployment_count,
  bl.code_hash IS NOT NULL AS is_bytecode_blacklisted
FROM bytecode b
LEFT JOIN deployment_counts dc ON dc.code_hash = b.code_hash
LEFT JOIN bytecode_blacklist bl ON bl.code_hash = b.code_hash
WHERE b.code_hash = $1
`, codeHash.Bytes())
	item, err := scanBytecodeDetailRecord(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) ListBytecodeDeployments(ctx context.Context, codeHash common.Hash, chainID int64, contract *common.Address, limit, offset int64) ([]BytecodeDeploymentRecord, int64, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT chain_id, contract, code_hash, first_seen_at, updated_at, COUNT(*) OVER()::bigint AS total
FROM contract_bytecode_deployment
WHERE code_hash = $1
  AND ($2::bigint IS NULL OR chain_id = $2)
  AND ($3::bytea IS NULL OR contract = $3)
ORDER BY updated_at DESC, chain_id, contract
LIMIT $4 OFFSET $5
`, codeHash.Bytes(), nullableInt64(chainID), nullableAddressPtrBytes(contract), limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list bytecode deployments: %w", err)
	}
	defer rows.Close()

	items := make([]BytecodeDeploymentRecord, 0)
	var total int64
	for rows.Next() {
		item, err := scanBytecodeDeploymentRecord(rows)
		if err != nil {
			return nil, 0, err
		}
		if total == 0 {
			total = item.Total
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate bytecode deployments: %w", err)
	}
	return items, total, nil
}

func (s *SQLStore) IsBytecodeBlacklisted(ctx context.Context, codeHash common.Hash) (bool, error) {
	var exists bool
	if err := s.db.QueryRowContext(ctx, `
SELECT EXISTS(SELECT 1 FROM bytecode_blacklist WHERE code_hash = $1)
`, codeHash.Bytes()).Scan(&exists); err != nil {
		return false, fmt.Errorf("check bytecode blacklist: %w", err)
	}
	return exists, nil
}

func (s *SQLStore) ListBytecodeBlacklistEntries(ctx context.Context) ([]BytecodeBlacklistEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT code_hash, note, source_chain_id, source_contract, created_at
FROM bytecode_blacklist
ORDER BY created_at DESC, code_hash
`)
	if err != nil {
		return nil, fmt.Errorf("list bytecode blacklist entries: %w", err)
	}
	defer rows.Close()

	items := make([]BytecodeBlacklistEntry, 0)
	for rows.Next() {
		item, err := scanBytecodeBlacklistEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bytecode blacklist entries: %w", err)
	}
	return items, nil
}

func (s *SQLStore) AddBytecodeBlacklistEntry(ctx context.Context, item BytecodeBlacklistEntry) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO bytecode_blacklist (code_hash, note, source_chain_id, source_contract)
VALUES ($1, $2, $3, $4)
`, item.CodeHash.Bytes(), nullableTrimmedText(item.Note), nullableInt64(item.SourceChainID), nullableAddressBytes(item.SourceContract))
	if err != nil {
		if isUniqueViolation(err) {
			return ErrBytecodeBlacklistAlreadyExists
		}
		return fmt.Errorf("add bytecode blacklist entry: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateBytecodeBlacklistNote(ctx context.Context, codeHash common.Hash, note string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE bytecode_blacklist
SET note = $2
WHERE code_hash = $1
`, codeHash.Bytes(), nullableTrimmedText(note))
	if err != nil {
		return fmt.Errorf("update bytecode blacklist note: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for update bytecode blacklist note: %w", err)
	}
	if affected == 0 {
		return ErrBytecodeBlacklistNotFound
	}
	return nil
}

func (s *SQLStore) DeleteBytecodeBlacklist(ctx context.Context, codeHash common.Hash) error {
	result, err := s.db.ExecContext(ctx, `
DELETE FROM bytecode_blacklist
WHERE code_hash = $1
`, codeHash.Bytes())
	if err != nil {
		return fmt.Errorf("delete bytecode blacklist: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for delete bytecode blacklist: %w", err)
	}
	if affected == 0 {
		return ErrBytecodeBlacklistNotFound
	}
	return nil
}

func (s *SQLStore) GetBytecodeBlacklistEntry(ctx context.Context, codeHash common.Hash) (*BytecodeBlacklistEntry, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT code_hash, note, source_chain_id, source_contract, created_at
FROM bytecode_blacklist
WHERE code_hash = $1
`, codeHash.Bytes())
	item, err := scanBytecodeBlacklistEntry(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func scanBytecode(scanner rowScanner) (Bytecode, error) {
	var (
		item                         Bytecode
		codeHash                     []byte
		sourceCode                   sql.NullString
		sourceCodeHash               []byte
		sourceCodeFetchedAt          sql.NullTime
		sourceCodeOrigin             sql.NullString
		sourceQualityReport          sql.NullString
		sourceQualityReportFetchedAt sql.NullTime
		sourceQualityReportOrigin    sql.NullString
	)
	if err := scanner.Scan(&codeHash, &item.RuntimeBytecode, &sourceCode, &sourceCodeHash, &sourceCodeFetchedAt, &sourceCodeOrigin, &sourceQualityReport, &sourceQualityReportFetchedAt, &sourceQualityReportOrigin, &item.SourceQualityPromptVersion, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Bytecode{}, fmt.Errorf("scan bytecode: %w", err)
	}
	item.CodeHash = common.BytesToHash(codeHash)
	item.SourceCode = sourceCode.String
	item.SourceCodeHash = common.BytesToHash(sourceCodeHash)
	if sourceCodeFetchedAt.Valid {
		item.SourceCodeFetchedAt = sourceCodeFetchedAt.Time
	}
	item.SourceCodeOrigin = sourceCodeOrigin.String
	item.SourceQualityReport = sourceQualityReport.String
	if sourceQualityReportFetchedAt.Valid {
		item.SourceQualityReportFetchedAt = sourceQualityReportFetchedAt.Time
	}
	item.SourceQualityReportOrigin = sourceQualityReportOrigin.String
	return item, nil
}

func scanBytecodeListRecord(scanner rowScanner) (BytecodeListRecord, error) {
	var (
		item     BytecodeListRecord
		codeHash []byte
	)
	if err := scanner.Scan(&codeHash, &item.RuntimeBytecodeSize, &item.DeploymentCount, &item.IsOpenSource, &item.IsBytecodeBlacklisted, &item.CreatedAt, &item.UpdatedAt, &item.Total); err != nil {
		return BytecodeListRecord{}, fmt.Errorf("scan bytecode list record: %w", err)
	}
	item.CodeHash = common.BytesToHash(codeHash)
	return item, nil
}

func scanBytecodeDetailRecord(scanner rowScanner) (BytecodeDetailRecord, error) {
	var item BytecodeDetailRecord
	bytecode, err := scanBytecodeWithExtra(scanner, &item.RuntimeBytecodeSize, &item.DeploymentCount, &item.IsBytecodeBlacklisted)
	if err != nil {
		return BytecodeDetailRecord{}, fmt.Errorf("scan bytecode detail record: %w", err)
	}
	item.Bytecode = bytecode
	return item, nil
}

func scanBytecodeWithExtra(scanner rowScanner, extra ...any) (Bytecode, error) {
	var (
		item                         Bytecode
		codeHash                     []byte
		sourceCode                   sql.NullString
		sourceCodeHash               []byte
		sourceCodeFetchedAt          sql.NullTime
		sourceCodeOrigin             sql.NullString
		sourceQualityReport          sql.NullString
		sourceQualityReportFetchedAt sql.NullTime
		sourceQualityReportOrigin    sql.NullString
	)
	dest := []any{&codeHash, &item.RuntimeBytecode, &sourceCode, &sourceCodeHash, &sourceCodeFetchedAt, &sourceCodeOrigin, &sourceQualityReport, &sourceQualityReportFetchedAt, &sourceQualityReportOrigin, &item.SourceQualityPromptVersion, &item.CreatedAt, &item.UpdatedAt}
	dest = append(dest, extra...)
	if err := scanner.Scan(dest...); err != nil {
		return Bytecode{}, err
	}
	item.CodeHash = common.BytesToHash(codeHash)
	item.SourceCode = sourceCode.String
	item.SourceCodeHash = common.BytesToHash(sourceCodeHash)
	if sourceCodeFetchedAt.Valid {
		item.SourceCodeFetchedAt = sourceCodeFetchedAt.Time
	}
	item.SourceCodeOrigin = sourceCodeOrigin.String
	item.SourceQualityReport = sourceQualityReport.String
	if sourceQualityReportFetchedAt.Valid {
		item.SourceQualityReportFetchedAt = sourceQualityReportFetchedAt.Time
	}
	item.SourceQualityReportOrigin = sourceQualityReportOrigin.String
	return item, nil
}

func scanBytecodeDeploymentRecord(scanner rowScanner) (BytecodeDeploymentRecord, error) {
	var (
		item     BytecodeDeploymentRecord
		contract []byte
		codeHash []byte
	)
	if err := scanner.Scan(&item.ChainID, &contract, &codeHash, &item.FirstSeenAt, &item.UpdatedAt, &item.Total); err != nil {
		return BytecodeDeploymentRecord{}, fmt.Errorf("scan bytecode deployment record: %w", err)
	}
	item.Contract = common.BytesToAddress(contract)
	item.CodeHash = common.BytesToHash(codeHash)
	return item, nil
}

func scanBytecodeBlacklistEntry(scanner rowScanner) (BytecodeBlacklistEntry, error) {
	var (
		item           BytecodeBlacklistEntry
		codeHash       []byte
		note           sql.NullString
		sourceChainID  sql.NullInt64
		sourceContract []byte
	)
	if err := scanner.Scan(&codeHash, &note, &sourceChainID, &sourceContract, &item.CreatedAt); err != nil {
		return BytecodeBlacklistEntry{}, fmt.Errorf("scan bytecode blacklist entry: %w", err)
	}
	item.CodeHash = common.BytesToHash(codeHash)
	item.Note = note.String
	if sourceChainID.Valid {
		item.SourceChainID = sourceChainID.Int64
	}
	item.SourceContract = common.BytesToAddress(sourceContract)
	return item, nil
}

func nullableTrimmedText(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableHashBytes(value common.Hash) any {
	if value == (common.Hash{}) {
		return nil
	}
	return value.Bytes()
}

func nullableHashPtrBytes(value *common.Hash) any {
	if value == nil {
		return nil
	}
	return value.Bytes()
}

func nullableAddressBytes(value common.Address) any {
	if value == (common.Address{}) {
		return nil
	}
	return value.Bytes()
}

func nullableAddressPtrBytes(value *common.Address) any {
	if value == nil {
		return nil
	}
	return value.Bytes()
}

func nullableInt64(value int64) any {
	if value == 0 {
		return nil
	}
	return value
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
