package simulation

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appcomponents "github.com/useryege/athena/internal/application/components"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func TestComponentIgnoresIrrelevantCompletedEvent(t *testing.T) {
	component := NewComponent(Options{
		Store:     &simulationStoreFake{},
		Fetcher:   &athenaFetcherFake{},
		Simulator: &projectSimulatorFake{},
	})

	err := component.handleEvent(context.Background(), appcomponents.Event{
		Type:      appcomponents.EventComponentCompleted,
		Component: appstore.ProjectComponentBytecodeFact,
		Contract:  common.BigToAddress(big.NewInt(1)).Hex(),
	})
	if err != nil {
		t.Fatalf("handle irrelevant event: %v", err)
	}
}

func TestComponentHandleCompletedEventWritesStateAndPublishes(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(31))
	creator := common.BigToAddress(big.NewInt(32))
	wethPair := common.BigToAddress(big.NewInt(33))
	usdtPair := common.BigToAddress(big.NewInt(34))
	store := &simulationStoreFake{
		baseByContract: map[common.Address]appstore.ProjectBase{
			contract: {Contract: contract, Creator: creator},
		},
		chainStateByContract: map[common.Address]appstore.ProjectChainState{
			contract: {ProjectContract: contract, WethPair: wethPair, UsdtPair: usdtPair},
		},
	}
	fetcher := &athenaFetcherFake{}
	simulator := &projectSimulatorFake{}
	bus := &eventBusFake{}
	component := NewComponent(Options{
		Store:     store,
		Fetcher:   fetcher,
		Simulator: simulator,
		Bus:       bus,
	})

	if err := component.handleEvent(ctx, appcomponents.Event{
		Type:      appcomponents.EventComponentCompleted,
		Component: appstore.ProjectComponentChainState,
		Contract:  contract.Hex(),
	}); err != nil {
		t.Fatalf("handle event: %v", err)
	}

	if fetcher.fetchSimulationCalls != 1 {
		t.Fatalf("fetch simulation calls = %d, want 1", fetcher.fetchSimulationCalls)
	}
	if simulator.simulateCalls != 1 {
		t.Fatalf("simulate calls = %d, want 1", simulator.simulateCalls)
	}
	if store.upsertSimulationCalls != 1 {
		t.Fatalf("upsert simulation calls = %d, want 1", store.upsertSimulationCalls)
	}
	if store.lastComponentStatus != appstore.ProjectComponentStatusSuccess {
		t.Fatalf("last status = %s, want success", store.lastComponentStatus)
	}
	if len(bus.events) != 1 || bus.events[0].Type != appcomponents.EventComponentCompleted {
		t.Fatalf("published events = %#v, want single component completed", bus.events)
	}
}

type simulationStoreFake struct {
	appstore.Store

	baseByContract       map[common.Address]appstore.ProjectBase
	chainStateByContract map[common.Address]appstore.ProjectChainState

	lastComponentStatus   string
	upsertSimulationCalls int
}

func (s *simulationStoreFake) GetProjectBaseByContract(_ context.Context, contract common.Address) (*appstore.ProjectBase, error) {
	item, ok := s.baseByContract[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *simulationStoreFake) GetProjectChainState(_ context.Context, contract common.Address) (*appstore.ProjectChainState, error) {
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

type projectSimulatorFake struct {
	simulateCalls int
}

func (s *projectSimulatorFake) SimulatePrimary(context.Context, common.Address, common.Address, common.Address, common.Address, athenacontract.AthenaSimulationState) (model.SimulateResult, error) {
	s.simulateCalls++
	return model.SimulateResult{}, nil
}

type eventBusFake struct {
	events []appcomponents.Event
}

func (b *eventBusFake) Publish(_ context.Context, event appcomponents.Event) error {
	b.events = append(b.events, event)
	return nil
}

func (b *eventBusFake) Subscribe(context.Context, string, string, appcomponents.Handler) error {
	return nil
}
