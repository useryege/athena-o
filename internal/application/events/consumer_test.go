package events

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/model"
)

type processorStoreFake struct {
	candidates        map[string]model.DiscoveredProjectCandidate
	collections       []model.ProjectRef
	collectionReasons []string
	projectsByPair    map[string][]model.Project
}

func (f *processorStoreFake) UpsertProjectCandidateAndEnqueueQualification(_ context.Context, candidate model.DiscoveredProjectCandidate) error {
	if f.candidates == nil {
		f.candidates = map[string]model.DiscoveredProjectCandidate{}
	}
	f.candidates[projectKey(candidate.ChainID, candidate.Contract)] = candidate
	return nil
}

func (f *processorStoreFake) EnqueueProjectCollection(_ context.Context, ref model.ProjectRef, reason string) error {
	f.collections = append(f.collections, ref)
	f.collectionReasons = append(f.collectionReasons, reason)
	return nil
}

func (f *processorStoreFake) ListProjectMetasByPairAddresses(_ context.Context, chainID int64, pairs []common.Address) ([]model.Project, error) {
	if f.projectsByPair == nil || len(pairs) == 0 {
		return nil, nil
	}
	return append([]model.Project(nil), f.projectsByPair[projectKey(chainID, pairs[0])]...), nil
}

func TestProcessorContractCreatedIsIdempotentByChainAndContract(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	creator := common.HexToAddress("0x2000000000000000000000000000000000000002")
	wethPair := common.HexToAddress("0x3000000000000000000000000000000000000003")
	usdtPair := common.HexToAddress("0x4000000000000000000000000000000000000004")
	codeHash := common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555")
	store := &processorStoreFake{}
	processor := NewProcessor(store)
	envelope := testEnvelope(t, EventTypeContractCreated, 56, ContractCreatedPayload{
		Contract:    contract.Hex(),
		Creator:     creator.Hex(),
		TxHash:      "0xabc",
		CodeHash:    codeHash.Hex(),
		WethPair:    wethPair.Hex(),
		UsdtPair:    usdtPair.Hex(),
		BlockNumber: 123,
		BlockTime:   456,
		TxIndex:     7,
	})

	if err := processor.ProcessEnvelope(context.Background(), envelope); err != nil {
		t.Fatalf("ProcessEnvelope first: %v", err)
	}
	if err := processor.ProcessEnvelope(context.Background(), envelope); err != nil {
		t.Fatalf("ProcessEnvelope duplicate: %v", err)
	}
	if len(store.candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(store.candidates))
	}
	item := store.candidates[projectKey(56, contract)]
	if item.ChainID != 56 || item.Contract != contract || item.Creator != creator || item.BlockNumber != 123 || item.BlockTime != 456 || item.TxIndex != 7 {
		t.Fatalf("candidate = %#v, want decoded event fields", item)
	}
	if item.WethPair != wethPair || item.UsdtPair != usdtPair {
		t.Fatalf("candidate pairs = %s/%s, want %s/%s", item.WethPair.Hex(), item.UsdtPair.Hex(), wethPair.Hex(), usdtPair.Hex())
	}
	if item.CodeHash != codeHash {
		t.Fatalf("candidate code hash = %s, want %s", item.CodeHash.Hex(), codeHash.Hex())
	}
	if item.Source != model.ProjectDiscoverySourceFollowHeads {
		t.Fatalf("source = %q, want follow_heads", item.Source)
	}
}

func TestProcessorContractCreatedKeepsSameContractOnDifferentChainsSeparate(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	store := &processorStoreFake{}
	processor := NewProcessor(store)

	for _, chainID := range []int64{1, 56} {
		envelope := testEnvelope(t, EventTypeContractCreated, chainID, ContractCreatedPayload{
			Contract:    contract.Hex(),
			CodeHash:    common.HexToHash("0x5555555555555555555555555555555555555555555555555555555555555555").Hex(),
			BlockNumber: chainID,
		})
		if err := processor.ProcessEnvelope(context.Background(), envelope); err != nil {
			t.Fatalf("ProcessEnvelope chain %d: %v", chainID, err)
		}
	}
	if len(store.candidates) != 2 {
		t.Fatalf("candidate count = %d, want 2 chain-scoped records", len(store.candidates))
	}
}

func TestProcessorContractCreatedRequiresCodeHash(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	processor := NewProcessor(&processorStoreFake{})

	for _, tc := range []struct {
		name     string
		codeHash string
	}{
		{name: "missing", codeHash: ""},
		{name: "invalid", codeHash: "0x1234"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			envelope := testEnvelope(t, EventTypeContractCreated, 56, ContractCreatedPayload{
				Contract:    contract.Hex(),
				CodeHash:    tc.codeHash,
				BlockNumber: 1,
			})
			if err := processor.ProcessEnvelope(context.Background(), envelope); err == nil {
				t.Fatalf("ProcessEnvelope error = nil, want invalid code hash")
			}
		})
	}
}

func TestProcessorDexSwapSchedulesKnownProjects(t *testing.T) {
	pair := common.HexToAddress("0x3000000000000000000000000000000000000003")
	token0 := common.HexToAddress("0x5000000000000000000000000000000000000005")
	token1 := common.HexToAddress("0x6000000000000000000000000000000000000006")
	project := common.HexToAddress("0x4000000000000000000000000000000000000004")
	store := &processorStoreFake{
		projectsByPair: map[string][]model.Project{
			projectKey(1, pair): {{ChainID: 1, Contract: project, WethPair: pair}},
		},
	}
	processor := NewProcessor(store)

	envelope := testEnvelope(t, EventTypeDexSwap, 1, DexSwapPayload{
		Pair:        pair.Hex(),
		Token0:      token0.Hex(),
		Token1:      token1.Hex(),
		BlockNumber: 100,
	})
	if err := processor.ProcessEnvelope(context.Background(), envelope); err != nil {
		t.Fatalf("ProcessEnvelope: %v", err)
	}
	if len(store.collections) != 1 {
		t.Fatalf("collections = %d, want 1", len(store.collections))
	}
	if store.collections[0] != (model.ProjectRef{ChainID: 1, Contract: project}) {
		t.Fatalf("collection ref = %#v", store.collections[0])
	}
	if store.collectionReasons[0] != ReasonDexSwap {
		t.Fatalf("reason = %q, want dex_swap", store.collectionReasons[0])
	}
}

func TestProcessorDexSwapRequiresPairTokens(t *testing.T) {
	pair := common.HexToAddress("0x3000000000000000000000000000000000000003")
	token0 := common.HexToAddress("0x5000000000000000000000000000000000000005")
	token1 := common.HexToAddress("0x6000000000000000000000000000000000000006")
	processor := NewProcessor(&processorStoreFake{})

	for _, tc := range []struct {
		name    string
		payload DexSwapPayload
	}{
		{name: "missing token0", payload: DexSwapPayload{Pair: pair.Hex(), Token1: token1.Hex()}},
		{name: "missing token1", payload: DexSwapPayload{Pair: pair.Hex(), Token0: token0.Hex()}},
		{name: "zero token0", payload: DexSwapPayload{Pair: pair.Hex(), Token0: common.Address{}.Hex(), Token1: token1.Hex()}},
		{name: "zero token1", payload: DexSwapPayload{Pair: pair.Hex(), Token0: token0.Hex(), Token1: common.Address{}.Hex()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			envelope := testEnvelope(t, EventTypeDexSwap, 1, tc.payload)
			if err := processor.ProcessEnvelope(context.Background(), envelope); err == nil {
				t.Fatalf("ProcessEnvelope error = nil, want invalid dex swap payload")
			}
		})
	}
}

func testEnvelope(t *testing.T, eventType string, chainID int64, payload any) Envelope {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return Envelope{
		EventID:       "test",
		EventType:     eventType,
		SchemaVersion: SchemaVersionV1,
		ChainID:       chainID,
		OccurredAt:    time.Unix(1, 0).UTC(),
		Payload:       raw,
	}
}

func projectKey(chainID int64, address common.Address) string {
	return ContractCreatedKey(chainID, address.Hex())
}
