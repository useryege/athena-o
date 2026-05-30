package components

import (
	"context"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appstore "github.com/useryege/athena/internal/application/store"
	solidityapiclient "github.com/useryege/athena/internal/solidity/apiclient"
	utilio "github.com/useryege/athena/util/io"
)

type BytecodeComponent struct {
	store    appstore.Store
	cache    appcache.ProjectComponentCache
	clients  solidityapiclient.Clientset
	chainID  int64
	bus      EventBus
	consumer *Consumer
}

func NewBytecodeComponent(store appstore.Store, cache appcache.ProjectComponentCache, clients solidityapiclient.Clientset, chainID int64, bus EventBus) *BytecodeComponent {
	if store == nil || clients == nil || chainID <= 0 {
		return nil
	}
	c := &BytecodeComponent{store: store, cache: cache, clients: clients, chainID: chainID, bus: bus}
	c.consumer = NewConsumer(appstore.ProjectComponentBytecodeFact, bus, c.handleEvent)
	return c
}

func (c *BytecodeComponent) Start(ctx context.Context) error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Start(ctx)
}

func (c *BytecodeComponent) Stop() error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Stop()
}

func (c *BytecodeComponent) handleEvent(ctx context.Context, event Event) error {
	if event.Type != EventProjectInitialized && event.Type != EventProjectRefresh {
		return nil
	}
	contract := event.ProjectContract()
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.refresh(ctx, contract); err != nil {
		_ = MarkComponentFailed(ctx, c.store, contract, appstore.ProjectComponentBytecodeFact, err, nowUTC())
		if c.bus != nil {
			_ = c.bus.Publish(ctx, ComponentFailedEvent(contract, appstore.ProjectComponentBytecodeFact, err))
		}
		return nil
	}
	return nil
}

func (c *BytecodeComponent) refresh(ctx context.Context, contract common.Address) error {
	if _, err := LoadProjectBase(ctx, c.cache, c.store, contract); err != nil {
		return err
	}
	if err := MarkComponentRunning(ctx, c.store, contract, appstore.ProjectComponentBytecodeFact, nowUTC()); err != nil {
		return err
	}
	closer, client, err := c.clients.NewSolidityServiceClient()
	if err != nil {
		return err
	}
	defer utilio.Close(closer)
	info, err := client.GetContractSourceInfo(ctx, &solidityapiclient.GetContractSourceInfoRequest{
		ChainId:  c.chainID,
		Contract: contract.Hex(),
	})
	if err != nil || info == nil {
		return err
	}
	item := appstore.ProjectBytecodeFact{
		ProjectContract:       contract,
		IsBytecodeBlacklisted: info.IsBytecodeBlacklisted,
		FetchedAt:             nowUTC(),
	}
	if strings.TrimSpace(info.CodeBinHash) != "" {
		item.CodeHash = common.HexToHash(info.CodeBinHash)
	}
	if err := c.store.UpsertProjectBytecodeFact(ctx, item); err != nil {
		return err
	}
	if c.cache != nil {
		if err := c.cache.SetBytecodeFact(ctx, item); err != nil {
			return err
		}
	}
	if err := MarkComponentSuccess(ctx, c.store, contract, appstore.ProjectComponentBytecodeFact, item.FetchedAt); err != nil {
		return err
	}
	if c.bus != nil {
		return c.bus.Publish(ctx, ComponentCompletedEvent(contract, appstore.ProjectComponentBytecodeFact))
	}
	return nil
}
