package bytecode

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appstore "github.com/useryege/athena/internal/application/store"
	applicationv1alpha1 "github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

func TestComponentCollectWritesState(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(41))
	store := &bytecodeStoreFake{
		baseByContract: map[common.Address]appstore.ProjectBase{
			contract: {ChainID: 1, Contract: contract},
		},
	}
	resolver := &contractSourceResolverFake{
		info: &applicationv1alpha1.ContractSourceInfo{
			IsBytecodeBlacklisted: true,
			CodeBinHash:           "0x2222222222222222222222222222222222222222222222222222222222222222",
		},
	}
	component := NewComponent(Options{
		Store:    store,
		Resolver: resolver,
		ChainID:  1,
	})
	if component == nil {
		t.Fatal("component is nil")
	}

	if err := component.Collect(ctx, 1, contract); err != nil {
		t.Fatalf("collect: %v", err)
	}

	if resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want 1", resolver.calls)
	}
	if store.upsertBytecodeFactCalls != 1 {
		t.Fatalf("upsert bytecode fact calls = %d, want 1", store.upsertBytecodeFactCalls)
	}
	if store.lastComponentStatus != appstore.ProjectComponentStatusSuccess {
		t.Fatalf("last status = %s, want success", store.lastComponentStatus)
	}
}

type bytecodeStoreFake struct {
	appstore.Store

	baseByContract          map[common.Address]appstore.ProjectBase
	upsertBytecodeFactCalls int
	lastComponentStatus     string
}

func (s *bytecodeStoreFake) GetProjectBaseByContract(_ context.Context, _ int64, contract common.Address) (*appstore.ProjectBase, error) {
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

type contractSourceResolverFake struct {
	info  *applicationv1alpha1.ContractSourceInfo
	calls int
}

func (c *contractSourceResolverFake) GetContractSourceInfo(context.Context, *applicationpkg.GetContractSourceInfoRequest) (*applicationv1alpha1.ContractSourceInfo, error) {
	c.calls++
	return c.info, nil
}
