package application

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/token/policy"
)

type Service struct {
	repository policy.Repository
	codeHash   policy.ContractCodeHashProvider
}

func NewService(repository policy.Repository, codeHash policy.ContractCodeHashProvider) *Service {
	return &Service{repository: repository, codeHash: codeHash}
}

func (s *Service) GetContractCodeBlocklistEntry(ctx context.Context, hash common.Hash) (*policy.ContractCodeBlocklistEntry, error) {
	return s.repository.GetContractCodeBlocklistEntry(ctx, hash)
}
func (s *Service) ListContractCodeBlocklistEntries(ctx context.Context) ([]policy.ContractCodeBlocklistEntry, error) {
	return s.repository.ListContractCodeBlocklistEntries(ctx)
}
func (s *Service) CreateContractCodeBlocklistEntry(ctx context.Context, chainID int64, contract common.Address, note string) error {
	hash, err := s.codeHash.ContractCodeHash(ctx, chainID, contract)
	if err != nil {
		return err
	}
	return s.repository.CreateContractCodeBlocklistEntry(ctx, policy.ContractCodeBlocklistEntry{CodeHash: hash, Note: note, SourceChainID: chainID, SourceContract: contract})
}
func (s *Service) UpdateContractCodeBlocklistEntryNote(ctx context.Context, hash common.Hash, note string) (int64, error) {
	return s.repository.UpdateContractCodeBlocklistEntryNote(ctx, hash, note)
}
func (s *Service) DeleteContractCodeBlocklistEntry(ctx context.Context, hash common.Hash) (int64, error) {
	return s.repository.DeleteContractCodeBlocklistEntry(ctx, hash)
}
func (s *Service) GetWalletBlocklistEntry(ctx context.Context, wallet common.Address) (*policy.WalletBlocklistEntry, error) {
	return s.repository.GetWalletBlocklistEntry(ctx, wallet)
}
func (s *Service) ListWalletBlocklistEntries(ctx context.Context) ([]policy.WalletBlocklistEntry, error) {
	return s.repository.ListWalletBlocklistEntries(ctx)
}
func (s *Service) CreateWalletBlocklistEntry(ctx context.Context, item policy.WalletBlocklistEntry) error {
	return s.repository.CreateWalletBlocklistEntry(ctx, item)
}
func (s *Service) UpdateWalletBlocklistEntryNote(ctx context.Context, wallet common.Address, note string) (int64, error) {
	return s.repository.UpdateWalletBlocklistEntryNote(ctx, wallet, note)
}
func (s *Service) DeleteWalletBlocklistEntry(ctx context.Context, wallet common.Address) (int64, error) {
	return s.repository.DeleteWalletBlocklistEntry(ctx, wallet)
}
