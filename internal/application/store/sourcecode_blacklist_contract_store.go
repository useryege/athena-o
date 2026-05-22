package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrSourcecodeBlacklistContractAlreadyExists = errors.New("sourcecode blacklist contract already exists")
	ErrSourcecodeBlacklistContractNotFound      = errors.New("sourcecode blacklist contract not found")
)

type SourcecodeBlacklistContract struct {
	Contract   common.Address
	SourceHash common.Hash
	Note       string
	CreatedAt  time.Time
}

type SourcecodeBlacklistContractStore interface {
	ListSourcecodeBlacklistContracts(ctx context.Context) ([]SourcecodeBlacklistContract, error)
	AddSourcecodeBlacklistContract(ctx context.Context, item SourcecodeBlacklistContract) error
	UpdateSourcecodeBlacklistContractNote(ctx context.Context, contract common.Address, note string) error
	DeleteSourcecodeBlacklistContract(ctx context.Context, contract common.Address) error
}

func (s *SQLStore) ListSourcecodeBlacklistContracts(ctx context.Context) ([]SourcecodeBlacklistContract, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT contract, source_hash, note, created_at
FROM sourcecode_blacklist_contract
ORDER BY created_at DESC, contract
`)
	if err != nil {
		return nil, fmt.Errorf("list sourcecode blacklist contracts: %w", err)
	}
	defer rows.Close()

	items := make([]SourcecodeBlacklistContract, 0)
	for rows.Next() {
		item, err := scanSourcecodeBlacklistContractRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sourcecode blacklist contracts: %w", err)
	}
	return items, nil
}

func (s *SQLStore) AddSourcecodeBlacklistContract(ctx context.Context, item SourcecodeBlacklistContract) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO sourcecode_blacklist_contract (contract, source_hash, note)
VALUES ($1, $2, $3)
`, item.Contract.Bytes(), item.SourceHash.Bytes(), nullableTrimmedText(item.Note))
	if err != nil {
		if isUniqueViolation(err) {
			return ErrSourcecodeBlacklistContractAlreadyExists
		}
		return fmt.Errorf("add sourcecode blacklist contract: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateSourcecodeBlacklistContractNote(ctx context.Context, contract common.Address, note string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE sourcecode_blacklist_contract
SET note = $2
WHERE contract = $1
`, contract.Bytes(), nullableTrimmedText(note))
	if err != nil {
		return fmt.Errorf("update sourcecode blacklist contract note: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for update sourcecode blacklist contract note: %w", err)
	}
	if affected == 0 {
		return ErrSourcecodeBlacklistContractNotFound
	}
	return nil
}

func (s *SQLStore) DeleteSourcecodeBlacklistContract(ctx context.Context, contract common.Address) error {
	result, err := s.db.ExecContext(ctx, `
DELETE FROM sourcecode_blacklist_contract
WHERE contract = $1
`, contract.Bytes())
	if err != nil {
		return fmt.Errorf("delete sourcecode blacklist contract: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for delete sourcecode blacklist contract: %w", err)
	}
	if affected == 0 {
		return ErrSourcecodeBlacklistContractNotFound
	}
	return nil
}

func scanSourcecodeBlacklistContractRow(scanner rowScanner) (SourcecodeBlacklistContract, error) {
	var (
		contract   []byte
		sourceHash []byte
		note       sql.NullString
		createdAt  time.Time
	)
	if err := scanner.Scan(&contract, &sourceHash, &note, &createdAt); err != nil {
		return SourcecodeBlacklistContract{}, fmt.Errorf("scan sourcecode blacklist contract: %w", err)
	}
	return SourcecodeBlacklistContract{
		Contract:   common.BytesToAddress(contract),
		SourceHash: common.BytesToHash(sourceHash),
		Note:       note.String,
		CreatedAt:  createdAt,
	}, nil
}
