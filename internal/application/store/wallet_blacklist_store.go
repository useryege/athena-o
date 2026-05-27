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
	ErrWalletBlacklistEntryAlreadyExists = errors.New("wallet blacklist entry already exists")
	ErrWalletBlacklistEntryNotFound      = errors.New("wallet blacklist entry not found")
)

type WalletBlacklistEntry struct {
	Wallet    common.Address
	Note      string
	CreatedAt time.Time
}

type WalletBlacklistStore interface {
	ListWalletBlacklistEntries(ctx context.Context) ([]WalletBlacklistEntry, error)
	AddWalletBlacklistEntry(ctx context.Context, item WalletBlacklistEntry) error
	UpdateWalletBlacklistEntryNote(ctx context.Context, wallet common.Address, note string) error
	DeleteWalletBlacklistEntry(ctx context.Context, wallet common.Address) error
}

func (s *SQLStore) ListWalletBlacklistEntries(ctx context.Context) ([]WalletBlacklistEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT wallet, note, created_at
FROM wallet_blacklist
ORDER BY created_at DESC, wallet
`)
	if err != nil {
		return nil, fmt.Errorf("list wallet blacklist entries: %w", err)
	}
	defer rows.Close()

	items := make([]WalletBlacklistEntry, 0)
	for rows.Next() {
		item, err := scanWalletBlacklistEntryRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wallet blacklist entries: %w", err)
	}
	return items, nil
}

func (s *SQLStore) AddWalletBlacklistEntry(ctx context.Context, item WalletBlacklistEntry) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO wallet_blacklist (wallet, note)
VALUES ($1, $2)
`, item.Wallet.Bytes(), nullableTrimmedText(item.Note))
	if err != nil {
		if isUniqueViolation(err) {
			return ErrWalletBlacklistEntryAlreadyExists
		}
		return fmt.Errorf("add wallet blacklist entry: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateWalletBlacklistEntryNote(ctx context.Context, wallet common.Address, note string) error {
	result, err := s.db.ExecContext(ctx, `
UPDATE wallet_blacklist
SET note = $2
WHERE wallet = $1
`, wallet.Bytes(), nullableTrimmedText(note))
	if err != nil {
		return fmt.Errorf("update wallet blacklist entry note: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for update wallet blacklist entry note: %w", err)
	}
	if affected == 0 {
		return ErrWalletBlacklistEntryNotFound
	}
	return nil
}

func (s *SQLStore) DeleteWalletBlacklistEntry(ctx context.Context, wallet common.Address) error {
	result, err := s.db.ExecContext(ctx, `
DELETE FROM wallet_blacklist
WHERE wallet = $1
`, wallet.Bytes())
	if err != nil {
		return fmt.Errorf("delete wallet blacklist entry: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for delete wallet blacklist entry: %w", err)
	}
	if affected == 0 {
		return ErrWalletBlacklistEntryNotFound
	}
	return nil
}

func scanWalletBlacklistEntryRow(scanner rowScanner) (WalletBlacklistEntry, error) {
	var (
		wallet    []byte
		note      sql.NullString
		createdAt time.Time
	)
	if err := scanner.Scan(&wallet, &note, &createdAt); err != nil {
		return WalletBlacklistEntry{}, fmt.Errorf("scan wallet blacklist entry: %w", err)
	}
	return WalletBlacklistEntry{
		Wallet:    common.BytesToAddress(wallet),
		Note:      note.String,
		CreatedAt: createdAt,
	}, nil
}

func nullableTrimmedText(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
