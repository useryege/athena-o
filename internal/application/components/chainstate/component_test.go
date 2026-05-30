package chainstate

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appcomponents "github.com/useryege/athena/internal/application/components"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func TestComponentHandleRefreshWritesStateAndPublishes(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(21))
	creator := common.BigToAddress(big.NewInt(22))
	store := &chainStateStoreFake{
		baseByContract: map[common.Address]appstore.ProjectBase{
			contract: {Contract: contract, Creator: creator},
		},
	}
	fetcher := &athenaFetcherFake{
		project: athenacontract.AthenaProject{
			TokenContract: contract,
			Token: athenacontract.AthenaToken{
				IsValidERC20: true,
				Name:         "T",
				Symbol:       "T",
			},
		},
	}
	bus := &eventBusFake{}
	component := NewComponent(Options{Store: store, Fetcher: fetcher, Bus: bus})
	if component == nil {
		t.Fatal("component is nil")
	}

	if err := component.handleEvent(ctx, appcomponents.Event{
		Type:     appcomponents.EventProjectRefresh,
		Contract: contract.Hex(),
	}); err != nil {
		t.Fatalf("handle event: %v", err)
	}

	if fetcher.fetchProjectCalls != 1 {
		t.Fatalf("fetch project calls = %d, want 1", fetcher.fetchProjectCalls)
	}
	if store.upsertChainStateCalls != 1 {
		t.Fatalf("upsert chain state calls = %d, want 1", store.upsertChainStateCalls)
	}
	if store.lastComponentStatus != appstore.ProjectComponentStatusSuccess {
		t.Fatalf("last status = %s, want success", store.lastComponentStatus)
	}
	if len(bus.events) != 1 || bus.events[0].Type != appcomponents.EventComponentCompleted {
		t.Fatalf("published events = %#v, want single component completed", bus.events)
	}
}

type chainStateStoreFake struct {
	appstore.Store

	baseByContract        map[common.Address]appstore.ProjectBase
	upsertChainStateCalls int
	lastComponentStatus   string
}

func (s *chainStateStoreFake) GetProjectBaseByContract(_ context.Context, contract common.Address) (*appstore.ProjectBase, error) {
	item, ok := s.baseByContract[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *chainStateStoreFake) UpsertProjectComponentState(_ context.Context, item appstore.ProjectComponentState) error {
	s.lastComponentStatus = item.Status
	return nil
}

func (s *chainStateStoreFake) UpsertProjectChainState(context.Context, appstore.ProjectChainState) error {
	s.upsertChainStateCalls++
	return nil
}

type athenaFetcherFake struct {
	project           athenacontract.AthenaProject
	fetchProjectCalls int
}

func (f *athenaFetcherFake) FetchProject(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaProject, error) {
	f.fetchProjectCalls++
	return f.project, nil
}

func (f *athenaFetcherFake) FetchProjects(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProject, error) {
	return nil, nil
}

func (f *athenaFetcherFake) FetchProjectsWithSimulationState(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProjectWithSimulationState, error) {
	return nil, nil
}

func (f *athenaFetcherFake) FetchSimulationState(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaSimulationState, error) {
	return athenacontract.AthenaSimulationState{}, nil
}

func (f *athenaFetcherFake) FetchSimulationStates(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaSimulationState, error) {
	return nil, nil
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
