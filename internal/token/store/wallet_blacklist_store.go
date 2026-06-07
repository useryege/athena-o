package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) AddWalletBlacklistEntry(ctx context.Context, item WalletBlacklistEntry) error {
	q, err := s.querier()
	if err != nil {
		return err
	}
	if err := q.AddWalletBlacklistEntry(ctx, tokensqlc.AddWalletBlacklistEntryParams{
		Wallet: item.Wallet.Bytes(),
		Note:   nullableText(item.Note),
	}); err != nil {
		return fmt.Errorf("add wallet blacklist entry: %w", err)
	}
	return nil
}

func (s *SQLStore) GetWalletBlacklistEntry(ctx context.Context, wallet common.Address) (*WalletBlacklistEntry, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetWalletBlacklistEntry(ctx, wallet.Bytes())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get wallet blacklist entry: %w", err)
	}
	return mapWalletBlacklistEntry(row), nil
}

func (s *SQLStore) ListWalletBlacklistEntries(ctx context.Context) ([]WalletBlacklistEntry, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListWalletBlacklistEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list wallet blacklist entries: %w", err)
	}
	return mapWalletBlacklistEntries(rows), nil
}

func (s *SQLStore) UpdateWalletBlacklistNote(ctx context.Context, wallet common.Address, note string) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.UpdateWalletBlacklistNote(ctx, tokensqlc.UpdateWalletBlacklistNoteParams{
		Wallet: wallet.Bytes(),
		Note:   nullableText(note),
	})
	if err != nil {
		return 0, fmt.Errorf("update wallet blacklist note: %w", err)
	}
	return rowsAffected, nil
}

func (s *SQLStore) DeleteWalletBlacklistEntry(ctx context.Context, wallet common.Address) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteWalletBlacklistEntry(ctx, wallet.Bytes())
	if err != nil {
		return 0, fmt.Errorf("delete wallet blacklist entry: %w", err)
	}
	return rowsAffected, nil
}

func (s *SQLStore) IsWalletBlacklisted(ctx context.Context, wallet common.Address) (bool, error) {
	q, err := s.querier()
	if err != nil {
		return false, err
	}
	blacklisted, err := q.IsWalletBlacklisted(ctx, wallet.Bytes())
	if err != nil {
		return false, fmt.Errorf("is wallet blacklisted: %w", err)
	}
	return blacklisted, nil
}
