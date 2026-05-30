package report

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	"github.com/useryege/athena/internal/application/model"
	"github.com/useryege/athena/internal/application/persistence"
	appstore "github.com/useryege/athena/internal/application/store"
)

type WalletBlacklistLister interface {
	ListWalletBlacklist(ctx context.Context) ([]common.Address, error)
}

type Options struct {
	Store              appstore.Store
	Cache              appcache.ProjectComponentCache
	Bus                appcomponents.EventBus
	WalletBlacklist    WalletBlacklistLister
	RequiredComponents []string
}

type Component struct {
	store              appstore.Store
	cache              appcache.ProjectComponentCache
	bus                appcomponents.EventBus
	walletBlacklist    WalletBlacklistLister
	requiredComponents []string
	consumer           *appcomponents.Consumer
}

func NewComponent(opts Options) *Component {
	if opts.Store == nil {
		return nil
	}
	c := &Component{
		store:              opts.Store,
		cache:              opts.Cache,
		bus:                opts.Bus,
		walletBlacklist:    opts.WalletBlacklist,
		requiredComponents: opts.RequiredComponents,
	}
	c.consumer = appcomponents.NewConsumer(appstore.ProjectComponentReport, opts.Bus, c.handleEvent)
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
	if event.Type != appcomponents.EventComponentCompleted {
		return nil
	}
	if event.Component == "" || event.Component == appstore.ProjectComponentReport {
		return nil
	}
	contract := event.ProjectContract()
	if contract == (common.Address{}) {
		return nil
	}
	if err := c.Evaluate(ctx, contract); err != nil {
		_ = appcomponents.MarkComponentFailed(ctx, c.store, contract, appstore.ProjectComponentReport, err, nowUTC())
		if c.bus != nil {
			_ = c.bus.Publish(ctx, appcomponents.ComponentFailedEvent(contract, appstore.ProjectComponentReport, err))
		}
		return nil
	}
	return nil
}

func (c *Component) Evaluate(ctx context.Context, contract common.Address) error {
	if c == nil || contract == (common.Address{}) {
		return nil
	}
	base, err := appcomponents.LoadProjectBase(ctx, c.cache, c.store, contract)
	if err != nil || base == nil {
		return err
	}
	walletBlacklist, err := c.walletBlacklistSet(ctx)
	if err != nil {
		return err
	}
	genesisWallets, err := c.projectGenesisWallets(ctx, contract)
	if err != nil {
		return err
	}
	simulation, err := c.projectSimulation(ctx, contract)
	if err != nil {
		return err
	}
	bytecodeFact, err := c.projectBytecodeFact(ctx, contract)
	if err != nil {
		return err
	}
	report := appstore.ProjectReport{
		IsReportEvaluated: true,
		IsReportComplete:  c.reportComplete(ctx, contract),
	}
	type match struct {
		rule     string
		evidence map[string]any
	}
	matches := make([]match, 0, 4)
	if _, ok := walletBlacklist[base.Creator]; ok {
		report.IsBlacklistedCreatorWallet = true
		matches = append(matches, match{rule: "wallet_blacklist_creator", evidence: map[string]any{"creator_wallet": strings.ToLower(base.Creator.Hex())}})
	}
	matchedGenesisWallets := make([]string, 0)
	seenGenesisWallets := map[common.Address]struct{}{}
	for _, wallet := range genesisWallets {
		if _, ok := walletBlacklist[wallet]; !ok {
			continue
		}
		if _, ok := seenGenesisWallets[wallet]; ok {
			continue
		}
		seenGenesisWallets[wallet] = struct{}{}
		matchedGenesisWallets = append(matchedGenesisWallets, strings.ToLower(wallet.Hex()))
	}
	if len(matchedGenesisWallets) > 0 {
		report.IsBlacklistedGenesisWallet = true
		matches = append(matches, match{rule: "wallet_blacklist_genesis_wallet", evidence: map[string]any{"blacklisted_genesis_wallets": matchedGenesisWallets}})
	}
	if bytecodeFact != nil && bytecodeFact.IsBytecodeBlacklisted {
		report.IsBlacklistedBytecode = true
		matches = append(matches, match{rule: "bytecode_blacklist", evidence: map[string]any{"contract": strings.ToLower(contract.Hex())}})
	}
	if simulationResult := simulateResultFromStore(simulation); simulationResult.HasMintRisk() {
		report.HasMintRisk = true
		matches = append(matches, match{rule: "simulate_result_mint_risk", evidence: map[string]any{"mintable_paths": simulationResult.MintablePaths()}})
	}
	now := nowUTC()
	state := appstore.ProjectReportState{ProjectContract: contract, Report: report, EvaluatedAt: now}
	if err := c.store.UpsertProjectReportState(ctx, state); err != nil {
		return err
	}
	if c.cache != nil {
		if err := c.cache.SetReport(ctx, state); err != nil {
			return err
		}
	}
	for _, item := range matches {
		event, err := persistence.NewProjectReportMatchedEvent(contract, item.rule, item.evidence, now)
		if err != nil {
			continue
		}
		_ = c.store.AddProjectEventLog(ctx, event)
	}
	if err := appcomponents.MarkComponentSuccess(ctx, c.store, contract, appstore.ProjectComponentReport, now); err != nil {
		return err
	}
	if c.bus != nil {
		return c.bus.Publish(ctx, appcomponents.ComponentCompletedEvent(contract, appstore.ProjectComponentReport))
	}
	return nil
}

func (c *Component) walletBlacklistSet(ctx context.Context) (map[common.Address]struct{}, error) {
	result := map[common.Address]struct{}{}
	if c.walletBlacklist == nil {
		return result, nil
	}
	wallets, err := c.walletBlacklist.ListWalletBlacklist(ctx)
	if err != nil {
		return nil, err
	}
	for _, wallet := range wallets {
		result[wallet] = struct{}{}
	}
	return result, nil
}

func (c *Component) projectGenesisWallets(ctx context.Context, contract common.Address) ([]common.Address, error) {
	if c.cache != nil {
		if items, ok, err := c.cache.GetGenesisWallets(ctx, contract); err != nil {
			return nil, err
		} else if ok {
			return appcomponents.GenesisWalletAddresses(items), nil
		}
	}
	items, err := c.store.ListProjectGenesisWalletsByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	if c.cache != nil {
		if err := c.cache.SetGenesisWallets(ctx, contract, items); err != nil {
			return nil, err
		}
	}
	return appcomponents.GenesisWalletAddresses(items), nil
}

func (c *Component) projectSimulation(ctx context.Context, contract common.Address) (*appstore.ProjectSimulationResult, error) {
	if c.cache != nil {
		if item, ok, err := c.cache.GetSimulation(ctx, contract); err != nil {
			return nil, err
		} else if ok && item != nil {
			return item, nil
		}
	}
	item, err := c.store.GetProjectSimulationResult(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	if c.cache != nil {
		if err := c.cache.SetSimulation(ctx, *item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (c *Component) projectBytecodeFact(ctx context.Context, contract common.Address) (*appstore.ProjectBytecodeFact, error) {
	if c.cache != nil {
		if item, ok, err := c.cache.GetBytecodeFact(ctx, contract); err != nil {
			return nil, err
		} else if ok && item != nil {
			return item, nil
		}
	}
	item, err := c.store.GetProjectBytecodeFact(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	if c.cache != nil {
		if err := c.cache.SetBytecodeFact(ctx, *item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (c *Component) reportComplete(ctx context.Context, contract common.Address) bool {
	for _, component := range c.requiredComponents {
		if component == "" || component == appstore.ProjectComponentReport {
			continue
		}
		ok, err := appcomponents.ComponentSucceeded(ctx, c.store, contract, component)
		if err != nil || !ok {
			return false
		}
	}
	return true
}

func simulateResultFromStore(item *appstore.ProjectSimulationResult) model.SimulateResult {
	if item == nil {
		return model.SimulateResult{}
	}
	return model.SimulateResult(item.Result)
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
