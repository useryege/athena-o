package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgtype"
	walletsqlc "github.com/useryege/athena/internal/wallet/store/sqlc"
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

func (s *SQLStore) ListWalletBlacklistEntries(ctx context.Context) ([]WalletBlacklistEntry, error) {
	if s.legacy != nil {
		rows, err := s.legacy.QueryContext(ctx, `
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
	if s.queries == nil {
		return nil, fmt.Errorf("wallet postgres database is not configured")
	}
	rows, err := s.queries.ListWalletBlacklistEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list wallet blacklist entries: %w", err)
	}

	items := make([]WalletBlacklistEntry, 0)
	for _, row := range rows {
		items = append(items, walletBlacklistEntryFromSQLC(row))
	}
	return items, nil
}

func (s *SQLStore) AddWalletBlacklistEntry(ctx context.Context, item WalletBlacklistEntry) error {
	if s.legacy != nil {
		_, err := s.legacy.ExecContext(ctx, `
INSERT INTO wallet_blacklist (wallet, note)
VALUES ($1, $2)
`, item.Wallet.Bytes(), legacyNullableTrimmedText(item.Note))
		if err != nil {
			if isUniqueViolation(err) {
				return ErrWalletBlacklistEntryAlreadyExists
			}
			return fmt.Errorf("add wallet blacklist entry: %w", err)
		}
		return nil
	}
	if s.queries == nil {
		return fmt.Errorf("wallet postgres database is not configured")
	}
	err := s.queries.AddWalletBlacklistEntry(ctx, walletsqlc.AddWalletBlacklistEntryParams{Wallet: item.Wallet.Bytes(), Note: nullableTrimmedText(item.Note)})
	if err != nil {
		if isUniqueViolation(err) {
			return ErrWalletBlacklistEntryAlreadyExists
		}
		return fmt.Errorf("add wallet blacklist entry: %w", err)
	}
	return nil
}

func (s *SQLStore) UpdateWalletBlacklistEntryNote(ctx context.Context, wallet common.Address, note string) error {
	if s.legacy != nil {
		result, err := s.legacy.ExecContext(ctx, `
UPDATE wallet_blacklist
SET note = $2
WHERE wallet = $1
`, wallet.Bytes(), legacyNullableTrimmedText(note))
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
	if s.queries == nil {
		return fmt.Errorf("wallet postgres database is not configured")
	}
	affected, err := s.queries.UpdateWalletBlacklistEntryNote(ctx, walletsqlc.UpdateWalletBlacklistEntryNoteParams{Wallet: wallet.Bytes(), Note: nullableTrimmedText(note)})
	if err != nil {
		return fmt.Errorf("update wallet blacklist entry note: %w", err)
	}
	if affected == 0 {
		return ErrWalletBlacklistEntryNotFound
	}
	return nil
}

func (s *SQLStore) DeleteWalletBlacklistEntry(ctx context.Context, wallet common.Address) error {
	if s.legacy != nil {
		result, err := s.legacy.ExecContext(ctx, `
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
	if s.queries == nil {
		return fmt.Errorf("wallet postgres database is not configured")
	}
	affected, err := s.queries.DeleteWalletBlacklistEntry(ctx, wallet.Bytes())
	if err != nil {
		return fmt.Errorf("delete wallet blacklist entry: %w", err)
	}
	if affected == 0 {
		return ErrWalletBlacklistEntryNotFound
	}
	return nil
}

func walletBlacklistEntryFromSQLC(row walletsqlc.WalletBlacklist) WalletBlacklistEntry {
	return WalletBlacklistEntry{
		Wallet:    common.BytesToAddress(row.Wallet),
		Note:      row.Note.String,
		CreatedAt: row.CreatedAt.Time,
	}
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

func nullableTrimmedText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func legacyNullableTrimmedText(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}
