package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
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
  source_quality_report_origin, created_at, updated_at
FROM bytecode
WHERE code_hash = $1
`, codeHash.Bytes())
	item, err := scanBytecode(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

func (s *SQLStore) UpdateBytecodeSourceQualityReport(ctx context.Context, codeHash common.Hash, report string, origin string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE bytecode
SET source_quality_report = $2,
  source_quality_report_fetched_at = now(),
  source_quality_report_origin = $3,
  updated_at = now()
WHERE code_hash = $1
`, codeHash.Bytes(), report, nullableTrimmedText(origin))
	if err != nil {
		return fmt.Errorf("update bytecode source quality report: %w", err)
	}
	return nil
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
		if errors.Is(err, sql.ErrNoRows) {
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
	if err := scanner.Scan(&codeHash, &item.RuntimeBytecode, &sourceCode, &sourceCodeHash, &sourceCodeFetchedAt, &sourceCodeOrigin, &sourceQualityReport, &sourceQualityReportFetchedAt, &sourceQualityReportOrigin, &item.CreatedAt, &item.UpdatedAt); err != nil {
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

func nullableAddressBytes(value common.Address) any {
	if value == (common.Address{}) {
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
