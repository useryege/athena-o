package simulation

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func TestComponentCollectIgnoresEmptyContract(t *testing.T) {
	component := NewComponent(Options{
		ChainID:    56,
		Store:      &simulationStoreFake{},
		Fetcher:    &athenaFetcherFake{},
		NodeClient: &simulationNodeClientFake{},
	})

	if err := component.Collect(context.Background(), 56, common.Address{}); err != nil {
		t.Fatalf("collect empty contract: %v", err)
	}
}

func TestComponentCollectWritesState(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(31))
	creator := common.BigToAddress(big.NewInt(32))
	wethPair := common.BigToAddress(big.NewInt(33))
	usdtPair := common.BigToAddress(big.NewInt(34))
	store := &simulationStoreFake{
		baseByContract: map[common.Address]appstore.ProjectBase{
			contract: {ChainID: 56, Contract: contract, Creator: creator},
		},
		chainStateByContract: map[common.Address]appstore.ProjectChainState{
			contract: {ChainID: 56, ProjectContract: contract, WethPair: wethPair, UsdtPair: usdtPair},
		},
	}
	fetcher := &athenaFetcherFake{}
	nodeClient := &simulationNodeClientFake{}
	component := NewComponent(Options{
		ChainID:    56,
		Store:      store,
		Fetcher:    fetcher,
		NodeClient: nodeClient,
	})

	if err := component.Collect(ctx, 56, contract); err != nil {
		t.Fatalf("collect: %v", err)
	}

	if fetcher.fetchSimulationCalls != 1 {
		t.Fatalf("fetch simulation calls = %d, want 1", fetcher.fetchSimulationCalls)
	}
	if nodeClient.batchCallCalls != 1 {
		t.Fatalf("batch call calls = %d, want 1", nodeClient.batchCallCalls)
	}
	if store.upsertSimulationCalls != 1 {
		t.Fatalf("upsert simulation calls = %d, want 1", store.upsertSimulationCalls)
	}
	if store.lastComponentStatus != appstore.ProjectComponentStatusSuccess {
		t.Fatalf("last status = %s, want success", store.lastComponentStatus)
	}
}

func TestComponentMarksSuccessWhenPairMissingWithoutPersistingSimulation(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(41))
	creator := common.BigToAddress(big.NewInt(42))
	store := &simulationStoreFake{
		baseByContract: map[common.Address]appstore.ProjectBase{
			contract: {ChainID: 56, Contract: contract, Creator: creator},
		},
		chainStateByContract: map[common.Address]appstore.ProjectChainState{
			contract: {ChainID: 56, ProjectContract: contract, WethPair: common.BigToAddress(big.NewInt(43))},
		},
	}
	fetcher := &athenaFetcherFake{}
	nodeClient := &simulationNodeClientFake{}
	component := NewComponent(Options{
		ChainID:    56,
		Store:      store,
		Fetcher:    fetcher,
		NodeClient: nodeClient,
	})

	if err := component.Collect(ctx, 56, contract); err != nil {
		t.Fatalf("collect: %v", err)
	}

	if fetcher.fetchSimulationCalls != 1 {
		t.Fatalf("fetch simulation calls = %d, want 1", fetcher.fetchSimulationCalls)
	}
	if nodeClient.batchCallCalls != 0 {
		t.Fatalf("batch call calls = %d, want 0", nodeClient.batchCallCalls)
	}
	if store.upsertSimulationCalls != 0 {
		t.Fatalf("upsert simulation calls = %d, want 0", store.upsertSimulationCalls)
	}
	if store.lastComponentStatus != appstore.ProjectComponentStatusSuccess {
		t.Fatalf("last status = %s, want success", store.lastComponentStatus)
	}
}

func TestComponentMarksFailedWhenBatchCallFails(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(51))
	creator := common.BigToAddress(big.NewInt(52))
	wethPair := common.BigToAddress(big.NewInt(53))
	usdtPair := common.BigToAddress(big.NewInt(54))
	store := &simulationStoreFake{
		baseByContract: map[common.Address]appstore.ProjectBase{
			contract: {ChainID: 56, Contract: contract, Creator: creator},
		},
		chainStateByContract: map[common.Address]appstore.ProjectChainState{
			contract: {ChainID: 56, ProjectContract: contract, WethPair: wethPair, UsdtPair: usdtPair},
		},
	}
	nodeClient := &simulationNodeClientFake{batchCallErr: errors.New("rpc down")}
	component := NewComponent(Options{
		ChainID:    56,
		Store:      store,
		Fetcher:    &athenaFetcherFake{},
		NodeClient: nodeClient,
	})

	if err := component.Collect(ctx, 56, contract); err == nil {
		t.Fatal("collect error = nil, want batch call error")
	}

	if nodeClient.batchCallCalls != 1 {
		t.Fatalf("batch call calls = %d, want 1", nodeClient.batchCallCalls)
	}
	if store.upsertSimulationCalls != 0 {
		t.Fatalf("upsert simulation calls = %d, want 0", store.upsertSimulationCalls)
	}
	if store.lastComponentStatus != appstore.ProjectComponentStatusFailed {
		t.Fatalf("last status = %s, want failed", store.lastComponentStatus)
	}
}

type simulationStoreFake struct {
	appstore.Store

	baseByContract       map[common.Address]appstore.ProjectBase
	chainStateByContract map[common.Address]appstore.ProjectChainState

	lastComponentStatus   string
	upsertSimulationCalls int
}

func (s *simulationStoreFake) GetProjectBaseByContract(_ context.Context, _ int64, contract common.Address) (*appstore.ProjectBase, error) {
	item, ok := s.baseByContract[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *simulationStoreFake) GetProjectChainState(_ context.Context, _ int64, contract common.Address) (*appstore.ProjectChainState, error) {
	item, ok := s.chainStateByContract[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *simulationStoreFake) UpsertProjectComponentState(_ context.Context, item appstore.ProjectComponentState) error {
	s.lastComponentStatus = item.Status
	return nil
}

func (s *simulationStoreFake) UpsertProjectSimulationResult(context.Context, appstore.ProjectSimulationResult) error {
	s.upsertSimulationCalls++
	return nil
}

type athenaFetcherFake struct {
	fetchSimulationCalls int
}

func (f *athenaFetcherFake) FetchProject(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaProject, error) {
	return athenacontract.AthenaProject{}, nil
}

func (f *athenaFetcherFake) FetchProjects(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProject, error) {
	return nil, nil
}

func (f *athenaFetcherFake) FetchProjectsWithSimulationState(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProjectWithSimulationState, error) {
	return nil, nil
}

func (f *athenaFetcherFake) FetchSimulationState(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaSimulationState, error) {
	f.fetchSimulationCalls++
	return athenacontract.AthenaSimulationState{}, nil
}

func (f *athenaFetcherFake) FetchSimulationStates(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaSimulationState, error) {
	return nil, nil
}

type simulationNodeClientFake struct {
	batchCallCalls int
	batchCallErr   error
	batchElems     []rpc.BatchElem
}

func (s *simulationNodeClientFake) BatchCallContext(_ context.Context, b []rpc.BatchElem) error {
	s.batchCallCalls++
	s.batchElems = append([]rpc.BatchElem(nil), b...)
	if s.batchCallErr != nil {
		return s.batchCallErr
	}
	return nil
}
