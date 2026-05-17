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
	ErrBytecodeBlacklistContractAlreadyExists = errors.New("bytecode blacklist contract already exists")
	ErrBytecodeBlacklistContractNotFound      = errors.New("bytecode blacklist contract not found")
)

type BytecodeBlacklistContract struct {
	Contract  common.Address
	CodeHash  common.Hash
	Note      string
	CreatedAt time.Time
}

type BytecodeBlacklistContractStore interface {
	ListBytecodeBlacklistContracts(ctx context.Context) ([]BytecodeBlacklistContract, error)
	AddBytecodeBlacklistContract(ctx context.Context, item BytecodeBlacklistContract) error
	UpdateBytecodeBlacklistContractNote(ctx context.Context, contract common.Address, note string) error
	DeleteBytecodeBlacklistContract(ctx context.Context, contract common.Address) error
}

func (s *SQLStore) ListBytecodeBlacklistContracts(ctx context.Context) ([]BytecodeBlacklistContract, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT contract, code_hash, note, created_at
FROM bytecode_blacklist_contract
ORDER BY created_at DESC, contract
`)
	if err != nil {
		return nil, fmt.Errorf("list bytecode blacklist contracts: %w", err)
	}
	defer rows.Close()

	items := make([]BytecodeBlacklistContract, 0)
	for rows.Next() {
		item, err := scanBytecodeBlacklistContractRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bytecode blacklist contracts: %w", err)
	}
	return items, nil
}

func (s *SQLStore) AddBytecodeBlacklistContract(ctx context.Context, item BytecodeBlacklistContract) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO bytecode_blacklist_contract (contract, code_hash, note)
VALUES ($1, $2, $3)
`, item.Contract.Bytes(), item.CodeHash.Bytes(), nullableTrimmedText(item.Note))
	if err != nil {
		if isUniqueViolation(err) {
			return ErrBytecodeBlacklistContractAlreadyExists
		}
		return fmt.Errorf("add bytecode blacklist contract: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateBytecodeBlacklistContractNote(ctx context.Context, contract common.Address, note string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE bytecode_blacklist_contract
SET note = $2
WHERE contract = $1
`, contract.Bytes(), nullableTrimmedText(note))
	if err != nil {
		return fmt.Errorf("update bytecode blacklist contract note: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for update bytecode blacklist contract note: %w", err)
	}
	if affected == 0 {
		return ErrBytecodeBlacklistContractNotFound
	}
	return nil
}

func (s *SQLStore) DeleteBytecodeBlacklistContract(ctx context.Context, contract common.Address) error {
	result, err := s.db.ExecContext(ctx, `
DELETE FROM bytecode_blacklist_contract
WHERE contract = $1
`, contract.Bytes())
	if err != nil {
		return fmt.Errorf("delete bytecode blacklist contract: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for delete bytecode blacklist contract: %w", err)
	}
	if affected == 0 {
		return ErrBytecodeBlacklistContractNotFound
	}
	return nil
}

func scanBytecodeBlacklistContractRow(scanner rowScanner) (BytecodeBlacklistContract, error) {
	var (
		contract  []byte
		codeHash  []byte
		note      sql.NullString
		createdAt time.Time
	)
	if err := scanner.Scan(&contract, &codeHash, &note, &createdAt); err != nil {
		return BytecodeBlacklistContract{}, fmt.Errorf("scan bytecode blacklist contract: %w", err)
	}
	return BytecodeBlacklistContract{
		Contract:  common.BytesToAddress(contract),
		CodeHash:  common.BytesToHash(codeHash),
		Note:      note.String,
		CreatedAt: createdAt,
	}, nil
}

func nullableTrimmedText(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
