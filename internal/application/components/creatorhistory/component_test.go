package creatorhistory

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appcomponents "github.com/useryege/athena/internal/application/components"
	appstore "github.com/useryege/athena/internal/application/store"
)

func TestComponentRefreshEventRerunsAfterSuccess(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(1))
	creator := common.BigToAddress(big.NewInt(2))
	historical := common.BigToAddress(big.NewInt(3))
	store := &refreshComponentStoreFake{
		baseByContract: map[common.Address]appstore.ProjectBase{
			contract: {
				Contract:    contract,
				Creator:     creator,
				BlockNumber: 10,
				TxIndex:     2,
			},
		},
		metasByCreatorBefore: map[common.Address][]appstore.ProjectMeta{
			creator: {{
				Contract: historical,
			}},
		},
		componentStates: map[string]appstore.ProjectComponentState{
			componentKey(contract, appstore.ProjectComponentCreatorHistory): {
				ProjectContract: contract,
				Component:       appstore.ProjectComponentCreatorHistory,
				Status:          appstore.ProjectComponentStatusSuccess,
				LastSuccessAt:   time.Now().UTC(),
			},
		},
	}
	component := NewComponent(Options{Store: store})
	if component == nil {
		t.Fatal("creator history component is nil")
	}

	if err := component.handleEvent(ctx, appcomponents.Event{
		Type:     appcomponents.EventProjectRefresh,
		Contract: contract.Hex(),
	}); err != nil {
		t.Fatalf("handle refresh: %v", err)
	}

	if got := store.replaceCreatorHistoryCalls; got != 1 {
		t.Fatalf("replace creator history calls = %d, want 1", got)
	}
	if got := len(store.lastCreatorHistoryItems); got != 1 {
		t.Fatalf("creator history items len = %d, want 1", got)
	}
	if got := store.lastCreatorHistoryItems[0].HistoricalProjectContract; got != historical {
		t.Fatalf("historical contract = %s, want %s", got.Hex(), historical.Hex())
	}
}

type refreshComponentStoreFake struct {
	appstore.Store

	baseByContract       map[common.Address]appstore.ProjectBase
	componentStates      map[string]appstore.ProjectComponentState
	metasByCreatorBefore map[common.Address][]appstore.ProjectMeta

	replaceCreatorHistoryCalls int
	lastCreatorHistoryItems    []appstore.ProjectCreatorHistoricalProject
}

func (s *refreshComponentStoreFake) GetProjectBaseByContract(_ context.Context, contract common.Address) (*appstore.ProjectBase, error) {
	item, ok := s.baseByContract[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *refreshComponentStoreFake) GetProjectComponentState(_ context.Context, contract common.Address, component string) (*appstore.ProjectComponentState, error) {
	item, ok := s.componentStates[componentKey(contract, component)]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *refreshComponentStoreFake) UpsertProjectComponentState(_ context.Context, item appstore.ProjectComponentState) error {
	if s.componentStates == nil {
		s.componentStates = map[string]appstore.ProjectComponentState{}
	}
	s.componentStates[componentKey(item.ProjectContract, item.Component)] = item
	return nil
}

func (s *refreshComponentStoreFake) ListProjectMetasByCreatorBefore(_ context.Context, creator common.Address, _ uint64, _ uint64) ([]appstore.ProjectMeta, error) {
	items := s.metasByCreatorBefore[creator]
	return append([]appstore.ProjectMeta(nil), items...), nil
}

func (s *refreshComponentStoreFake) ReplaceProjectCreatorHistoricalProjects(_ context.Context, _ common.Address, items []appstore.ProjectCreatorHistoricalProject) error {
	s.replaceCreatorHistoryCalls++
	s.lastCreatorHistoryItems = append([]appstore.ProjectCreatorHistoricalProject(nil), items...)
	return nil
}

func componentKey(contract common.Address, component string) string {
	return contract.Hex() + "|" + component
}
