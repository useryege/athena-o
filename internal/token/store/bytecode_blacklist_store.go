package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

func (s *SQLStore) AddBytecodeBlacklistEntry(ctx context.Context, item BytecodeBlacklistEntry) error {
	q, err := s.querier()
	if err != nil {
		return err
	}
	if err := q.AddBytecodeBlacklistEntry(ctx, tokensqlc.AddBytecodeBlacklistEntryParams{
		CodeHash:       item.CodeHash.Bytes(),
		Note:           nullableText(item.Note),
		SourceChainID:  nullableInt64(item.SourceChainID),
		SourceContract: optionalAddressBytes(item.SourceContract),
	}); err != nil {
		return fmt.Errorf("add bytecode blacklist entry: %w", err)
	}
	return nil
}

func (s *SQLStore) GetBytecodeBlacklistEntry(ctx context.Context, codeHash common.Hash) (*BytecodeBlacklistEntry, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := q.GetBytecodeBlacklistEntry(ctx, codeHash.Bytes())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get bytecode blacklist entry: %w", err)
	}
	return mapBytecodeBlacklistEntry(row), nil
}

func (s *SQLStore) ListBytecodeBlacklistEntries(ctx context.Context) ([]BytecodeBlacklistEntry, error) {
	q, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := q.ListBytecodeBlacklistEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list bytecode blacklist entries: %w", err)
	}
	return mapBytecodeBlacklistEntries(rows), nil
}

func (s *SQLStore) UpdateBytecodeBlacklistNote(ctx context.Context, codeHash common.Hash, note string) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.UpdateBytecodeBlacklistNote(ctx, tokensqlc.UpdateBytecodeBlacklistNoteParams{
		CodeHash: codeHash.Bytes(),
		Note:     nullableText(note),
	})
	if err != nil {
		return 0, fmt.Errorf("update bytecode blacklist note: %w", err)
	}
	return rowsAffected, nil
}

func (s *SQLStore) DeleteBytecodeBlacklistEntry(ctx context.Context, codeHash common.Hash) (int64, error) {
	q, err := s.querier()
	if err != nil {
		return 0, err
	}
	rowsAffected, err := q.DeleteBytecodeBlacklistEntry(ctx, codeHash.Bytes())
	if err != nil {
		return 0, fmt.Errorf("delete bytecode blacklist entry: %w", err)
	}
	return rowsAffected, nil
}

func (s *SQLStore) IsBytecodeBlacklisted(ctx context.Context, codeHash common.Hash) (bool, error) {
	q, err := s.querier()
	if err != nil {
		return false, err
	}
	blacklisted, err := q.IsBytecodeBlacklisted(ctx, codeHash.Bytes())
	if err != nil {
		return false, fmt.Errorf("is bytecode blacklisted: %w", err)
	}
	return blacklisted, nil
}
