package policy

// A project policy audit event is recorded when any policy rule matches:
// - wallet_blacklist_creator: creator wallet matches the wallet blacklist.
// - wallet_blacklist_genesis_wallet: a genesis wallet matches the wallet blacklist.
// - bytecode_blacklist: runtime code hash matches the bytecode blacklist.
// - simulate_result_mint_risk: creator simulation result has mint risk.

import (
	"context"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/persistence"
	appstore "github.com/useryege/athena/internal/application/store"
)

type ProjectPolicyFacts struct {
	WalletBlacklist map[common.Address]struct{}
}

type ProjectPolicyRule interface {
	Name() string
	Evaluate(ctx context.Context, project *Project, facts ProjectPolicyFacts) (match bool, evidence map[string]any, err error)
}

type projectPolicyEngineImpl struct {
	componentCache appcache.ProjectComponentCache
	projectCache   appcache.ProjectSnapshotCache
	projectStore   appstore.ProjectStore

	walletBlacklist walletBlacklistLister

	persistencePublisher persistence.PersistenceEventPublisher
	rules                []ProjectPolicyRule
}

type walletBlacklistLister interface {
	ListWalletBlacklist(ctx context.Context) ([]common.Address, error)
}

func NewProjectPolicyEngine(
	componentCache appcache.ProjectComponentCache,
	projectStore appstore.ProjectStore,
	walletBlacklist walletBlacklistLister,
	persistencePublisher persistence.PersistenceEventPublisher,
) Engine {
	return &projectPolicyEngineImpl{
		componentCache:       componentCache,
		projectStore:         projectStore,
		walletBlacklist:      walletBlacklist,
		persistencePublisher: persistencePublisher,
	}
}

func (e *projectPolicyEngineImpl) EvaluateProject(ctx context.Context, contract common.Address) (ProjectReport, error) {
	if contract == (common.Address{}) {
		return ProjectReport{}, nil
	}

	facts, err := e.buildFacts(ctx)
	if err != nil {
		return ProjectReport{}, err
	}

	facts, projectFacts, err := e.projectFacts(ctx, contract, facts)
	if err != nil || projectFacts == nil {
		return ProjectReport{}, err
	}
	return e.evaluateFactsForProject(ctx, projectFacts, facts)
}

type projectPolicyProjectFacts struct {
	Contract            common.Address
	Creator             common.Address
	GenesisWallets      []common.Address
	SimulationResult    SimulateResult
	BytecodeBlacklisted bool
}

func (e *projectPolicyEngineImpl) projectFacts(ctx context.Context, contract common.Address, facts ProjectPolicyFacts) (ProjectPolicyFacts, *projectPolicyProjectFacts, error) {
	base, err := e.projectBase(ctx, contract)
	if err != nil || base == nil {
		return facts, nil, err
	}
	projectFacts := &projectPolicyProjectFacts{
		Contract: contract,
		Creator:  base.Creator,
	}
	if wallets, err := e.projectGenesisWallets(ctx, contract); err != nil {
		return facts, nil, err
	} else {
		projectFacts.GenesisWallets = wallets
	}
	if simulation, err := e.projectSimulation(ctx, contract); err != nil {
		return facts, nil, err
	} else if simulation != nil {
		projectFacts.SimulationResult = SimulateResult(simulation.Result)
	}
	if bytecodeFact, err := e.projectBytecodeFact(ctx, contract); err != nil {
		return facts, nil, err
	} else if bytecodeFact != nil {
		projectFacts.BytecodeBlacklisted = bytecodeFact.IsBytecodeBlacklisted
	}
	return facts, projectFacts, nil
}

func (e *projectPolicyEngineImpl) projectBase(ctx context.Context, contract common.Address) (*appstore.ProjectBase, error) {
	if e.componentCache != nil {
		if base, ok, err := e.componentCache.GetBase(ctx, contract); err != nil {
			return nil, err
		} else if ok && base != nil {
			return base, nil
		}
	}
	store, ok := e.projectStore.(appstore.ProjectBaseStore)
	if !ok || store == nil {
		return nil, nil
	}
	base, err := store.GetProjectBaseByContract(ctx, contract)
	if err != nil || base == nil {
		return base, err
	}
	if e.componentCache != nil {
		if err := e.componentCache.SetBase(ctx, *base); err != nil {
			return nil, err
		}
	}
	return base, nil
}

func (e *projectPolicyEngineImpl) projectGenesisWallets(ctx context.Context, contract common.Address) ([]common.Address, error) {
	if e.componentCache != nil {
		if items, ok, err := e.componentCache.GetGenesisWallets(ctx, contract); err != nil {
			return nil, err
		} else if ok {
			return genesisWalletAddressesFromStore(items), nil
		}
	}
	store, ok := e.projectStore.(appstore.ProjectGenesisWalletStore)
	if !ok || store == nil {
		return nil, nil
	}
	items, err := store.ListProjectGenesisWalletsByContract(ctx, contract)
	if err != nil {
		return nil, err
	}
	if e.componentCache != nil {
		if err := e.componentCache.SetGenesisWallets(ctx, contract, items); err != nil {
			return nil, err
		}
	}
	return genesisWalletAddressesFromStore(items), nil
}

func (e *projectPolicyEngineImpl) projectSimulation(ctx context.Context, contract common.Address) (*appstore.ProjectSimulationResult, error) {
	if e.componentCache != nil {
		if item, ok, err := e.componentCache.GetSimulation(ctx, contract); err != nil {
			return nil, err
		} else if ok && item != nil {
			return item, nil
		}
	}
	store, ok := e.projectStore.(appstore.ProjectSimulationStore)
	if !ok || store == nil {
		return nil, nil
	}
	item, err := store.GetProjectSimulationResult(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	if e.componentCache != nil {
		if err := e.componentCache.SetSimulation(ctx, *item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func (e *projectPolicyEngineImpl) projectBytecodeFact(ctx context.Context, contract common.Address) (*appstore.ProjectBytecodeFact, error) {
	if e.componentCache != nil {
		if item, ok, err := e.componentCache.GetBytecodeFact(ctx, contract); err != nil {
			return nil, err
		} else if ok && item != nil {
			return item, nil
		}
	}
	store, ok := e.projectStore.(appstore.ProjectBytecodeFactStore)
	if !ok || store == nil {
		return nil, nil
	}
	item, err := store.GetProjectBytecodeFact(ctx, contract)
	if err != nil || item == nil {
		return item, err
	}
	if e.componentCache != nil {
		if err := e.componentCache.SetBytecodeFact(ctx, *item); err != nil {
			return nil, err
		}
	}
	return item, nil
}

func genesisWalletAddressesFromStore(items []appstore.ProjectGenesisWallet) []common.Address {
	addresses := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item.Wallet == (common.Address{}) {
			continue
		}
		addresses = append(addresses, item.Wallet)
	}
	return addresses
}

func genesisWalletAddressesFromModel(items []GenesisWalletMeta) []common.Address {
	addresses := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item.Wallet == (common.Address{}) {
			continue
		}
		addresses = append(addresses, item.Wallet)
	}
	return addresses
}

func (e *projectPolicyEngineImpl) buildFacts(ctx context.Context) (ProjectPolicyFacts, error) {
	facts := ProjectPolicyFacts{}
	if e.walletBlacklist != nil {
		wallets, err := e.walletBlacklist.ListWalletBlacklist(ctx)
		if err != nil {
			return facts, err
		}
		facts.WalletBlacklist = make(map[common.Address]struct{}, len(wallets))
		for _, wallet := range wallets {
			facts.WalletBlacklist[wallet] = struct{}{}
		}
	}
	return facts, nil
}

func (e *projectPolicyEngineImpl) evaluateFactsForProject(ctx context.Context, project *projectPolicyProjectFacts, facts ProjectPolicyFacts) (ProjectReport, error) {
	if project == nil {
		return ProjectReport{}, nil
	}

	report := ProjectReport{IsPolicyEvaluated: true}
	type projectPolicyRuleMatch struct {
		ruleName string
		evidence map[string]any
	}
	matches := make([]projectPolicyRuleMatch, 0, 4)
	if _, ok := facts.WalletBlacklist[project.Creator]; ok {
		report.IsBlacklistedCreatorWallet = true
		matches = append(matches, projectPolicyRuleMatch{ruleName: "wallet_blacklist_creator", evidence: map[string]any{"creator_wallet": strings.ToLower(project.Creator.Hex())}})
	}
	matchedGenesisWallets := make([]string, 0)
	seenGenesisWallets := map[common.Address]struct{}{}
	for _, wallet := range project.GenesisWallets {
		if _, ok := facts.WalletBlacklist[wallet]; !ok {
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
		matches = append(matches, projectPolicyRuleMatch{ruleName: "wallet_blacklist_genesis_wallet", evidence: map[string]any{"blacklisted_genesis_wallets": matchedGenesisWallets}})
	}
	if project.BytecodeBlacklisted {
		report.IsBlacklistedBytecode = true
		matches = append(matches, projectPolicyRuleMatch{ruleName: "bytecode_blacklist", evidence: map[string]any{"contract": strings.ToLower(project.Contract.Hex())}})
	}
	if project.SimulationResult.HasMintRisk() {
		report.HasMintRisk = true
		matches = append(matches, projectPolicyRuleMatch{ruleName: "simulate_result_mint_risk", evidence: map[string]any{"mintable_paths": project.SimulationResult.MintablePaths()}})
	}
	if err := e.updateProjectReport(ctx, project.Contract, report); err != nil {
		return ProjectReport{}, err
	}

	for _, match := range matches {
		now := time.Now().UTC()
		if err := e.persistPolicyAuditEvent(ctx, project.Contract, match.ruleName, match.evidence, now); err != nil {
			continue
		}
	}
	return report, nil
}

func (e *projectPolicyEngineImpl) evaluateRulesForProject(ctx context.Context, project *Project, facts ProjectPolicyFacts) (ProjectReport, error) {
	if project == nil {
		return ProjectReport{}, nil
	}
	if len(e.rules) > 0 {
		report := ProjectReport{IsPolicyEvaluated: true}
		type projectPolicyRuleMatch struct {
			ruleName string
			evidence map[string]any
		}
		matches := make([]projectPolicyRuleMatch, 0, len(e.rules))
		for _, rule := range e.rules {
			if rule == nil {
				continue
			}
			match, evidence, err := rule.Evaluate(ctx, project, facts)
			if err != nil || !match {
				continue
			}
			markProjectReportRuleMatch(&report, rule.Name())
			matches = append(matches, projectPolicyRuleMatch{ruleName: rule.Name(), evidence: evidence})
		}
		if err := e.updateProjectReport(ctx, project.Meta.Contract, report); err != nil {
			return ProjectReport{}, err
		}
		for _, match := range matches {
			if err := e.persistPolicyAuditEvent(ctx, project.Meta.Contract, match.ruleName, match.evidence, time.Now().UTC()); err != nil {
				continue
			}
		}
		return report, nil
	}
	projectFacts := &projectPolicyProjectFacts{
		Contract:            project.Meta.Contract,
		Creator:             project.Meta.Creator,
		GenesisWallets:      genesisWalletAddressesFromModel(project.Meta.GenesisWallets),
		SimulationResult:    project.Meta.CreatorResult,
		BytecodeBlacklisted: project.Meta.IsBytecodeBlacklisted,
	}
	return e.evaluateFactsForProject(ctx, projectFacts, facts)
}

func (e *projectPolicyEngineImpl) updateProjectReport(ctx context.Context, contract common.Address, report ProjectReport) error {
	if e == nil {
		return nil
	}
	item := appstore.ProjectPolicyReport{ProjectContract: contract, Report: appstore.ProjectReport(report), EvaluatedAt: time.Now().UTC()}
	if e.componentCache == nil && e.projectCache != nil {
		changed := false
		_, err := e.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
			if !exists || current == nil || current.Report == report {
				return nil, false, nil
			}
			current.Report = report
			changed = true
			return current, true, nil
		})
		if err != nil {
			return err
		}
		if changed && e.persistencePublisher != nil {
			return e.persistencePublisher.PublishProjectReportUpdate(ctx, contract, report)
		}
		return nil
	}
	if store, ok := e.projectStore.(appstore.ProjectPolicyReportStore); ok && store != nil {
		if err := store.UpsertProjectPolicyReport(ctx, item); err != nil {
			return err
		}
	}
	if e.componentCache != nil {
		if err := e.componentCache.SetReport(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func markProjectReportRuleMatch(report *ProjectReport, ruleName string) {
	if report == nil {
		return
	}
	switch ruleName {
	case walletBlacklistCreatorRule{}.Name():
		report.IsBlacklistedCreatorWallet = true
	case walletBlacklistGenesisWalletRule{}.Name():
		report.IsBlacklistedGenesisWallet = true
	case bytecodeBlacklistRule{}.Name():
		report.IsBlacklistedBytecode = true
	case simulateMintRiskRule{}.Name():
		report.HasMintRisk = true
	}
}

func (e *projectPolicyEngineImpl) persistPolicyAuditEvent(ctx context.Context, contract common.Address, ruleName string, evidence map[string]any, now time.Time) error {
	if e.persistencePublisher == nil {
		return nil
	}
	event, err := persistence.NewProjectPolicyMatchedEvent(contract, ruleName, evidence, now)
	if err != nil {
		return err
	}
	return e.persistencePublisher.PublishProjectEventLog(ctx, event)
}

type bytecodeBlacklistRule struct{}

func (r bytecodeBlacklistRule) Name() string { return "bytecode_blacklist" }

func (r bytecodeBlacklistRule) Evaluate(_ context.Context, project *Project, facts ProjectPolicyFacts) (bool, map[string]any, error) {
	if project == nil || !project.Meta.IsBytecodeBlacklisted {
		return false, nil, nil
	}
	return true, map[string]any{
		"contract": strings.ToLower(project.Meta.Contract.Hex()),
	}, nil
}

type simulateMintRiskRule struct{}

func (r simulateMintRiskRule) Name() string { return "simulate_result_mint_risk" }

func (r simulateMintRiskRule) Evaluate(_ context.Context, project *Project, _ ProjectPolicyFacts) (bool, map[string]any, error) {
	if project == nil {
		return false, nil, nil
	}
	if !project.Meta.CreatorResult.HasMintRisk() {
		return false, nil, nil
	}
	return true, map[string]any{
		"mintable_paths": project.Meta.CreatorResult.MintablePaths(),
	}, nil
}

type walletBlacklistCreatorRule struct{}

func (r walletBlacklistCreatorRule) Name() string { return "wallet_blacklist_creator" }

func (r walletBlacklistCreatorRule) Evaluate(_ context.Context, project *Project, facts ProjectPolicyFacts) (bool, map[string]any, error) {
	if project == nil || len(facts.WalletBlacklist) == 0 {
		return false, nil, nil
	}
	if _, ok := facts.WalletBlacklist[project.Meta.Creator]; !ok {
		return false, nil, nil
	}
	return true, map[string]any{
		"creator_wallet": strings.ToLower(project.Meta.Creator.Hex()),
	}, nil
}

type walletBlacklistGenesisWalletRule struct{}

func (r walletBlacklistGenesisWalletRule) Name() string { return "wallet_blacklist_genesis_wallet" }

func (r walletBlacklistGenesisWalletRule) Evaluate(_ context.Context, project *Project, facts ProjectPolicyFacts) (bool, map[string]any, error) {
	if project == nil || len(project.Meta.GenesisWallets) == 0 || len(facts.WalletBlacklist) == 0 {
		return false, nil, nil
	}

	matched := make([]string, 0)
	seen := make(map[common.Address]struct{})
	for _, item := range project.Meta.GenesisWallets {
		if _, ok := facts.WalletBlacklist[item.Wallet]; !ok {
			continue
		}
		if _, duplicated := seen[item.Wallet]; duplicated {
			continue
		}
		seen[item.Wallet] = struct{}{}
		matched = append(matched, strings.ToLower(item.Wallet.Hex()))
	}
	if len(matched) == 0 {
		return false, nil, nil
	}
	return true, map[string]any{
		"blacklisted_genesis_wallets": matched,
	}, nil
}
