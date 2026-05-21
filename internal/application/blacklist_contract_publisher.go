package application

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appstore "github.com/useryege/athena/internal/application/store"
)

var _ appcache.BytecodeBlacklistWritePublisher = &bytecodeBlacklistEventPublisher{}
var _ appcache.WalletBlacklistWritePublisher = &walletBlacklistEventPublisher{}

type bytecodeBlacklistEventPublisher struct {
	publisher PersistenceEventPublisher
}

func newBytecodeBlacklistEventPublisher(publisher PersistenceEventPublisher) appcache.BytecodeBlacklistWritePublisher {
	if publisher == nil {
		return nil
	}
	return &bytecodeBlacklistEventPublisher{publisher: publisher}
}

func (p *bytecodeBlacklistEventPublisher) PublishAdd(ctx context.Context, item appstore.BytecodeBlacklistContract) error {
	return p.publisher.PublishBytecodeBlacklistAdd(ctx, item)
}

func (p *bytecodeBlacklistEventPublisher) PublishUpdateNote(ctx context.Context, contract common.Address, note string) error {
	return p.publisher.PublishBytecodeBlacklistUpdateNote(ctx, contract, note)
}

func (p *bytecodeBlacklistEventPublisher) PublishDelete(ctx context.Context, contract common.Address) error {
	return p.publisher.PublishBytecodeBlacklistDelete(ctx, contract)
}

type walletBlacklistEventPublisher struct {
	publisher PersistenceEventPublisher
}

func newWalletBlacklistEventPublisher(publisher PersistenceEventPublisher) appcache.WalletBlacklistWritePublisher {
	if publisher == nil {
		return nil
	}
	return &walletBlacklistEventPublisher{publisher: publisher}
}

func (p *walletBlacklistEventPublisher) PublishAdd(ctx context.Context, item appstore.WalletBlacklistContract) error {
	return p.publisher.PublishWalletBlacklistAdd(ctx, item)
}

func (p *walletBlacklistEventPublisher) PublishUpdateNote(ctx context.Context, contract common.Address, note string) error {
	return p.publisher.PublishWalletBlacklistUpdateNote(ctx, contract, note)
}

func (p *walletBlacklistEventPublisher) PublishDelete(ctx context.Context, contract common.Address) error {
	return p.publisher.PublishWalletBlacklistDelete(ctx, contract)
}
