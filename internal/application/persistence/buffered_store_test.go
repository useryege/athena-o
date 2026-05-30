package persistence

import (
	"context"
	"errors"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
)

type bufferedStoreDBFake struct {
	appstore.Store

	err error

	calls       []string
	bases       map[common.Address]appstore.ProjectBase
	chainStates map[common.Address]appstore.ProjectChainState
	reports     map[common.Address]appstore.ProjectReportState
	genesis     map[common.Address][]appstore.ProjectGenesisWallet
	events      []appstore.ProjectEventLog
}

func newBufferedStoreTest(t *testing.T, db *bufferedStoreDBFake) (*redis.Client, *RedisBufferedStore) {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	store, err := NewRedisBufferedStore(db, redisport.NewGoRedisAdapter(client))
	if err != nil {
		t.Fatalf("new buffered store: %v", err)
	}
	return client, store
}

func (s *bufferedStoreDBFake) SaveProjectBase(_ context.Context, item appstore.ProjectBase) error {
	s.calls = append(s.calls, "base")
	if s.err != nil {
		return s.err
	}
	if s.bases == nil {
		s.bases = map[common.Address]appstore.ProjectBase{}
	}
	s.bases[item.Contract] = item
	return nil
}

func (s *bufferedStoreDBFake) GetProjectBaseByContract(_ context.Context, contract common.Address) (*appstore.ProjectBase, error) {
	item, ok := s.bases[contract]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (s *bufferedStoreDBFake) UpsertProjectChainState(_ context.Context, item appstore.ProjectChainState) error {
	s.calls = append(s.calls, "chain")
	if s.err != nil {
		return s.err
	}
	if s.chainStates == nil {
		s.chainStates = map[common.Address]appstore.ProjectChainState{}
	}
	s.chainStates[item.ProjectContract] = item
	return nil
}

func (s *bufferedStoreDBFake) GetProjectChainState(_ context.Context, contract common.Address) (*appstore.ProjectChainState, error) {
	item, ok := s.chainStates[contract]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (s *bufferedStoreDBFake) UpsertProjectReportState(_ context.Context, item appstore.ProjectReportState) error {
	s.calls = append(s.calls, "report")
	if s.err != nil {
		return s.err
	}
	if s.reports == nil {
		s.reports = map[common.Address]appstore.ProjectReportState{}
	}
	s.reports[item.ProjectContract] = item
	return nil
}

func (s *bufferedStoreDBFake) GetProjectReportState(_ context.Context, contract common.Address) (*appstore.ProjectReportState, error) {
	item, ok := s.reports[contract]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (s *bufferedStoreDBFake) ListProjectReportStatesByContracts(_ context.Context, _ []common.Address) (map[common.Address]appstore.ProjectReportState, error) {
	result := map[common.Address]appstore.ProjectReportState{}
	for contract, item := range s.reports {
		result[contract] = item
	}
	return result, nil
}

func (s *bufferedStoreDBFake) ReplaceProjectGenesisWallets(_ context.Context, contract common.Address, items []appstore.ProjectGenesisWallet) error {
	s.calls = append(s.calls, "genesis")
	if s.err != nil {
		return s.err
	}
	if s.genesis == nil {
		s.genesis = map[common.Address][]appstore.ProjectGenesisWallet{}
	}
	s.genesis[contract] = append([]appstore.ProjectGenesisWallet(nil), items...)
	return nil
}

func (s *bufferedStoreDBFake) AddProjectEventLog(_ context.Context, item appstore.ProjectEventLog) error {
	s.calls = append(s.calls, "event_log")
	if s.err != nil {
		return s.err
	}
	s.events = append(s.events, item)
	return nil
}

func TestRedisBufferedStoreWritesBaseToRedisOnly(t *testing.T) {
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	db := &bufferedStoreDBFake{}
	_, store := newBufferedStoreTest(t, db)

	if err := store.SaveProjectBase(ctx, appstore.ProjectBase{
		BlockNumber: 10,
		BlockTime:   20,
		Contract:    contract,
		Creator:     common.HexToAddress("0x1000000000000000000000000000000000000002"),
		TxHash:      common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		TxIndex:     3,
	}); err != nil {
		t.Fatalf("save base: %v", err)
	}
	if len(db.calls) != 0 {
		t.Fatalf("db calls = %v, want none before flush", db.calls)
	}
	base, err := store.GetProjectBaseByContract(ctx, contract)
	if err != nil {
		t.Fatalf("get base: %v", err)
	}
	if base == nil || base.Contract != contract || base.BlockNumber != 10 {
		t.Fatalf("base = %+v, want redis-staged base", base)
	}
	dirty, err := store.dirtyMembers(ctx, bufferedBaseKind)
	if err != nil {
		t.Fatalf("dirty members: %v", err)
	}
	if !reflect.DeepEqual(dirty, []string{contract.Hex()}) {
		t.Fatalf("dirty = %v, want %s", dirty, contract.Hex())
	}
}

func TestRedisBufferedStoreReadFallsBackAndCaches(t *testing.T) {
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	db := &bufferedStoreDBFake{bases: map[common.Address]appstore.ProjectBase{
		contract: {BlockNumber: 11, Contract: contract},
	}}
	_, store := newBufferedStoreTest(t, db)

	base, err := store.GetProjectBaseByContract(ctx, contract)
	if err != nil {
		t.Fatalf("get base: %v", err)
	}
	if base == nil || base.BlockNumber != 11 {
		t.Fatalf("base = %+v, want db base", base)
	}
	db.bases = nil
	base, err = store.GetProjectBaseByContract(ctx, contract)
	if err != nil {
		t.Fatalf("get cached base: %v", err)
	}
	if base == nil || base.BlockNumber != 11 {
		t.Fatalf("cached base = %+v, want cached db base", base)
	}
}

func TestRedisBufferedFlusherFlushesBaseBeforeDependentsAndClearsDirty(t *testing.T) {
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	db := &bufferedStoreDBFake{}
	_, store := newBufferedStoreTest(t, db)

	if err := store.SaveProjectBase(ctx, appstore.ProjectBase{BlockNumber: 10, Contract: contract, TxHash: common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}); err != nil {
		t.Fatalf("save base: %v", err)
	}
	if err := store.UpsertProjectChainState(ctx, appstore.ProjectChainState{ProjectContract: contract, WethPair: common.HexToAddress("0x1000000000000000000000000000000000000002"), FetchedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("upsert chain: %v", err)
	}
	if err := store.ReplaceProjectGenesisWallets(ctx, contract, []appstore.ProjectGenesisWallet{{
		ProjectContract: contract,
		Wallet:          common.HexToAddress("0x1000000000000000000000000000000000000003"),
		NetAmount:       big.NewInt(1),
		TotalSupply:     big.NewInt(1),
		SourceTxHash:    common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
	}}); err != nil {
		t.Fatalf("replace genesis: %v", err)
	}

	if err := NewFlusher(store, time.Hour).Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	wantCalls := []string{"base", "chain", "genesis"}
	if !reflect.DeepEqual(db.calls, wantCalls) {
		t.Fatalf("calls = %v, want %v", db.calls, wantCalls)
	}
	for _, kind := range []string{bufferedBaseKind, bufferedChainStateKind, bufferedGenesisKind} {
		dirty, err := store.dirtyMembers(ctx, kind)
		if err != nil {
			t.Fatalf("dirty %s: %v", kind, err)
		}
		if len(dirty) != 0 {
			t.Fatalf("dirty %s = %v, want empty", kind, dirty)
		}
	}
}

func TestRedisBufferedFlusherKeepsDirtyOnDBError(t *testing.T) {
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	db := &bufferedStoreDBFake{err: errors.New("db down")}
	_, store := newBufferedStoreTest(t, db)

	if err := store.SaveProjectBase(ctx, appstore.ProjectBase{BlockNumber: 10, Contract: contract, TxHash: common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")}); err != nil {
		t.Fatalf("save base: %v", err)
	}
	if err := NewFlusher(store, time.Hour).Flush(ctx); err == nil {
		t.Fatal("expected flush error")
	}
	dirty, err := store.dirtyMembers(ctx, bufferedBaseKind)
	if err != nil {
		t.Fatalf("dirty members: %v", err)
	}
	if !reflect.DeepEqual(dirty, []string{contract.Hex()}) {
		t.Fatalf("dirty = %v, want retained base", dirty)
	}
}

func TestRedisBufferedStoreFlushesEventLogs(t *testing.T) {
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	db := &bufferedStoreDBFake{}
	_, store := newBufferedStoreTest(t, db)

	item := appstore.ProjectEventLog{
		Contract:       contract,
		EventType:      5,
		OccurredAt:     time.Date(2026, 5, 30, 1, 2, 3, 0, time.UTC),
		Message:        "matched",
		Payload:        `{"rule":"x"}`,
		IdempotencyKey: "report:x",
	}
	if err := store.AddProjectEventLog(ctx, item); err != nil {
		t.Fatalf("add event log: %v", err)
	}
	if len(db.events) != 0 {
		t.Fatalf("db events = %d, want none before flush", len(db.events))
	}
	if err := NewFlusher(store, time.Hour).Flush(ctx); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if len(db.events) != 1 || db.events[0].IdempotencyKey != item.IdempotencyKey {
		t.Fatalf("events = %+v, want flushed event", db.events)
	}
}
