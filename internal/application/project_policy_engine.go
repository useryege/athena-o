package application

// A project policy audit event is recorded when any policy rule matches:
// - wallet_blacklist_creator: creator wallet matches the wallet blacklist.
// - wallet_blacklist_genesis_wallet: a genesis wallet matches the wallet blacklist.
// - bytecode_blacklist: runtime code hash matches the bytecode blacklist.
// - sourcecode_blacklist_contract: source code hash matches the sourcecode blacklist.
// - simulate_result_mint_risk: creator simulation result has mint risk.

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	appstore "github.com/useryege/athena/internal/application/store"
)

const (
	projectPolicyWorkerCount                = 10
	projectPolicyTaskQueueCapacity          = 4096
	projectPolicyFallbackEvaluationInterval = 45 * time.Minute
)

type ProjectPolicyFacts struct {
	BytecodeBlacklist   map[common.Hash]struct{}
	SourcecodeBlacklist map[common.Hash]struct{}
	WalletBlacklist     map[common.Address]struct{}
}

type projectPolicyFactsSnapshot struct {
	facts   ProjectPolicyFacts
	version string
	ready   bool
}

type ProjectPolicyRule interface {
	Name() string
	Evaluate(ctx context.Context, project *Project, facts ProjectPolicyFacts) (match bool, evidence map[string]any, err error)
}

type policyTaskState struct {
	queued     bool
	processing bool
	dirty      bool
}

type projectPolicyEngineImpl struct {
	projectCache ProjectSnapshotCache

	bytecodeBlacklist   bytecodeBlacklistLister
	sourcecodeBlacklist sourcecodeBlacklistContractLister
	walletBlacklist     walletBlacklistLister

	persistencePublisher PersistenceEventPublisher
	rules                []ProjectPolicyRule
	triggerCh            <-chan common.Address
	taskCh               chan common.Address

	pendingMu sync.Mutex
	pending   map[common.Address]policyTaskState
	factsMu   sync.RWMutex
	facts     projectPolicyFactsSnapshot

	wg sync.WaitGroup
}

type bytecodeBlacklistLister interface {
	List(ctx context.Context) ([]appstore.BytecodeBlacklistContract, error)
}

type sourcecodeBlacklistContractLister interface {
	List(ctx context.Context) ([]appstore.SourcecodeBlacklistContract, error)
}

type walletBlacklistLister interface {
	List(ctx context.Context) ([]appstore.WalletBlacklistEntry, error)
}

type blacklistVersionReader interface {
	Version(ctx context.Context) (string, error)
}

func NewProjectPolicyEngine(
	projectCache ProjectSnapshotCache,
	bytecodeBlacklist bytecodeBlacklistLister,
	sourcecodeBlacklist sourcecodeBlacklistContractLister,
	walletBlacklist walletBlacklistLister,
	persistencePublisher PersistenceEventPublisher,
	triggerCh <-chan common.Address,
) ProjectPolicyEngine {
	return &projectPolicyEngineImpl{
		projectCache:         projectCache,
		bytecodeBlacklist:    bytecodeBlacklist,
		sourcecodeBlacklist:  sourcecodeBlacklist,
		walletBlacklist:      walletBlacklist,
		persistencePublisher: persistencePublisher,
		triggerCh:            triggerCh,
		taskCh:               make(chan common.Address, projectPolicyTaskQueueCapacity),
		pending:              make(map[common.Address]policyTaskState),
		rules: []ProjectPolicyRule{
			walletBlacklistCreatorRule{},
			walletBlacklistGenesisWalletRule{},
			bytecodeBlacklistRule{},
			sourcecodeBlacklistContractRule{},
			simulateMintRiskRule{},
		},
	}
}

func (e *projectPolicyEngineImpl) Start(ctx context.Context) error {
	for i := 0; i < projectPolicyWorkerCount; i++ {
		e.wg.Add(1)
		go func(workerID int) {
			defer e.wg.Done()
			e.workerLoop(ctx, workerID)
		}(i + 1)
	}

	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.runLoop(ctx)
	}()
	return nil
}

func (e *projectPolicyEngineImpl) Stop() error {
	e.wg.Wait()
	return nil
}

func (e *projectPolicyEngineImpl) runLoop(ctx context.Context) {
	ticker := time.NewTicker(projectPolicyFallbackEvaluationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case contract := <-e.triggerCh:
			e.enqueueContract(ctx, contract, "trigger")
		case <-ticker.C:
			if err := e.enqueueAllProjects(ctx); err != nil {
				log.WithFields(log.Fields{
					"component": "project_policy_engine",
					"error":     err.Error(),
				}).Warn("project policy fallback enqueue failed")
			}
		}
	}
}

func (e *projectPolicyEngineImpl) workerLoop(ctx context.Context, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case contract := <-e.taskCh:
			e.markProcessing(contract)
			if err := e.evaluateProject(ctx, contract); err != nil && ctx.Err() == nil {
				log.WithFields(log.Fields{
					"component": "project_policy_engine",
					"worker":    workerID,
					"contract":  contract.Hex(),
					"error":     err.Error(),
				}).Warn("project policy evaluation failed")
			}
			requeue := e.finishProcessing(contract)
			if requeue {
				e.requeueContract(ctx, contract)
			}
		}
	}
}

func (e *projectPolicyEngineImpl) enqueueAllProjects(ctx context.Context) error {
	projects, err := e.listAllProjects(ctx)
	if err != nil {
		return err
	}
	for _, project := range projects {
		if err := ctx.Err(); err != nil {
			return err
		}
		if project == nil {
			continue
		}
		e.enqueueContract(ctx, project.Meta.Contract, "fallback")
	}
	return nil
}

func (e *projectPolicyEngineImpl) enqueueContract(ctx context.Context, contract common.Address, source string) {
	if contract == (common.Address{}) {
		return
	}

	shouldQueue := false
	e.pendingMu.Lock()
	state := e.pending[contract]
	if state.processing {
		state.dirty = true
		e.pending[contract] = state
		e.pendingMu.Unlock()
		return
	}
	if state.queued {
		e.pendingMu.Unlock()
		return
	}
	state.queued = true
	e.pending[contract] = state
	shouldQueue = true
	e.pendingMu.Unlock()

	if !shouldQueue {
		return
	}

	select {
	case <-ctx.Done():
		e.clearQueuedState(contract)
	case e.taskCh <- contract:
		log.WithFields(log.Fields{
			"component": "project_policy_engine",
			"source":    source,
			"contract":  contract.Hex(),
		}).Debug("enqueued project policy task")
	}
}

func (e *projectPolicyEngineImpl) markProcessing(contract common.Address) {
	e.pendingMu.Lock()
	defer e.pendingMu.Unlock()
	state := e.pending[contract]
	state.queued = false
	state.processing = true
	e.pending[contract] = state
}

func (e *projectPolicyEngineImpl) finishProcessing(contract common.Address) bool {
	e.pendingMu.Lock()
	defer e.pendingMu.Unlock()
	state, ok := e.pending[contract]
	if !ok {
		return false
	}
	state.processing = false
	if state.dirty {
		state.dirty = false
		state.queued = true
		e.pending[contract] = state
		return true
	}
	delete(e.pending, contract)
	return false
}

func (e *projectPolicyEngineImpl) requeueContract(ctx context.Context, contract common.Address) {
	select {
	case <-ctx.Done():
		e.clearQueuedState(contract)
	case e.taskCh <- contract:
		log.WithFields(log.Fields{
			"component": "project_policy_engine",
			"source":    "dirty_requeue",
			"contract":  contract.Hex(),
		}).Debug("requeued project policy task")
	}
}

func (e *projectPolicyEngineImpl) clearQueuedState(contract common.Address) {
	e.pendingMu.Lock()
	defer e.pendingMu.Unlock()
	state, ok := e.pending[contract]
	if !ok {
		return
	}
	state.queued = false
	state.dirty = false
	if state.processing {
		e.pending[contract] = state
		return
	}
	delete(e.pending, contract)
}

func (e *projectPolicyEngineImpl) evaluateProject(ctx context.Context, contract common.Address) error {
	if contract == (common.Address{}) {
		return nil
	}

	facts, err := e.buildFacts(ctx)
	if err != nil {
		return err
	}

	project, exists, err := e.projectCache.GetProject(ctx, contract)
	if err != nil {
		return err
	}
	if !exists || project == nil {
		return nil
	}
	if err := e.evaluateRulesForProject(ctx, project, facts); err != nil {
		return err
	}
	return nil
}

func (e *projectPolicyEngineImpl) buildFacts(ctx context.Context) (ProjectPolicyFacts, error) {
	version, versioned, err := e.blacklistFactsVersion(ctx)
	if err != nil {
		return ProjectPolicyFacts{}, err
	}
	if versioned {
		e.factsMu.RLock()
		if e.facts.ready && e.facts.version == version {
			facts := cloneProjectPolicyFacts(e.facts.facts)
			e.factsMu.RUnlock()
			return facts, nil
		}
		e.factsMu.RUnlock()
	}

	facts := ProjectPolicyFacts{}
	if e.bytecodeBlacklist != nil {
		records, err := e.bytecodeBlacklist.List(ctx)
		if err != nil {
			return facts, err
		}
		facts.BytecodeBlacklist = make(map[common.Hash]struct{}, len(records))
		for _, item := range records {
			facts.BytecodeBlacklist[item.CodeHash] = struct{}{}
		}
	}
	if e.sourcecodeBlacklist != nil {
		records, err := e.sourcecodeBlacklist.List(ctx)
		if err != nil {
			return facts, err
		}
		facts.SourcecodeBlacklist = make(map[common.Hash]struct{}, len(records))
		for _, item := range records {
			facts.SourcecodeBlacklist[item.SourceHash] = struct{}{}
		}
	}
	if e.walletBlacklist != nil {
		records, err := e.walletBlacklist.List(ctx)
		if err != nil {
			return facts, err
		}
		facts.WalletBlacklist = make(map[common.Address]struct{}, len(records))
		for _, item := range records {
			facts.WalletBlacklist[item.Wallet] = struct{}{}
		}
	}
	if versioned {
		e.factsMu.Lock()
		e.facts = projectPolicyFactsSnapshot{
			facts:   cloneProjectPolicyFacts(facts),
			version: version,
			ready:   true,
		}
		e.factsMu.Unlock()
	}
	return facts, nil
}

func (e *projectPolicyEngineImpl) blacklistFactsVersion(ctx context.Context) (string, bool, error) {
	bytecodeVersion, bytecodeOK, err := blacklistVersion(ctx, e.bytecodeBlacklist)
	if err != nil {
		return "", false, err
	}
	sourcecodeVersion, sourcecodeOK, err := blacklistVersion(ctx, e.sourcecodeBlacklist)
	if err != nil {
		return "", false, err
	}
	walletVersion, walletOK, err := blacklistVersion(ctx, e.walletBlacklist)
	if err != nil {
		return "", false, err
	}
	if !bytecodeOK || !sourcecodeOK || !walletOK {
		return "", false, nil
	}
	return bytecodeVersion + "|" + sourcecodeVersion + "|" + walletVersion, true, nil
}

func blacklistVersion(ctx context.Context, value any) (string, bool, error) {
	if value == nil {
		return "", true, nil
	}
	reader, ok := value.(blacklistVersionReader)
	if !ok {
		return "", false, nil
	}
	version, err := reader.Version(ctx)
	if err != nil {
		return "", false, err
	}
	return version, true, nil
}

func cloneProjectPolicyFacts(facts ProjectPolicyFacts) ProjectPolicyFacts {
	cloned := ProjectPolicyFacts{}
	if facts.BytecodeBlacklist != nil {
		cloned.BytecodeBlacklist = make(map[common.Hash]struct{}, len(facts.BytecodeBlacklist))
		for key := range facts.BytecodeBlacklist {
			cloned.BytecodeBlacklist[key] = struct{}{}
		}
	}
	if facts.SourcecodeBlacklist != nil {
		cloned.SourcecodeBlacklist = make(map[common.Hash]struct{}, len(facts.SourcecodeBlacklist))
		for key := range facts.SourcecodeBlacklist {
			cloned.SourcecodeBlacklist[key] = struct{}{}
		}
	}
	if facts.WalletBlacklist != nil {
		cloned.WalletBlacklist = make(map[common.Address]struct{}, len(facts.WalletBlacklist))
		for key := range facts.WalletBlacklist {
			cloned.WalletBlacklist[key] = struct{}{}
		}
	}
	return cloned
}

func (e *projectPolicyEngineImpl) listAllProjects(ctx context.Context) ([]*Project, error) {
	projects, err := e.projectCache.ListProjects(ctx)
	if err != nil {
		return nil, err
	}
	all := make([]*Project, 0, len(projects))
	seen := make(map[common.Address]struct{}, len(projects))
	appendUnique := func(items []*Project) {
		for _, project := range items {
			if project == nil {
				continue
			}
			contract := project.Meta.Contract
			if _, ok := seen[contract]; ok {
				continue
			}
			seen[contract] = struct{}{}
			all = append(all, project)
		}
	}
	appendUnique(projects)
	return all, nil
}

func (e *projectPolicyEngineImpl) evaluateRulesForProject(ctx context.Context, project *Project, facts ProjectPolicyFacts) error {
	if project == nil {
		return nil
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
		return err
	}
	project.Report = report

	for _, match := range matches {
		now := time.Now().UTC()
		if err := e.persistPolicyAuditEvent(ctx, project.Meta.Contract, match.ruleName, match.evidence, now); err != nil {
			continue
		}
	}
	return nil
}

func (e *projectPolicyEngineImpl) updateProjectReport(ctx context.Context, contract common.Address, report ProjectReport) error {
	if e == nil || e.projectCache == nil {
		return nil
	}
	_, err := e.projectCache.UpdateProject(ctx, contract, func(current *Project, exists bool) (*Project, bool, error) {
		if !exists || current == nil || current.Report == report {
			return nil, false, nil
		}
		current.Report = report
		return current, true, nil
	})
	return err
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
	case sourcecodeBlacklistContractRule{}.Name():
		report.IsBlacklistedSourceCode = true
	case simulateMintRiskRule{}.Name():
		report.HasMintRisk = true
	}
}

func (e *projectPolicyEngineImpl) persistPolicyAuditEvent(ctx context.Context, contract common.Address, ruleName string, evidence map[string]any, now time.Time) error {
	if e.persistencePublisher == nil {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{
		"rule":     ruleName,
		"evidence": evidence,
		"source":   "policy_engine",
	})
	return e.persistencePublisher.PublishProjectEventLog(ctx, appstore.ProjectEventLog{
		Contract:       contract,
		EventType:      projectEventTypePolicyMatchAudit,
		OccurredAt:     now,
		Message:        fmt.Sprintf("Policy rule %s matched project", ruleName),
		Payload:        string(payload),
		IdempotencyKey: projectEventIdempotencyPolicyMatch(ruleName),
	})
}

type bytecodeBlacklistRule struct{}

func (r bytecodeBlacklistRule) Name() string { return "bytecode_blacklist" }

func (r bytecodeBlacklistRule) Evaluate(_ context.Context, project *Project, facts ProjectPolicyFacts) (bool, map[string]any, error) {
	if project == nil || project.Meta.CodeBinHash == (common.Hash{}) || len(facts.BytecodeBlacklist) == 0 {
		return false, nil, nil
	}
	if _, ok := facts.BytecodeBlacklist[project.Meta.CodeBinHash]; !ok {
		return false, nil, nil
	}
	return true, map[string]any{
		"code_bin_hash": strings.ToLower(project.Meta.CodeBinHash.Hex()),
	}, nil
}

type sourcecodeBlacklistContractRule struct{}

func (r sourcecodeBlacklistContractRule) Name() string { return "sourcecode_blacklist_contract" }

func (r sourcecodeBlacklistContractRule) Evaluate(_ context.Context, project *Project, facts ProjectPolicyFacts) (bool, map[string]any, error) {
	if project == nil || project.Meta.SourceCode == "" || len(facts.SourcecodeBlacklist) == 0 {
		return false, nil, nil
	}
	sourceHash := project.Meta.SourceCodeHash
	if sourceHash == (common.Hash{}) {
		sourceHash = crypto.Keccak256Hash([]byte(project.Meta.SourceCode))
	}
	if _, ok := facts.SourcecodeBlacklist[sourceHash]; !ok {
		return false, nil, nil
	}
	return true, map[string]any{
		"source_hash": strings.ToLower(sourceHash.Hex()),
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
