package chainstate

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

func TestComponentCollectWritesState(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(21))
	creator := common.BigToAddress(big.NewInt(22))
	store := &chainStateStoreFake{
		baseByContract: map[common.Address]appstore.ProjectRecord{
			contract: {ChainID: 1, Contract: contract, Creator: creator},
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
	component := NewComponent(Options{ChainID: 1, Store: store, Fetcher: fetcher})
	if component == nil {
		t.Fatal("component is nil")
	}

	if err := component.Collect(ctx, 1, contract); err != nil {
		t.Fatalf("collect: %v", err)
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
}

type chainStateStoreFake struct {
	appstore.Store

	baseByContract        map[common.Address]appstore.ProjectRecord
	upsertChainStateCalls int
	lastComponentStatus   string
}

func (s *chainStateStoreFake) GetProjectByContract(_ context.Context, _ int64, contract common.Address) (*appstore.ProjectRecord, error) {
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
