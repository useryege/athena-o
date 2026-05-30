package initializer

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appcomponents "github.com/useryege/athena/internal/application/components"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
)

func TestComponentIntakeCandidatesPublishesAndWritesState(t *testing.T) {
	contract := common.BigToAddress(big.NewInt(10))
	creator := common.BigToAddress(big.NewInt(11))
	store := &initializerStoreFake{}
	bus := &eventBusFake{}
	component := NewComponent(Options{Store: store, Bus: bus})

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
	if len(bus.events) != 2 {
		t.Fatalf("published events = %d, want 2", len(bus.events))
	}
	if bus.events[0].Type != appcomponents.EventProjectInitialized {
		t.Fatalf("first event = %s, want %s", bus.events[0].Type, appcomponents.EventProjectInitialized)
	}
	if bus.events[1].Type != appcomponents.EventComponentCompleted {
		t.Fatalf("second event = %s, want %s", bus.events[1].Type, appcomponents.EventComponentCompleted)
	}
}

type initializerStoreFake struct {
	appstore.Store

	savedBases            []appstore.ProjectBase
	componentStatusByName map[string]string
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
