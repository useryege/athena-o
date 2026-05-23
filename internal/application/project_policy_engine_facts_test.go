package application

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
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

type sourcecodeBlacklistVersionedFake struct {
	items        []appstore.SourcecodeBlacklistContract
	version      string
	listCalls    int
	versionCalls int
}

func (f *sourcecodeBlacklistVersionedFake) List(context.Context) ([]appstore.SourcecodeBlacklistContract, error) {
	f.listCalls++
	return append([]appstore.SourcecodeBlacklistContract(nil), f.items...), nil
}

func (f *sourcecodeBlacklistVersionedFake) Version(context.Context) (string, error) {
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
	sourcecodeHash := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	sourcecode := &sourcecodeBlacklistVersionedFake{
		items:   []appstore.SourcecodeBlacklistContract{{SourceHash: sourcecodeHash}},
		version: "sourcecode-v1",
	}
	walletAddress := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	wallet := &walletBlacklistVersionedFake{
		items:   []appstore.WalletBlacklistEntry{{Wallet: walletAddress}},
		version: "wallet-v1",
	}
	engine := &projectPolicyEngineImpl{
		bytecodeBlacklist:   bytecode,
		sourcecodeBlacklist: sourcecode,
		walletBlacklist:     wallet,
	}

	facts, err := engine.buildFacts(context.Background())
	if err != nil {
		t.Fatalf("buildFacts first: %v", err)
	}
	if _, ok := facts.BytecodeBlacklist[bytecodeHash]; !ok {
		t.Fatal("bytecode blacklist fact missing")
	}
	if _, ok := facts.SourcecodeBlacklist[sourcecodeHash]; !ok {
		t.Fatal("sourcecode blacklist fact missing")
	}
	if _, ok := facts.WalletBlacklist[walletAddress]; !ok {
		t.Fatal("wallet blacklist fact missing")
	}

	if _, err := engine.buildFacts(context.Background()); err != nil {
		t.Fatalf("buildFacts second: %v", err)
	}
	if bytecode.listCalls != 1 || sourcecode.listCalls != 1 || wallet.listCalls != 1 {
		t.Fatalf("list calls = bytecode:%d sourcecode:%d wallet:%d, want all 1", bytecode.listCalls, sourcecode.listCalls, wallet.listCalls)
	}

	wallet.version = "wallet-v2"
	if _, err := engine.buildFacts(context.Background()); err != nil {
		t.Fatalf("buildFacts after version change: %v", err)
	}
	if bytecode.listCalls != 2 || sourcecode.listCalls != 2 || wallet.listCalls != 2 {
		t.Fatalf("list calls after version change = bytecode:%d sourcecode:%d wallet:%d, want all 2", bytecode.listCalls, sourcecode.listCalls, wallet.listCalls)
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

func TestProjectPolicyEngineEvaluateAllOnceEvaluatesCachedProjects(t *testing.T) {
	contractA := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	contractB := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	cache := newPolicyReevaluationProjectCache(
		&Project{Meta: ProjectMeta{Contract: contractA}},
		&Project{Meta: ProjectMeta{Contract: contractB}},
	)
	var got []common.Address
	engine := &projectPolicyEngineImpl{
		projectCache: cache,
		rules:        []ProjectPolicyRule{recordingPolicyRule{contracts: &got}},
	}

	if err := engine.EvaluateAllOnce(context.Background()); err != nil {
		t.Fatalf("EvaluateAllOnce: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("evaluated contracts = %v, want 2 contracts", got)
	}
	seen := map[common.Address]bool{}
	for _, contract := range got {
		seen[contract] = true
	}
	if !seen[contractA] || !seen[contractB] {
		t.Fatalf("evaluated contracts = %v, want %s and %s", got, contractA.Hex(), contractB.Hex())
	}
}

func TestProjectPolicyEngineEvaluateRulesUpdatesProjectReport(t *testing.T) {
	contract := common.HexToAddress("0x0000000000000000000000000000000000000201")
	creator := common.HexToAddress("0x0000000000000000000000000000000000000202")
	genesisWallet := common.HexToAddress("0x0000000000000000000000000000000000000203")
	codeBinHash := common.HexToHash("0x2020202020202020202020202020202020202020202020202020202020202020")
	sourceCode := "contract Source {}"
	sourceHash := crypto.Keccak256Hash([]byte(sourceCode))
	project := &Project{
		Meta: ProjectMeta{
			Contract:       contract,
			Creator:        creator,
			SourceCode:     sourceCode,
			SourceCodeHash: sourceHash,
			CodeBinHash:    codeBinHash,
			GenesisWallets: []GenesisWalletMeta{{
				Wallet: genesisWallet,
			}},
			CreatorResult: SimulateResult{
				CanMintFromZeroViaTransferFrom: true,
			},
		},
	}
	cache := newPolicyReevaluationProjectCache(project)
	engine := &projectPolicyEngineImpl{
		projectCache: cache,
		rules: []ProjectPolicyRule{
			walletBlacklistCreatorRule{},
			walletBlacklistGenesisWalletRule{},
			bytecodeBlacklistRule{},
			sourcecodeBlacklistContractRule{},
			simulateMintRiskRule{},
		},
	}

	if err := engine.evaluateRulesForProject(context.Background(), project, ProjectPolicyFacts{
		BytecodeBlacklist: map[common.Hash]struct{}{
			codeBinHash: {},
		},
		SourcecodeBlacklist: map[common.Hash]struct{}{
			sourceHash: {},
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
		!report.IsBlacklistedSourceCode ||
		!report.HasMintRisk {
		t.Fatalf("project report = %+v, want all policy fields true", report)
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
			IsBlacklistedSourceCode:    true,
			HasMintRisk:                true,
		},
	}
	cache := newPolicyReevaluationProjectCache(project)
	engine := &projectPolicyEngineImpl{
		projectCache: cache,
		rules: []ProjectPolicyRule{
			walletBlacklistCreatorRule{},
			walletBlacklistGenesisWalletRule{},
			bytecodeBlacklistRule{},
			sourcecodeBlacklistContractRule{},
			simulateMintRiskRule{},
		},
	}

	if err := engine.evaluateRulesForProject(context.Background(), project, ProjectPolicyFacts{}); err != nil {
		t.Fatalf("evaluateRulesForProject: %v", err)
	}

	want := ProjectReport{IsPolicyEvaluated: true}
	if got := cache.projects[contract].Report; got != want {
		t.Fatalf("project report = %+v, want %+v", got, want)
	}
}
