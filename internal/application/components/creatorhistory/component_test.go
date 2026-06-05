package creatorhistory

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
)

func TestComponentCollectRerunsAfterSuccess(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(1))
	creator := common.BigToAddress(big.NewInt(2))
	historical := common.BigToAddress(big.NewInt(3))
	store := &refreshComponentStoreFake{
		baseByContract: map[common.Address]appstore.ProjectBase{
			contract: {
				ChainID:     56,
				Contract:    contract,
				Creator:     creator,
				BlockNumber: 10,
				TxIndex:     2,
			},
		},
		metasByCreatorBefore: map[common.Address][]appstore.ProjectMeta{
			creator: {{
				ChainID:  56,
				Contract: historical,
			}},
		},
		componentStates: map[string]appstore.ProjectComponentState{
			componentKey(contract, appstore.ProjectComponentCreatorHistory): {
				ChainID:         56,
				ProjectContract: contract,
				Component:       appstore.ProjectComponentCreatorHistory,
				Status:          appstore.ProjectComponentStatusSuccess,
				LastSuccessAt:   time.Now().UTC(),
			},
		},
	}
	component := NewComponent(Options{ChainID: 56, Store: store})
	if component == nil {
		t.Fatal("creator history component is nil")
	}

	if err := component.Collect(ctx, 56, contract); err != nil {
		t.Fatalf("collect: %v", err)
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

func (s *refreshComponentStoreFake) GetProjectBaseByContract(_ context.Context, _ int64, contract common.Address) (*appstore.ProjectBase, error) {
	item, ok := s.baseByContract[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *refreshComponentStoreFake) GetProjectComponentState(_ context.Context, _ int64, contract common.Address, component string) (*appstore.ProjectComponentState, error) {
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

func (s *refreshComponentStoreFake) ListProjectMetasByCreatorBefore(_ context.Context, _ int64, creator common.Address, _ uint64, _ uint64) ([]appstore.ProjectMeta, error) {
	items := s.metasByCreatorBefore[creator]
	return append([]appstore.ProjectMeta(nil), items...), nil
}

func (s *refreshComponentStoreFake) ReplaceProjectCreatorHistoricalProjects(_ context.Context, _ int64, _ common.Address, items []appstore.ProjectCreatorHistoricalProject) error {
	s.replaceCreatorHistoryCalls++
	s.lastCreatorHistoryItems = append([]appstore.ProjectCreatorHistoricalProject(nil), items...)
	return nil
}

func componentKey(contract common.Address, component string) string {
	return contract.Hex() + "|" + component
}
