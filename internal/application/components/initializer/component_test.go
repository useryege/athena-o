package initializer

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
)

func TestComponentIntakeCandidatesWritesBaseAndState(t *testing.T) {
	contract := common.BigToAddress(big.NewInt(10))
	creator := common.BigToAddress(big.NewInt(11))
	store := &initializerStoreFake{}
	component := NewComponent(Options{Store: store})

	err := component.IntakeCandidates(context.Background(), []model.DiscoveredProjectCandidate{
		{Contract: contract, Creator: creator},
	})
	if err != nil {
		t.Fatalf("intake candidates: %v", err)
	}

	if len(store.savedBases) != 1 {
		t.Fatalf("saved bases = %d, want 1", len(store.savedBases))
	}
	if store.componentStatusByName[appstore.ProjectComponentInitializer] != appstore.ProjectComponentStatusSuccess {
		t.Fatalf("initializer status = %s, want success", store.componentStatusByName[appstore.ProjectComponentInitializer])
	}
}

func TestComponentScheduleProjectsEnqueuesCollection(t *testing.T) {
	contract := common.BigToAddress(big.NewInt(12))
	store := &initializerStoreFake{}
	component := NewComponent(Options{Store: store, ChainID: 56})

	if err := component.ScheduleProjects(context.Background(), []model.DiscoveredProjectCandidate{{Contract: contract}}); err != nil {
		t.Fatalf("schedule projects: %v", err)
	}
	if len(store.enqueuedCollections) != 1 {
		t.Fatalf("enqueued collections = %d, want 1", len(store.enqueuedCollections))
	}
	if got := store.enqueuedCollections[0]; got.ChainID != 56 || got.Contract != contract || store.enqueuedReasons[0] != string(model.ProjectDiscoverySourcePairSwap) {
		t.Fatalf("enqueued = %#v reason=%q, want chain 56 pair_swap", got, store.enqueuedReasons[0])
	}
}

type initializerStoreFake struct {
	appstore.Store

	savedBases            []appstore.ProjectBase
	componentStatusByName map[string]string
	enqueuedCollections   []model.ProjectRef
	enqueuedReasons       []string
}

func (s *initializerStoreFake) SaveProjectBase(_ context.Context, base appstore.ProjectBase) error {
	s.savedBases = append(s.savedBases, base)
	return nil
}

func (s *initializerStoreFake) UpsertProjectComponentState(_ context.Context, item appstore.ProjectComponentState) error {
	if s.componentStatusByName == nil {
		s.componentStatusByName = map[string]string{}
	}
	s.componentStatusByName[item.Component] = item.Status
	return nil
}

func (s *initializerStoreFake) EnqueueProjectCollection(_ context.Context, ref model.ProjectRef, reason string) error {
	s.enqueuedCollections = append(s.enqueuedCollections, ref)
	s.enqueuedReasons = append(s.enqueuedReasons, reason)
	return nil
}
