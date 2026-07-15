package application

import (
	"context"

	"github.com/useryege/athena/internal/token/policy"
	"github.com/useryege/athena/internal/token/shared"
)

type Service struct {
	repository Repository
	codeHash   ContractCodeHashProvider
}

type Repository interface {
	GetContractCodeBlocklistEntry(context.Context, shared.Hash) (*policy.ContractCodeBlocklistEntry, error)
	ListContractCodeBlocklistEntries(context.Context) ([]policy.ContractCodeBlocklistEntry, error)
	CreateContractCodeBlocklistEntry(context.Context, policy.ContractCodeBlocklistEntry) error
	UpdateContractCodeBlocklistEntryNote(context.Context, shared.Hash, string) (int64, error)
	DeleteContractCodeBlocklistEntry(context.Context, shared.Hash) (int64, error)
	GetWalletBlocklistEntry(context.Context, shared.Address) (*policy.WalletBlocklistEntry, error)
	ListWalletBlocklistEntries(context.Context) ([]policy.WalletBlocklistEntry, error)
	CreateWalletBlocklistEntry(context.Context, policy.WalletBlocklistEntry) error
	UpdateWalletBlocklistEntryNote(context.Context, shared.Address, string) (int64, error)
	DeleteWalletBlocklistEntry(context.Context, shared.Address) (int64, error)
}

type ContractCodeHashProvider interface {
	ContractCodeHash(context.Context, int64, shared.Address) (shared.Hash, error)
}

func NewService(repository Repository, codeHash ContractCodeHashProvider) *Service {
	return &Service{repository: repository, codeHash: codeHash}
}

func (s *Service) GetContractCodeBlocklistEntry(ctx context.Context, hash shared.Hash) (*policy.ContractCodeBlocklistEntry, error) {
	return s.repository.GetContractCodeBlocklistEntry(ctx, hash)
}
func (s *Service) ListContractCodeBlocklistEntries(ctx context.Context) ([]policy.ContractCodeBlocklistEntry, error) {
	return s.repository.ListContractCodeBlocklistEntries(ctx)
}
func (s *Service) CreateContractCodeBlocklistEntry(ctx context.Context, chainID int64, contract shared.Address, note string) error {
	hash, err := s.codeHash.ContractCodeHash(ctx, chainID, contract)
	if err != nil {
		return err
	}
	return s.repository.CreateContractCodeBlocklistEntry(ctx, policy.ContractCodeBlocklistEntry{CodeHash: hash, Note: note, SourceChainID: chainID, SourceContract: contract})
}
func (s *Service) UpdateContractCodeBlocklistEntryNote(ctx context.Context, hash shared.Hash, note string) (int64, error) {
	return s.repository.UpdateContractCodeBlocklistEntryNote(ctx, hash, note)
}
func (s *Service) DeleteContractCodeBlocklistEntry(ctx context.Context, hash shared.Hash) (int64, error) {
	return s.repository.DeleteContractCodeBlocklistEntry(ctx, hash)
}
func (s *Service) GetWalletBlocklistEntry(ctx context.Context, wallet shared.Address) (*policy.WalletBlocklistEntry, error) {
	return s.repository.GetWalletBlocklistEntry(ctx, wallet)
}
func (s *Service) ListWalletBlocklistEntries(ctx context.Context) ([]policy.WalletBlocklistEntry, error) {
	return s.repository.ListWalletBlocklistEntries(ctx)
}
func (s *Service) CreateWalletBlocklistEntry(ctx context.Context, item policy.WalletBlocklistEntry) error {
	return s.repository.CreateWalletBlocklistEntry(ctx, item)
}
func (s *Service) UpdateWalletBlocklistEntryNote(ctx context.Context, wallet shared.Address, note string) (int64, error) {
	return s.repository.UpdateWalletBlocklistEntryNote(ctx, wallet, note)
}
func (s *Service) DeleteWalletBlocklistEntry(ctx context.Context, wallet shared.Address) (int64, error) {
	return s.repository.DeleteWalletBlocklistEntry(ctx, wallet)
}
