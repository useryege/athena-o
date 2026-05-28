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
)

type ProjectPolicyFacts struct {
	WalletBlacklist map[common.Address]struct{}
}

type ProjectPolicyRule interface {
	Name() string
	Evaluate(ctx context.Context, project *Project, facts ProjectPolicyFacts) (match bool, evidence map[string]any, err error)
}

type projectPolicyEngineImpl struct {
	projectCache appcache.ProjectSnapshotCache

	walletBlacklist walletBlacklistLister

	persistencePublisher persistence.PersistenceEventPublisher
	rules                []ProjectPolicyRule
}

type walletBlacklistLister interface {
	ListWalletBlacklist(ctx context.Context) ([]common.Address, error)
}

func NewProjectPolicyEngine(
	projectCache appcache.ProjectSnapshotCache,
	walletBlacklist walletBlacklistLister,
	persistencePublisher persistence.PersistenceEventPublisher,
) Engine {
	return &projectPolicyEngineImpl{
		projectCache:         projectCache,
		walletBlacklist:      walletBlacklist,
		persistencePublisher: persistencePublisher,
		rules: []ProjectPolicyRule{
			walletBlacklistCreatorRule{},
			walletBlacklistGenesisWalletRule{},
			bytecodeBlacklistRule{},
			simulateMintRiskRule{},
		},
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

	project, exists, err := e.projectCache.GetProject(ctx, contract)
	if err != nil {
		return ProjectReport{}, err
	}
	if !exists || project == nil {
		return ProjectReport{}, nil
	}
	return e.evaluateRulesForProject(ctx, project, facts)
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

func (e *projectPolicyEngineImpl) evaluateRulesForProject(ctx context.Context, project *Project, facts ProjectPolicyFacts) (ProjectReport, error) {
	if project == nil {
		return ProjectReport{}, nil
	}

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
		if err != nil {
			continue
		}
		if !match {
			continue
		}
		markProjectReportRuleMatch(&report, rule.Name())
		matches = append(matches, projectPolicyRuleMatch{
			ruleName: rule.Name(),
			evidence: evidence,
		})
	}
	if err := e.updateProjectReport(ctx, project.Meta.Contract, report); err != nil {
		return ProjectReport{}, err
	}
	project.Report = report

	for _, match := range matches {
		now := time.Now().UTC()
		if err := e.persistPolicyAuditEvent(ctx, project.Meta.Contract, match.ruleName, match.evidence, now); err != nil {
			continue
		}
	}
	return report, nil
}

func (e *projectPolicyEngineImpl) updateProjectReport(ctx context.Context, contract common.Address, report ProjectReport) error {
	if e == nil {
		return nil
	}
	changed := e.projectCache == nil
	if e.projectCache != nil {
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
	}
	if !changed || e.persistencePublisher == nil {
		return nil
	}
	return e.persistencePublisher.PublishProjectReportUpdate(ctx, contract, report)
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
