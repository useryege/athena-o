package application

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
)

type bytecodeBlacklistVersionedFake struct {
	items        []appstore.BytecodeBlacklistContract
	version      string
	listCalls    int
	versionCalls int
}

func (f *bytecodeBlacklistVersionedFake) List(context.Context) ([]appstore.BytecodeBlacklistContract, error) {
	f.listCalls++
	return append([]appstore.BytecodeBlacklistContract(nil), f.items...), nil
}

func (f *bytecodeBlacklistVersionedFake) Version(context.Context) (string, error) {
	f.versionCalls++
	return f.version, nil
}

type walletBlacklistVersionedFake struct {
	items        []appstore.WalletBlacklistEntry
	version      string
	listCalls    int
	versionCalls int
}

func (f *walletBlacklistVersionedFake) List(context.Context) ([]appstore.WalletBlacklistEntry, error) {
	f.listCalls++
	return append([]appstore.WalletBlacklistEntry(nil), f.items...), nil
}

func (f *walletBlacklistVersionedFake) Version(context.Context) (string, error) {
	f.versionCalls++
	return f.version, nil
}

func TestProjectPolicyEngineBuildFactsCachesByBlacklistVersion(t *testing.T) {
	bytecodeHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	bytecode := &bytecodeBlacklistVersionedFake{
		items:   []appstore.BytecodeBlacklistContract{{CodeHash: bytecodeHash}},
		version: "bytecode-v1",
	}
	walletAddress := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	wallet := &walletBlacklistVersionedFake{
		items:   []appstore.WalletBlacklistEntry{{Wallet: walletAddress}},
		version: "wallet-v1",
	}
	engine := &projectPolicyEngineImpl{
		bytecodeBlacklist: bytecode,
		walletBlacklist:   wallet,
	}

	facts, err := engine.buildFacts(context.Background())
	if err != nil {
		t.Fatalf("buildFacts first: %v", err)
	}
	if _, ok := facts.BytecodeBlacklist[bytecodeHash]; !ok {
		t.Fatal("bytecode blacklist fact missing")
	}
	if _, ok := facts.WalletBlacklist[walletAddress]; !ok {
		t.Fatal("wallet blacklist fact missing")
	}

	if _, err := engine.buildFacts(context.Background()); err != nil {
		t.Fatalf("buildFacts second: %v", err)
	}
	if bytecode.listCalls != 1 || wallet.listCalls != 1 {
		t.Fatalf("list calls = bytecode:%d wallet:%d, want all 1", bytecode.listCalls, wallet.listCalls)
	}

	wallet.version = "wallet-v2"
	if _, err := engine.buildFacts(context.Background()); err != nil {
		t.Fatalf("buildFacts after version change: %v", err)
	}
	if bytecode.listCalls != 2 || wallet.listCalls != 2 {
		t.Fatalf("list calls after version change = bytecode:%d wallet:%d, want all 2", bytecode.listCalls, wallet.listCalls)
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

func TestProjectPolicyEngineEvaluateRulesUpdatesProjectReport(t *testing.T) {
	contract := common.HexToAddress("0x0000000000000000000000000000000000000201")
	creator := common.HexToAddress("0x0000000000000000000000000000000000000202")
	genesisWallet := common.HexToAddress("0x0000000000000000000000000000000000000203")
	codeBinHash := common.HexToHash("0x2020202020202020202020202020202020202020202020202020202020202020")
	project := &Project{
		Meta: ProjectMeta{
			Contract:    contract,
			Creator:     creator,
			CodeBinHash: codeBinHash,
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
		BytecodeBlacklist: map[common.Hash]struct{}{
			codeBinHash: {},
		},
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
