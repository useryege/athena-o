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
	ErrWalletBlacklistContractAlreadyExists = errors.New("wallet blacklist contract already exists")
	ErrWalletBlacklistContractNotFound      = errors.New("wallet blacklist contract not found")
)

type WalletBlacklistContract struct {
	Contract  common.Address
	Note      string
	CreatedAt time.Time
}

type WalletBlacklistContractStore interface {
	ListWalletBlacklistContracts(ctx context.Context) ([]WalletBlacklistContract, error)
	AddWalletBlacklistContract(ctx context.Context, item WalletBlacklistContract) error
	UpdateWalletBlacklistContractNote(ctx context.Context, contract common.Address, note string) error
	DeleteWalletBlacklistContract(ctx context.Context, contract common.Address) error
}

func (s *SQLStore) ListWalletBlacklistContracts(ctx context.Context) ([]WalletBlacklistContract, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT contract, note, created_at
FROM wallet_blacklist_contract
ORDER BY created_at DESC, contract
`)
	if err != nil {
		return nil, fmt.Errorf("list wallet blacklist contracts: %w", err)
	}
	defer rows.Close()

	items := make([]WalletBlacklistContract, 0)
	for rows.Next() {
		item, err := scanWalletBlacklistContractRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wallet blacklist contracts: %w", err)
	}
	return items, nil
}

func (s *SQLStore) AddWalletBlacklistContract(ctx context.Context, item WalletBlacklistContract) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO wallet_blacklist_contract (contract, note)
VALUES ($1, $2)
`, item.Contract.Bytes(), nullableTrimmedText(item.Note))
	if err != nil {
		if isUniqueViolation(err) {
			return ErrWalletBlacklistContractAlreadyExists
		}
		return fmt.Errorf("add wallet blacklist contract: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateWalletBlacklistContractNote(ctx context.Context, contract common.Address, note string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE wallet_blacklist_contract
SET note = $2
WHERE contract = $1
`, contract.Bytes(), nullableTrimmedText(note))
	if err != nil {
		return fmt.Errorf("update wallet blacklist contract note: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for update wallet blacklist contract note: %w", err)
	}
	if affected == 0 {
		return ErrWalletBlacklistContractNotFound
	}
	return nil
}

func (s *SQLStore) DeleteWalletBlacklistContract(ctx context.Context, contract common.Address) error {
	result, err := s.db.ExecContext(ctx, `
DELETE FROM wallet_blacklist_contract
WHERE contract = $1
`, contract.Bytes())
	if err != nil {
		return fmt.Errorf("delete wallet blacklist contract: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for delete wallet blacklist contract: %w", err)
	}
	if affected == 0 {
		return ErrWalletBlacklistContractNotFound
	}
	return nil
}

func scanWalletBlacklistContractRow(scanner rowScanner) (WalletBlacklistContract, error) {
	var (
		contract  []byte
		note      sql.NullString
		createdAt time.Time
	)
	if err := scanner.Scan(&contract, &note, &createdAt); err != nil {
		return WalletBlacklistContract{}, fmt.Errorf("scan wallet blacklist contract: %w", err)
	}
	return WalletBlacklistContract{
		Contract:  common.BytesToAddress(contract),
		Note:      note.String,
		CreatedAt: createdAt,
	}, nil
}
