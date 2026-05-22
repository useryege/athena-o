package cache

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/store"
)

type SourceCodeBlacklistCache interface {
	Take(ctx context.Context, loader func(context.Context) ([]string, error)) ([]string, error)
	Set(ctx context.Context, fields []string) error
	Del(ctx context.Context) error
	Version(ctx context.Context) (string, error)
}

type SourceCodeBlacklistRemoteCache interface {
	Get(ctx context.Context) ([]string, string, bool, error)
	Set(ctx context.Context, fields []string) (string, error)
	Del(ctx context.Context) error
	Version(ctx context.Context) (string, bool, error)
}

type SourceCodeBlacklistModel interface {
	Load(ctx context.Context) error
	List(ctx context.Context) ([]string, error)
	Add(ctx context.Context, field string) error
	Delete(ctx context.Context, field string) error
	Version(ctx context.Context) (string, error)
}

type SourceCodeBlacklistWritePublisher interface {
	PublishAdd(ctx context.Context, field string) error
	PublishDelete(ctx context.Context, field string) error
}

type BytecodeBlacklistCache interface {
	Take(ctx context.Context, loader func(context.Context) ([]store.BytecodeBlacklistContract, error)) ([]store.BytecodeBlacklistContract, error)
	Set(ctx context.Context, items []store.BytecodeBlacklistContract) error
	Del(ctx context.Context) error
	Version(ctx context.Context) (string, error)
}

type BytecodeBlacklistRemoteCache interface {
	Get(ctx context.Context) ([]store.BytecodeBlacklistContract, string, bool, error)
	Set(ctx context.Context, items []store.BytecodeBlacklistContract) (string, error)
	Del(ctx context.Context) error
	Version(ctx context.Context) (string, bool, error)
}

type BytecodeBlacklistModel interface {
	Load(ctx context.Context) error
	List(ctx context.Context) ([]store.BytecodeBlacklistContract, error)
	Add(ctx context.Context, item store.BytecodeBlacklistContract) error
	UpdateNote(ctx context.Context, contract common.Address, note string) error
	Delete(ctx context.Context, contract common.Address) error
	Version(ctx context.Context) (string, error)
}

type BytecodeBlacklistWritePublisher interface {
	PublishAdd(ctx context.Context, item store.BytecodeBlacklistContract) error
	PublishUpdateNote(ctx context.Context, contract common.Address, note string) error
	PublishDelete(ctx context.Context, contract common.Address) error
}

type SourcecodeBlacklistContractCache interface {
	Take(ctx context.Context, loader func(context.Context) ([]store.SourcecodeBlacklistContract, error)) ([]store.SourcecodeBlacklistContract, error)
	Set(ctx context.Context, items []store.SourcecodeBlacklistContract) error
	Del(ctx context.Context) error
	Version(ctx context.Context) (string, error)
}

type SourcecodeBlacklistContractRemoteCache interface {
	Get(ctx context.Context) ([]store.SourcecodeBlacklistContract, string, bool, error)
	Set(ctx context.Context, items []store.SourcecodeBlacklistContract) (string, error)
	Del(ctx context.Context) error
	Version(ctx context.Context) (string, bool, error)
}

type SourcecodeBlacklistContractModel interface {
	Load(ctx context.Context) error
	List(ctx context.Context) ([]store.SourcecodeBlacklistContract, error)
	Add(ctx context.Context, item store.SourcecodeBlacklistContract) error
	UpdateNote(ctx context.Context, contract common.Address, note string) error
	Delete(ctx context.Context, contract common.Address) error
	Version(ctx context.Context) (string, error)
}

type SourcecodeBlacklistContractWritePublisher interface {
	PublishAdd(ctx context.Context, item store.SourcecodeBlacklistContract) error
	PublishUpdateNote(ctx context.Context, contract common.Address, note string) error
	PublishDelete(ctx context.Context, contract common.Address) error
}

type WalletBlacklistCache interface {
	Take(ctx context.Context, loader func(context.Context) ([]store.WalletBlacklistEntry, error)) ([]store.WalletBlacklistEntry, error)
	Set(ctx context.Context, items []store.WalletBlacklistEntry) error
	Del(ctx context.Context) error
	Version(ctx context.Context) (string, error)
}

type WalletBlacklistRemoteCache interface {
	Get(ctx context.Context) ([]store.WalletBlacklistEntry, string, bool, error)
	Set(ctx context.Context, items []store.WalletBlacklistEntry) (string, error)
	Del(ctx context.Context) error
	Version(ctx context.Context) (string, bool, error)
}

type WalletBlacklistModel interface {
	Load(ctx context.Context) error
	List(ctx context.Context) ([]store.WalletBlacklistEntry, error)
	Add(ctx context.Context, item store.WalletBlacklistEntry) error
	UpdateNote(ctx context.Context, wallet common.Address, note string) error
	Delete(ctx context.Context, wallet common.Address) error
	Version(ctx context.Context) (string, error)
}

type WalletBlacklistWritePublisher interface {
	PublishAdd(ctx context.Context, item store.WalletBlacklistEntry) error
	PublishUpdateNote(ctx context.Context, wallet common.Address, note string) error
	PublishDelete(ctx context.Context, wallet common.Address) error
}
