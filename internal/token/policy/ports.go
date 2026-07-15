package policy

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
)

type Repository interface {
	GetContractCodeBlocklistEntry(context.Context, common.Hash) (*ContractCodeBlocklistEntry, error)
	ListContractCodeBlocklistEntries(context.Context) ([]ContractCodeBlocklistEntry, error)
	CreateContractCodeBlocklistEntry(context.Context, ContractCodeBlocklistEntry) error
	UpdateContractCodeBlocklistEntryNote(context.Context, common.Hash, string) (int64, error)
	DeleteContractCodeBlocklistEntry(context.Context, common.Hash) (int64, error)
	GetWalletBlocklistEntry(context.Context, common.Address) (*WalletBlocklistEntry, error)
	ListWalletBlocklistEntries(context.Context) ([]WalletBlocklistEntry, error)
	CreateWalletBlocklistEntry(context.Context, WalletBlocklistEntry) error
	UpdateWalletBlocklistEntryNote(context.Context, common.Address, string) (int64, error)
	DeleteWalletBlocklistEntry(context.Context, common.Address) (int64, error)
}

type ContractCodeHashProvider interface {
	ContractCodeHash(context.Context, int64, common.Address) (common.Hash, error)
}
