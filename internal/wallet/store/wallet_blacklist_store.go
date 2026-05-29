package store

import (
	"context"
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

func nullableTrimmedText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}
