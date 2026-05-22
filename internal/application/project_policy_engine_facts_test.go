package application

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
)

type sourceBlacklistVersionedFake struct {
	fields       []string
	version      string
	listCalls    int
	versionCalls int
}

func (f *sourceBlacklistVersionedFake) List(context.Context) ([]string, error) {
	f.listCalls++
	return append([]string(nil), f.fields...), nil
}

func (f *sourceBlacklistVersionedFake) Version(context.Context) (string, error) {
	f.versionCalls++
	return f.version, nil
}

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
	source := &sourceBlacklistVersionedFake{
		fields:  []string{"owner"},
		version: "source-v1",
	}
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
		sourceBlacklist:     source,
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
	if source.listCalls != 1 || bytecode.listCalls != 1 || sourcecode.listCalls != 1 || wallet.listCalls != 1 {
		t.Fatalf("list calls = source:%d bytecode:%d sourcecode:%d wallet:%d, want all 1", source.listCalls, bytecode.listCalls, sourcecode.listCalls, wallet.listCalls)
	}

	wallet.version = "wallet-v2"
	if _, err := engine.buildFacts(context.Background()); err != nil {
		t.Fatalf("buildFacts after version change: %v", err)
	}
	if source.listCalls != 2 || bytecode.listCalls != 2 || sourcecode.listCalls != 2 || wallet.listCalls != 2 {
		t.Fatalf("list calls after version change = source:%d bytecode:%d sourcecode:%d wallet:%d, want all 2", source.listCalls, bytecode.listCalls, sourcecode.listCalls, wallet.listCalls)
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

func TestProjectPolicyEngineEvaluateAllOnceEvaluatesActiveAndArchivedProjects(t *testing.T) {
	active := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	archived := common.HexToAddress("0x00000000000000000000000000000000000000a2")
	cache := newPolicyReevaluationProjectCache(
		&Project{Meta: ProjectMeta{Contract: active}},
		&Project{Meta: ProjectMeta{Contract: archived, IsArchived: true}},
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
	if !seen[active] || !seen[archived] {
		t.Fatalf("evaluated contracts = %v, want active %s and archived %s", got, active.Hex(), archived.Hex())
	}
}
