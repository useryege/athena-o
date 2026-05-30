package bytecode

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appcomponents "github.com/useryege/athena/internal/application/components"
	appstore "github.com/useryege/athena/internal/application/store"
	solidityapiclient "github.com/useryege/athena/internal/solidity/apiclient"
	applicationv1alpha1 "github.com/useryege/athena/pkg/apis/application/v1alpha1"
	utilio "github.com/useryege/athena/util/io"
	"google.golang.org/grpc"
)

func TestComponentHandleRefreshWritesStateAndPublishes(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(41))
	store := &bytecodeStoreFake{
		baseByContract: map[common.Address]appstore.ProjectBase{
			contract: {Contract: contract},
		},
	}
	client := &solidityClientFake{
		info: &applicationv1alpha1.ContractSourceInfo{
			IsBytecodeBlacklisted: true,
			CodeBinHash:           "0x2222222222222222222222222222222222222222222222222222222222222222",
		},
	}
	clients := &solidityClientsetFake{client: client}
	bus := &eventBusFake{}
	component := NewComponent(Options{
		Store:   store,
		Clients: clients,
		ChainID: 1,
		Bus:     bus,
	})
	if component == nil {
		t.Fatal("component is nil")
	}

	if err := component.handleEvent(ctx, appcomponents.Event{
		Type:     appcomponents.EventProjectRefresh,
		Contract: contract.Hex(),
	}); err != nil {
		t.Fatalf("handle event: %v", err)
	}

	if clients.newClientCalls != 1 {
		t.Fatalf("new client calls = %d, want 1", clients.newClientCalls)
	}
	if store.upsertBytecodeFactCalls != 1 {
		t.Fatalf("upsert bytecode fact calls = %d, want 1", store.upsertBytecodeFactCalls)
	}
	if store.lastComponentStatus != appstore.ProjectComponentStatusSuccess {
		t.Fatalf("last status = %s, want success", store.lastComponentStatus)
	}
	if len(bus.events) != 1 || bus.events[0].Type != appcomponents.EventComponentCompleted {
		t.Fatalf("published events = %#v, want single component completed", bus.events)
	}
}

type bytecodeStoreFake struct {
	appstore.Store

	baseByContract          map[common.Address]appstore.ProjectBase
	upsertBytecodeFactCalls int
	lastComponentStatus     string
}

func (s *bytecodeStoreFake) GetProjectBaseByContract(_ context.Context, contract common.Address) (*appstore.ProjectBase, error) {
	item, ok := s.baseByContract[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *bytecodeStoreFake) UpsertProjectComponentState(_ context.Context, item appstore.ProjectComponentState) error {
	s.lastComponentStatus = item.Status
	return nil
}

func (s *bytecodeStoreFake) UpsertProjectBytecodeFact(context.Context, appstore.ProjectBytecodeFact) error {
	s.upsertBytecodeFactCalls++
	return nil
}

type solidityClientsetFake struct {
	client         solidityapiclient.SolidityServiceClient
	newClientCalls int
}

func (c *solidityClientsetFake) NewSolidityServiceClient() (utilio.Closer, solidityapiclient.SolidityServiceClient, error) {
	c.newClientCalls++
	return utilio.NopCloser, c.client, nil
}

type solidityClientFake struct {
	solidityapiclient.SolidityServiceClient
	info *applicationv1alpha1.ContractSourceInfo
}

func (c *solidityClientFake) GetContractSourceInfo(context.Context, *solidityapiclient.GetContractSourceInfoRequest, ...grpc.CallOption) (*applicationv1alpha1.ContractSourceInfo, error) {
	return c.info, nil
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
