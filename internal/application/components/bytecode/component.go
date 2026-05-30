package bytecode

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	appstore "github.com/useryege/athena/internal/application/store"
	solidityapiclient "github.com/useryege/athena/internal/solidity/apiclient"
	utilio "github.com/useryege/athena/util/io"
)

type Options struct {
	Store   appstore.Store
	Cache   appcache.ProjectComponentCache
	Clients solidityapiclient.Clientset
	ChainID int64
	Bus     appcomponents.EventBus
}

type Component struct {
	store    appstore.Store
	cache    appcache.ProjectComponentCache
	clients  solidityapiclient.Clientset
	chainID  int64
	bus      appcomponents.EventBus
	consumer *appcomponents.Consumer
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil || opts.Clients == nil || opts.ChainID <= 0 {
		return nil
	}
	c := &Component{store: opts.Store, cache: opts.Cache, clients: opts.Clients, chainID: opts.ChainID, bus: opts.Bus}
	c.consumer = appcomponents.NewConsumer(appstore.ProjectComponentBytecodeFact, opts.Bus, c.handleEvent)
	return c
}

func (c *Component) Start(ctx context.Context) error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Start(ctx)
}

func (c *Component) Stop() error {
	if c == nil || c.consumer == nil {
		return nil
	}
	return c.consumer.Stop()
}

func (c *Component) handleEvent(ctx context.Context, event appcomponents.Event) error {
	if event.Type != appcomponents.EventProjectInitialized && event.Type != appcomponents.EventProjectRefresh {
		return nil
	}
	contract := event.ProjectContract()
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.refresh(ctx, contract); err != nil {
		_ = appcomponents.MarkComponentFailed(ctx, c.store, contract, appstore.ProjectComponentBytecodeFact, err, nowUTC())
		if c.bus != nil {
			_ = c.bus.Publish(ctx, appcomponents.ComponentFailedEvent(contract, appstore.ProjectComponentBytecodeFact, err))
		}
		return nil
	}
	return nil
}

func (c *Component) refresh(ctx context.Context, contract common.Address) error {
	if _, err := appcomponents.LoadProjectBase(ctx, c.cache, c.store, contract); err != nil {
		return err
	}
	if err := appcomponents.MarkComponentRunning(ctx, c.store, contract, appstore.ProjectComponentBytecodeFact, nowUTC()); err != nil {
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
	if err := appcomponents.MarkComponentSuccess(ctx, c.store, contract, appstore.ProjectComponentBytecodeFact, item.FetchedAt); err != nil {
		return err
	}
	if c.bus != nil {
		return c.bus.Publish(ctx, appcomponents.ComponentCompletedEvent(contract, appstore.ProjectComponentBytecodeFact))
	}
	return nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
