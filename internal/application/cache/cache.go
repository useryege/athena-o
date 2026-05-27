package cache

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/store"
)

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
