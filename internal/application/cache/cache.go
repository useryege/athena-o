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
}

type SourceCodeBlacklistRemoteCache interface {
	Get(ctx context.Context) ([]string, bool, error)
	Set(ctx context.Context, fields []string) error
	Del(ctx context.Context) error
}

type SourceCodeBlacklistModel interface {
	Load(ctx context.Context) error
	List(ctx context.Context) ([]string, error)
	Add(ctx context.Context, field string) error
	Delete(ctx context.Context, field string) error
}

type SourceCodeBlacklistWritePublisher interface {
	PublishAdd(ctx context.Context, field string) error
	PublishDelete(ctx context.Context, field string) error
}

type BytecodeBlacklistCache interface {
	Take(ctx context.Context, loader func(context.Context) ([]store.BytecodeBlacklistContract, error)) ([]store.BytecodeBlacklistContract, error)
	Set(ctx context.Context, items []store.BytecodeBlacklistContract) error
	Del(ctx context.Context) error
}

type BytecodeBlacklistRemoteCache interface {
	Get(ctx context.Context) ([]store.BytecodeBlacklistContract, bool, error)
	Set(ctx context.Context, items []store.BytecodeBlacklistContract) error
	Del(ctx context.Context) error
}

type BytecodeBlacklistModel interface {
	Load(ctx context.Context) error
	List(ctx context.Context) ([]store.BytecodeBlacklistContract, error)
	Add(ctx context.Context, item store.BytecodeBlacklistContract) error
	UpdateNote(ctx context.Context, contract common.Address, note string) error
	Delete(ctx context.Context, contract common.Address) error
}

type BytecodeBlacklistWritePublisher interface {
	PublishAdd(ctx context.Context, item store.BytecodeBlacklistContract) error
	PublishUpdateNote(ctx context.Context, contract common.Address, note string) error
	PublishDelete(ctx context.Context, contract common.Address) error
}

type WalletBlacklistCache interface {
	Take(ctx context.Context, loader func(context.Context) ([]store.WalletBlacklistContract, error)) ([]store.WalletBlacklistContract, error)
	Set(ctx context.Context, items []store.WalletBlacklistContract) error
	Del(ctx context.Context) error
}

type WalletBlacklistRemoteCache interface {
	Get(ctx context.Context) ([]store.WalletBlacklistContract, bool, error)
	Set(ctx context.Context, items []store.WalletBlacklistContract) error
	Del(ctx context.Context) error
}

type WalletBlacklistModel interface {
	Load(ctx context.Context) error
	List(ctx context.Context) ([]store.WalletBlacklistContract, error)
	Add(ctx context.Context, item store.WalletBlacklistContract) error
	UpdateNote(ctx context.Context, contract common.Address, note string) error
	Delete(ctx context.Context, contract common.Address) error
}

type WalletBlacklistWritePublisher interface {
	PublishAdd(ctx context.Context, item store.WalletBlacklistContract) error
	PublishUpdateNote(ctx context.Context, contract common.Address, note string) error
	PublishDelete(ctx context.Context, contract common.Address) error
}
