package application

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appstore "github.com/useryege/athena/internal/application/store"
)

var _ appcache.WalletBlacklistWritePublisher = &walletBlacklistEventPublisher{}

type walletBlacklistEventPublisher struct {
	publisher PersistenceEventPublisher
}

func newWalletBlacklistEventPublisher(publisher PersistenceEventPublisher) appcache.WalletBlacklistWritePublisher {
	if publisher == nil {
		return nil
	}
	return &walletBlacklistEventPublisher{publisher: publisher}
}

func (p *walletBlacklistEventPublisher) PublishAdd(ctx context.Context, item appstore.WalletBlacklistEntry) error {
	return p.publisher.PublishWalletBlacklistAdd(ctx, item)
}

func (p *walletBlacklistEventPublisher) PublishUpdateNote(ctx context.Context, wallet common.Address, note string) error {
	return p.publisher.PublishWalletBlacklistUpdateNote(ctx, wallet, note)
}

func (p *walletBlacklistEventPublisher) PublishDelete(ctx context.Context, wallet common.Address) error {
	return p.publisher.PublishWalletBlacklistDelete(ctx, wallet)
}
