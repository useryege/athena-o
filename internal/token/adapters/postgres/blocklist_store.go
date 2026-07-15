package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/policy"
)

func (s *Database) CreateContractCodeBlocklistEntry(ctx context.Context, item policy.ContractCodeBlocklistEntry) error {
	q, err := s.querier()
	if err != nil {
		return err
	}
	err = q.CreateContractCodeBlocklistEntry(ctx, tokensqlc.CreateContractCodeBlocklistEntryParams{CodeHash: item.CodeHash.Bytes(), Note: nullableText(item.Note), SourceChainID: nullableInt64(item.SourceChainID), SourceContract: optionalAddressBytes(item.SourceContract)})
	if err != nil {
		return fmt.Errorf("create contract code blocklist entry: %w", err)
	}
	return nil
}
func (s *Database) GetContractCodeBlocklistEntry(ctx context.Context, codeHash common.Hash) (*policy.ContractCodeBlocklistEntry, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	r, e := q.GetContractCodeBlocklistEntry(ctx, codeHash.Bytes())
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, fmt.Errorf("get contract code blocklist entry: %w", e)
	}
	return mapContractCodeBlocklistEntry(r), nil
}
func (s *Database) ListContractCodeBlocklistEntries(ctx context.Context) ([]policy.ContractCodeBlocklistEntry, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	r, e := q.ListContractCodeBlocklistEntries(ctx)
	if e != nil {
		return nil, fmt.Errorf("list contract code blocklist entries: %w", e)
	}
	return mapContractCodeBlocklistEntries(r), nil
}
func (s *Database) UpdateContractCodeBlocklistEntryNote(ctx context.Context, codeHash common.Hash, note string) (int64, error) {
	q, e := s.querier()
	if e != nil {
		return 0, e
	}
	n, e := q.UpdateContractCodeBlocklistEntryNote(ctx, tokensqlc.UpdateContractCodeBlocklistEntryNoteParams{CodeHash: codeHash.Bytes(), Note: nullableText(note)})
	if e != nil {
		return 0, fmt.Errorf("update contract code blocklist entry: %w", e)
	}
	return n, nil
}
func (s *Database) DeleteContractCodeBlocklistEntry(ctx context.Context, codeHash common.Hash) (int64, error) {
	q, e := s.querier()
	if e != nil {
		return 0, e
	}
	n, e := q.DeleteContractCodeBlocklistEntry(ctx, codeHash.Bytes())
	if e != nil {
		return 0, fmt.Errorf("delete contract code blocklist entry: %w", e)
	}
	return n, nil
}
func (s *Database) IsContractCodeBlocked(ctx context.Context, codeHash common.Hash) (bool, error) {
	q, e := s.querier()
	if e != nil {
		return false, e
	}
	v, e := q.IsContractCodeBlocked(ctx, codeHash.Bytes())
	if e != nil {
		return false, fmt.Errorf("check contract code blocklist: %w", e)
	}
	return v, nil
}

func (s *Database) CreateWalletBlocklistEntry(ctx context.Context, item policy.WalletBlocklistEntry) error {
	q, e := s.querier()
	if e != nil {
		return e
	}
	e = q.CreateWalletBlocklistEntry(ctx, tokensqlc.CreateWalletBlocklistEntryParams{Wallet: item.Wallet.Bytes(), Note: nullableText(item.Note)})
	if e != nil {
		return fmt.Errorf("create wallet blocklist entry: %w", e)
	}
	return nil
}
func (s *Database) GetWalletBlocklistEntry(ctx context.Context, wallet common.Address) (*policy.WalletBlocklistEntry, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	r, e := q.GetWalletBlocklistEntry(ctx, wallet.Bytes())
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, fmt.Errorf("get wallet blocklist entry: %w", e)
	}
	return mapWalletBlocklistEntry(r), nil
}
func (s *Database) ListWalletBlocklistEntries(ctx context.Context) ([]policy.WalletBlocklistEntry, error) {
	q, e := s.querier()
	if e != nil {
		return nil, e
	}
	r, e := q.ListWalletBlocklistEntries(ctx)
	if e != nil {
		return nil, fmt.Errorf("list wallet blocklist entries: %w", e)
	}
	return mapWalletBlocklistEntries(r), nil
}
func (s *Database) UpdateWalletBlocklistEntryNote(ctx context.Context, wallet common.Address, note string) (int64, error) {
	q, e := s.querier()
	if e != nil {
		return 0, e
	}
	n, e := q.UpdateWalletBlocklistEntryNote(ctx, tokensqlc.UpdateWalletBlocklistEntryNoteParams{Wallet: wallet.Bytes(), Note: nullableText(note)})
	if e != nil {
		return 0, fmt.Errorf("update wallet blocklist entry: %w", e)
	}
	return n, nil
}
func (s *Database) DeleteWalletBlocklistEntry(ctx context.Context, wallet common.Address) (int64, error) {
	q, e := s.querier()
	if e != nil {
		return 0, e
	}
	n, e := q.DeleteWalletBlocklistEntry(ctx, wallet.Bytes())
	if e != nil {
		return 0, fmt.Errorf("delete wallet blocklist entry: %w", e)
	}
	return n, nil
}
func (s *Database) IsWalletBlocked(ctx context.Context, wallet common.Address) (bool, error) {
	q, e := s.querier()
	if e != nil {
		return false, e
	}
	v, e := q.IsWalletBlocked(ctx, wallet.Bytes())
	if e != nil {
		return false, fmt.Errorf("check wallet blocklist: %w", e)
	}
	return v, nil
}
