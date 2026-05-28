package application

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

type walletBlacklistVersionedFake struct {
	items     []common.Address
	listCalls int
}

func (f *walletBlacklistVersionedFake) ListWalletBlacklist(context.Context) ([]common.Address, error) {
	f.listCalls++
	return append([]common.Address(nil), f.items...), nil
}

type policyEngineProjectCache struct {
	projects map[common.Address]*Project
	active   []common.Address
}

func newPolicyReevaluationProjectCache(projects ...*Project) *policyEngineProjectCache {
	cache := &policyEngineProjectCache{
		projects: make(map[common.Address]*Project, len(projects)),
	}
	for _, project := range projects {
		if project == nil {
			continue
		}
		contract := project.Meta.Contract
		cache.projects[contract] = project
		cache.active = append(cache.active, contract)
	}
	return cache
}

func (c *policyEngineProjectCache) ReplaceAll(context.Context, []*Project) error {
	return nil
}

func (c *policyEngineProjectCache) SetProject(_ context.Context, project *Project) error {
	if project == nil {
		return nil
	}
	c.projects[project.Meta.Contract] = project
	return nil
}

func (c *policyEngineProjectCache) UpdateProject(_ context.Context, contract common.Address, updater ProjectUpdater) (bool, error) {
	current, exists := c.projects[contract]
	next, changed, err := updater(current, exists)
	if err != nil || !changed {
		return false, err
	}
	if next == nil {
		delete(c.projects, contract)
		return true, nil
	}
	c.projects[contract] = next
	return true, nil
}

func (c *policyEngineProjectCache) DeleteProject(_ context.Context, contract common.Address) error {
	delete(c.projects, contract)
	return nil
}

func (c *policyEngineProjectCache) GetProject(_ context.Context, contract common.Address) (*Project, bool, error) {
	project, ok := c.projects[contract]
	return project, ok, nil
}

func (c *policyEngineProjectCache) ListProjects(context.Context) ([]*Project, error) {
	projects := make([]*Project, 0, len(c.active))
	for _, contract := range c.active {
		if project := c.projects[contract]; project != nil {
			projects = append(projects, project)
		}
	}
	return projects, nil
}

func (c *policyEngineProjectCache) ListProjectsByPairAddresses(_ context.Context, pairs []common.Address) ([]*Project, error) {
	pairSet := make(map[common.Address]struct{}, len(pairs))
	for _, pair := range pairs {
		pairSet[pair] = struct{}{}
	}
	projects := make([]*Project, 0)
	for _, project := range c.projects {
		if project == nil {
			continue
		}
		if _, ok := pairSet[project.Meta.WethPair]; ok {
			projects = append(projects, project)
			continue
		}
		if _, ok := pairSet[project.Meta.UsdtPair]; ok {
			projects = append(projects, project)
		}
	}
	return projects, nil
}

func (c *policyEngineProjectCache) ListProjectsPage(_ context.Context, page int32, pageSize int32) ([]*Project, int64, int32, int32, error) {
	projects, err := c.ListProjects(context.Background())
	if err != nil {
		return nil, 0, 0, 0, err
	}
	total, page, pageSize := paginateProjects(page, pageSize, &projects)
	return projects, total, page, pageSize, nil
}

var _ ProjectSnapshotCache = (*policyEngineProjectCache)(nil)

func TestProjectPolicyEngineBuildFactsReadsWalletBlacklistEachTime(t *testing.T) {
	walletAddress := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	wallet := &walletBlacklistVersionedFake{
		items: []common.Address{walletAddress},
	}
	engine := &projectPolicyEngineImpl{
		walletBlacklist: wallet,
	}

	facts, err := engine.buildFacts(context.Background())
	if err != nil {
		t.Fatalf("buildFacts first: %v", err)
	}
	if _, ok := facts.WalletBlacklist[walletAddress]; !ok {
		t.Fatal("wallet blacklist fact missing")
	}

	if _, err := engine.buildFacts(context.Background()); err != nil {
		t.Fatalf("buildFacts second: %v", err)
	}
	if wallet.listCalls != 2 {
		t.Fatalf("wallet list calls = %d, want 2", wallet.listCalls)
	}
}

type recordingPolicyRule struct {
	contracts *[]common.Address
}

func (r recordingPolicyRule) Name() string { return "recording" }

func (r recordingPolicyRule) Evaluate(_ context.Context, project *Project, _ ProjectPolicyFacts) (bool, map[string]any, error) {
	*r.contracts = append(*r.contracts, project.Meta.Contract)
	return false, nil, nil
}

type matchingPolicyRule struct {
	name     string
	evidence map[string]any
}

func (r matchingPolicyRule) Name() string { return r.name }

func (r matchingPolicyRule) Evaluate(context.Context, *Project, ProjectPolicyFacts) (bool, map[string]any, error) {
	return true, r.evidence, nil
}

func TestProjectPolicyEngineEvaluateRulesUpdatesProjectReport(t *testing.T) {
	contract := common.HexToAddress("0x0000000000000000000000000000000000000201")
	creator := common.HexToAddress("0x0000000000000000000000000000000000000202")
	genesisWallet := common.HexToAddress("0x0000000000000000000000000000000000000203")
	project := &Project{
		Meta: ProjectMeta{
			Contract:              contract,
			Creator:               creator,
			IsBytecodeBlacklisted: true,
			GenesisWallets: []GenesisWalletMeta{{
				Wallet: genesisWallet,
			}},
			CreatorResult: SimulateResult{
				CanMintFromZeroViaTransferFrom: true,
			},
		},
	}
	cache := newPolicyReevaluationProjectCache(project)
	publisher := &persistencePublisherFake{}
	engine := &projectPolicyEngineImpl{
		projectCache:         cache,
		persistencePublisher: publisher,
		rules: []ProjectPolicyRule{
			walletBlacklistCreatorRule{},
			walletBlacklistGenesisWalletRule{},
			bytecodeBlacklistRule{},
			simulateMintRiskRule{},
		},
	}

	if _, err := engine.evaluateRulesForProject(context.Background(), project, ProjectPolicyFacts{
		WalletBlacklist: map[common.Address]struct{}{
			creator:       {},
			genesisWallet: {},
		},
	}); err != nil {
		t.Fatalf("evaluateRulesForProject: %v", err)
	}

	report := cache.projects[contract].Report
	if !report.IsPolicyEvaluated ||
		!report.IsBlacklistedCreatorWallet ||
		!report.IsBlacklistedGenesisWallet ||
		!report.IsBlacklistedBytecode ||
		!report.HasMintRisk {
		t.Fatalf("project report = %+v, want all policy fields true", report)
	}
	if publisher.projectReports[contract] != report {
		t.Fatalf("persisted project report = %+v, want %+v", publisher.projectReports[contract], report)
	}
}

func TestProjectPolicyEngineEvaluateRulesPublishesPolicyMatchedEvent(t *testing.T) {
	contract := common.HexToAddress("0x0000000000000000000000000000000000000204")
	project := &Project{Meta: ProjectMeta{Contract: contract}}
	cache := newPolicyReevaluationProjectCache(project)
	publisher := &persistencePublisherFake{}
	engine := &projectPolicyEngineImpl{
		projectCache:         cache,
		persistencePublisher: publisher,
		rules: []ProjectPolicyRule{
			matchingPolicyRule{
				name: "custom_policy_rule",
				evidence: map[string]any{
					"reason": "matched",
				},
			},
		},
	}

	if _, err := engine.evaluateRulesForProject(context.Background(), project, ProjectPolicyFacts{}); err != nil {
		t.Fatalf("evaluateRulesForProject: %v", err)
	}

	if len(publisher.projectEventLogs) != 1 {
		t.Fatalf("project event logs = %d, want 1", len(publisher.projectEventLogs))
	}
	item := publisher.projectEventLogs[0]
	if item.Contract != contract {
		t.Fatalf("event contract = %s, want %s", item.Contract.Hex(), contract.Hex())
	}
	if item.EventType != int16(projectEventTypePolicyMatchAudit) {
		t.Fatalf("event type = %d, want %d", item.EventType, projectEventTypePolicyMatchAudit)
	}
	if item.IdempotencyKey != projectEventIdempotencyPolicyMatch("custom_policy_rule") {
		t.Fatalf("idempotency key = %q, want %q", item.IdempotencyKey, projectEventIdempotencyPolicyMatch("custom_policy_rule"))
	}
	if !json.Valid([]byte(item.Payload)) {
		t.Fatalf("payload is not valid JSON: %q", item.Payload)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(item.Payload), &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["rule"] != "custom_policy_rule" {
		t.Fatalf("payload rule = %v, want custom_policy_rule", payload["rule"])
	}
	if payload["source"] != "policy_engine" {
		t.Fatalf("payload source = %v, want policy_engine", payload["source"])
	}
	evidence, ok := payload["evidence"].(map[string]any)
	if !ok {
		t.Fatalf("payload evidence = %T, want object", payload["evidence"])
	}
	if evidence["reason"] != "matched" {
		t.Fatalf("payload evidence reason = %v, want matched", evidence["reason"])
	}
}

func TestProjectPolicyEngineEvaluateRulesRecomputesProjectReport(t *testing.T) {
	contract := common.HexToAddress("0x0000000000000000000000000000000000000211")
	project := &Project{
		Meta: ProjectMeta{Contract: contract},
		Report: ProjectReport{
			IsPolicyEvaluated:          true,
			IsBlacklistedCreatorWallet: true,
			IsBlacklistedGenesisWallet: true,
			IsBlacklistedBytecode:      true,
			HasMintRisk:                true,
		},
	}
	cache := newPolicyReevaluationProjectCache(project)
	publisher := &persistencePublisherFake{}
	engine := &projectPolicyEngineImpl{
		projectCache:         cache,
		persistencePublisher: publisher,
		rules: []ProjectPolicyRule{
			walletBlacklistCreatorRule{},
			walletBlacklistGenesisWalletRule{},
			bytecodeBlacklistRule{},
			simulateMintRiskRule{},
		},
	}

	if _, err := engine.evaluateRulesForProject(context.Background(), project, ProjectPolicyFacts{}); err != nil {
		t.Fatalf("evaluateRulesForProject: %v", err)
	}

	want := ProjectReport{IsPolicyEvaluated: true}
	if got := cache.projects[contract].Report; got != want {
		t.Fatalf("project report = %+v, want %+v", got, want)
	}
	if got := publisher.projectReports[contract]; got != want {
		t.Fatalf("persisted project report = %+v, want %+v", got, want)
	}
}
