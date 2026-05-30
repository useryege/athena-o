package api

import (
	"context"
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	appcache "github.com/useryege/athena/internal/application/cache"
	appcomponents "github.com/useryege/athena/internal/application/components"
	"github.com/useryege/athena/internal/application/model"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServiceListProjectsUsesSQLAndSchedulesRefreshForCurrentPage(t *testing.T) {
	ctx := context.Background()
	cache := newComponentCacheForTest(t)
	bus := newProjectRefreshEventBusFake()

	cacheOnlyContract := common.BigToAddress(big.NewInt(100))
	if err := cache.SetBase(ctx, appstore.ProjectBase{Contract: cacheOnlyContract, BlockNumber: 999}); err != nil {
		t.Fatalf("seed cache base: %v", err)
	}

	first := common.BigToAddress(big.NewInt(101))
	second := common.BigToAddress(big.NewInt(102))
	store := &projectQueryStoreFake{
		bases: []appstore.ProjectBase{
			{BlockNumber: 1, BlockTime: 100, Contract: first, Creator: common.BigToAddress(big.NewInt(1)), TxIndex: 0},
			{BlockNumber: 2, BlockTime: 200, Contract: second, Creator: common.BigToAddress(big.NewInt(2)), TxIndex: 0},
		},
		reports: map[common.Address]appstore.ProjectReportState{
			second: {ProjectContract: second, Report: appstore.ProjectReport{IsReportEvaluated: true, HasMintRisk: true}},
		},
	}
	service := &Service{
		dbStore:               store,
		componentCache:        cache,
		componentEventBus:     bus,
		projectRefreshLastRun: map[common.Address]time.Time{},
		projectRefreshWindow:  30 * time.Second,
	}

	resp, err := service.ListProjects(ctx, &applicationpkg.ListProjectsRequest{Page: 2, PageSize: 1})
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if store.listBasePageCalls != 1 {
		t.Fatalf("list base page calls = %d, want 1", store.listBasePageCalls)
	}
	if got := len(resp.GetItems()); got != 1 {
		t.Fatalf("items len = %d, want 1", got)
	}
	if got := resp.GetItems()[0].Contract; got != second.Hex() {
		t.Fatalf("item contract = %s, want %s", got, second.Hex())
	}
	if resp.GetItems()[0].HasMintRisk != true {
		t.Fatalf("has mint risk = %t, want true", resp.GetItems()[0].HasMintRisk)
	}

	events := bus.waitForCount(t, 1, time.Second)
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if got := events[0].ProjectContract(); got != second {
		t.Fatalf("refresh contract = %s, want %s", got.Hex(), second.Hex())
	}
}

func TestServiceListProjectsRefreshThrottle(t *testing.T) {
	ctx := context.Background()
	cache := newComponentCacheForTest(t)
	bus := newProjectRefreshEventBusFake()
	contract := common.BigToAddress(big.NewInt(111))
	store := &projectQueryStoreFake{
		bases: []appstore.ProjectBase{{BlockNumber: 1, Contract: contract, Creator: common.BigToAddress(big.NewInt(2))}},
	}
	service := &Service{
		dbStore:               store,
		componentCache:        cache,
		componentEventBus:     bus,
		projectRefreshLastRun: map[common.Address]time.Time{},
		projectRefreshWindow:  30 * time.Second,
	}

	if _, err := service.ListProjects(ctx, &applicationpkg.ListProjectsRequest{Page: 1, PageSize: 20}); err != nil {
		t.Fatalf("list first: %v", err)
	}
	if _, err := service.ListProjects(ctx, &applicationpkg.ListProjectsRequest{Page: 1, PageSize: 20}); err != nil {
		t.Fatalf("list second: %v", err)
	}
	_ = bus.waitForCount(t, 1, time.Second)
	time.Sleep(80 * time.Millisecond)
	if got := bus.count(); got != 1 {
		t.Fatalf("refresh event count = %d, want 1", got)
	}
}

func TestServiceGetProjectUsesComponentCache(t *testing.T) {
	ctx := context.Background()
	cache := newComponentCacheForTest(t)
	bus := newProjectRefreshEventBusFake()
	contract := common.BigToAddress(big.NewInt(201))
	base := appstore.ProjectBase{
		BlockNumber: 88,
		BlockTime:   999,
		Contract:    contract,
		Creator:     common.BigToAddress(big.NewInt(5)),
		TxHash:      common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
		TxIndex:     3,
		CreatedAt:   time.Date(2026, 5, 30, 10, 0, 0, 0, time.UTC),
	}
	if err := cache.SetBase(ctx, base); err != nil {
		t.Fatalf("set base: %v", err)
	}
	if err := cache.SetChainState(ctx, appstore.ProjectChainState{
		ProjectContract: contract,
		FetchedAt:       time.Date(2026, 5, 30, 10, 1, 0, 0, time.UTC),
		WethPair:        common.BigToAddress(big.NewInt(9)),
		UsdtPair:        common.BigToAddress(big.NewInt(10)),
		ChainState: athenacontract.AthenaProject{
			Token: athenacontract.AthenaToken{
				Name:         "Cached",
				Symbol:       "CCH",
				IsValidERC20: true,
			},
		},
	}); err != nil {
		t.Fatalf("set chain state: %v", err)
	}
	if err := cache.SetSimulation(ctx, appstore.ProjectSimulationResult{
		ProjectContract: contract,
		Result:          appstore.SimulateResult{CanMintFromZeroViaTransferFrom: true},
	}); err != nil {
		t.Fatalf("set simulation: %v", err)
	}
	if err := cache.SetReport(ctx, appstore.ProjectReportState{
		ProjectContract: contract,
		Report:          appstore.ProjectReport{IsReportEvaluated: true, HasMintRisk: true},
	}); err != nil {
		t.Fatalf("set report: %v", err)
	}
	if err := cache.SetGenesisWallets(ctx, contract, []appstore.ProjectGenesisWallet{{
		ProjectContract: contract,
		Wallet:          common.BigToAddress(big.NewInt(33)),
		NetAmount:       big.NewInt(100),
		RatioBPS:        100,
		RankIndex:       0,
	}}); err != nil {
		t.Fatalf("set genesis wallets: %v", err)
	}
	if err := cache.SetCreatorHistory(ctx, contract, []appstore.ProjectCreatorHistoricalProject{{
		ProjectContract:           contract,
		HistoricalProjectContract: common.BigToAddress(big.NewInt(44)),
		RankIndex:                 0,
	}}); err != nil {
		t.Fatalf("set creator history: %v", err)
	}
	if err := cache.SetAveDetail(ctx, contract, appstore.ProjectAveDetail{
		Status:    1,
		FetchedAt: time.Date(2026, 5, 30, 10, 2, 0, 0, time.UTC),
		Token: appstore.ProjectAveTokenDetail{
			Symbol: "AVE",
		},
	}); err != nil {
		t.Fatalf("set ave detail: %v", err)
	}
	if err := cache.SetComponentState(ctx, appstore.ProjectComponentState{
		ProjectContract: contract,
		Component:       appstore.ProjectComponentGenesisWallet,
		Status:          appstore.ProjectComponentStatusSuccess,
		LastSuccessAt:   time.Date(2026, 5, 30, 10, 3, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("set genesis state: %v", err)
	}
	if err := cache.SetComponentState(ctx, appstore.ProjectComponentState{
		ProjectContract: contract,
		Component:       appstore.ProjectComponentCreatorHistory,
		Status:          appstore.ProjectComponentStatusSuccess,
		LastSuccessAt:   time.Date(2026, 5, 30, 10, 4, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("set creator state: %v", err)
	}

	store := &projectQueryStoreFake{}
	service := &Service{
		dbStore:               store,
		componentCache:        cache,
		componentEventBus:     bus,
		projectRefreshLastRun: map[common.Address]time.Time{},
		projectRefreshWindow:  30 * time.Second,
	}

	resp, err := service.GetProject(ctx, &applicationpkg.GetProjectRequest{Contract: contract.Hex()})
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if got := resp.GetItem().Meta.Token.Name; got != "Cached" {
		t.Fatalf("token name = %q, want Cached", got)
	}
	if got := len(resp.GetItem().Meta.GenesisWallets); got != 1 {
		t.Fatalf("genesis wallets len = %d, want 1", got)
	}
	if got := len(resp.GetItem().Meta.CreatorHistoricalProjects); got != 1 {
		t.Fatalf("creator history len = %d, want 1", got)
	}
	if store.getBaseCalls != 0 {
		t.Fatalf("db get base calls = %d, want 0", store.getBaseCalls)
	}
	time.Sleep(50 * time.Millisecond)
	if got := bus.count(); got != 0 {
		t.Fatalf("refresh event count = %d, want 0", got)
	}
}

func TestServiceGetProjectCacheMissLoadsDBBackfillsCacheAndRefreshes(t *testing.T) {
	ctx := context.Background()
	cache := newComponentCacheForTest(t)
	bus := newProjectRefreshEventBusFake()
	contract := common.BigToAddress(big.NewInt(301))
	wallet := common.BigToAddress(big.NewInt(302))
	previous := common.BigToAddress(big.NewInt(303))
	store := &projectQueryStoreFake{
		baseByContract: map[common.Address]appstore.ProjectBase{
			contract: {
				BlockNumber: 123,
				BlockTime:   888,
				Contract:    contract,
				Creator:     common.BigToAddress(big.NewInt(9)),
				TxHash:      common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
				TxIndex:     7,
				CreatedAt:   time.Date(2026, 5, 30, 11, 0, 0, 0, time.UTC),
			},
		},
		chainStates: map[common.Address]appstore.ProjectChainState{
			contract: {
				ProjectContract: contract,
				FetchedAt:       time.Date(2026, 5, 30, 11, 1, 0, 0, time.UTC),
				ChainState: athenacontract.AthenaProject{
					Token: athenacontract.AthenaToken{
						Name:         "FromDB",
						Symbol:       "FDB",
						IsValidERC20: true,
					},
				},
			},
		},
		simulations: map[common.Address]appstore.ProjectSimulationResult{
			contract: {ProjectContract: contract, Result: appstore.SimulateResult{CanMintFromZeroViaTransferFrom: true}},
		},
		reportsByContract: map[common.Address]*appstore.ProjectReportState{
			contract: {ProjectContract: contract, Report: appstore.ProjectReport{IsReportEvaluated: true, HasMintRisk: true}},
		},
		aveDetails: map[common.Address]appstore.ProjectAveDetail{
			contract: {Status: 1, Token: appstore.ProjectAveTokenDetail{Symbol: "AVE"}},
		},
		genesisWallets: map[common.Address][]appstore.ProjectGenesisWallet{
			contract: {{ProjectContract: contract, Wallet: wallet, NetAmount: big.NewInt(200), RatioBPS: 50}},
		},
		creatorHistory: map[common.Address][]appstore.ProjectCreatorHistoricalProject{
			contract: {{ProjectContract: contract, HistoricalProjectContract: previous}},
		},
		componentStates: map[string]appstore.ProjectComponentState{
			componentStateKey(contract, appstore.ProjectComponentGenesisWallet): {
				ProjectContract: contract,
				Component:       appstore.ProjectComponentGenesisWallet,
				Status:          appstore.ProjectComponentStatusSuccess,
				LastSuccessAt:   time.Date(2026, 5, 30, 11, 2, 0, 0, time.UTC),
			},
			componentStateKey(contract, appstore.ProjectComponentCreatorHistory): {
				ProjectContract: contract,
				Component:       appstore.ProjectComponentCreatorHistory,
				Status:          appstore.ProjectComponentStatusSuccess,
				LastSuccessAt:   time.Date(2026, 5, 30, 11, 3, 0, 0, time.UTC),
			},
		},
	}
	service := &Service{
		dbStore:               store,
		componentCache:        cache,
		componentEventBus:     bus,
		projectRefreshLastRun: map[common.Address]time.Time{},
		projectRefreshWindow:  30 * time.Second,
	}

	resp, err := service.GetProject(ctx, &applicationpkg.GetProjectRequest{Contract: contract.Hex()})
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if got := resp.GetItem().Meta.Token.Name; got != "FromDB" {
		t.Fatalf("token name = %q, want FromDB", got)
	}
	if got := resp.GetItem().Meta.CreatorHistoricalProjects; len(got) != 1 || got[0] != previous.Hex() {
		t.Fatalf("creator history = %v, want %s", got, previous.Hex())
	}
	if got := resp.GetItem().Meta.GenesisWallets; len(got) != 1 || got[0].Wallet != wallet.Hex() {
		t.Fatalf("genesis wallets = %+v, want %s", got, wallet.Hex())
	}

	base, ok, err := cache.GetBase(ctx, contract)
	if err != nil {
		t.Fatalf("cache get base: %v", err)
	}
	if !ok || base == nil {
		t.Fatal("expected base backfilled into component cache")
	}
	events := bus.waitForCount(t, 1, time.Second)
	if len(events) != 1 {
		t.Fatalf("events len = %d, want 1", len(events))
	}
	if got := events[0].Source; got != model.ProjectDiscoverySourceFollowHeads {
		t.Fatalf("refresh source = %s, want %s", got, model.ProjectDiscoverySourceFollowHeads)
	}
}

func TestServiceGetProjectCacheMissNotFound(t *testing.T) {
	ctx := context.Background()
	service := &Service{
		dbStore:               &projectQueryStoreFake{},
		componentCache:        nil,
		projectRefreshLastRun: map[common.Address]time.Time{},
		projectRefreshWindow:  30 * time.Second,
	}
	contract := common.BigToAddress(big.NewInt(401))

	_, err := service.GetProject(ctx, &applicationpkg.GetProjectRequest{Contract: contract.Hex()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status = %s, want NotFound (err %v)", status.Code(err), err)
	}
}

type projectQueryStoreFake struct {
	appstore.Store

	bases             []appstore.ProjectBase
	baseByContract    map[common.Address]appstore.ProjectBase
	reports           map[common.Address]appstore.ProjectReportState
	reportsByContract map[common.Address]*appstore.ProjectReportState
	chainStates       map[common.Address]appstore.ProjectChainState
	simulations       map[common.Address]appstore.ProjectSimulationResult
	aveDetails        map[common.Address]appstore.ProjectAveDetail
	genesisWallets    map[common.Address][]appstore.ProjectGenesisWallet
	creatorHistory    map[common.Address][]appstore.ProjectCreatorHistoricalProject
	componentStates   map[string]appstore.ProjectComponentState

	listBasePageCalls int
	getBaseCalls      int
}

func (s *projectQueryStoreFake) ListProjectBasesPage(_ context.Context, page int32, pageSize int32) ([]appstore.ProjectBase, int64, int32, int32, error) {
	s.listBasePageCalls++
	page, pageSize = normalizeCachePage(page, pageSize)
	items := append([]appstore.ProjectBase(nil), s.bases...)
	total := int64(len(items))
	start := int64(page-1) * int64(pageSize)
	if start >= total {
		return nil, total, page, pageSize, nil
	}
	stop := start + int64(pageSize)
	if stop > total {
		stop = total
	}
	return items[start:stop], total, page, pageSize, nil
}

func (s *projectQueryStoreFake) GetProjectBaseByContract(_ context.Context, contract common.Address) (*appstore.ProjectBase, error) {
	s.getBaseCalls++
	if item, ok := s.baseByContract[contract]; ok {
		base := item
		return &base, nil
	}
	for _, item := range s.bases {
		if item.Contract == contract {
			base := item
			return &base, nil
		}
	}
	return nil, nil
}

func (s *projectQueryStoreFake) ListProjectReportStatesByContracts(_ context.Context, contracts []common.Address) (map[common.Address]appstore.ProjectReportState, error) {
	result := map[common.Address]appstore.ProjectReportState{}
	for _, contract := range contracts {
		if item, ok := s.reports[contract]; ok {
			result[contract] = item
			continue
		}
		if item, ok := s.reportsByContract[contract]; ok && item != nil {
			result[contract] = *item
		}
	}
	return result, nil
}

func (s *projectQueryStoreFake) GetProjectReportState(_ context.Context, contract common.Address) (*appstore.ProjectReportState, error) {
	if item, ok := s.reportsByContract[contract]; ok {
		if item == nil {
			return nil, nil
		}
		copy := *item
		return &copy, nil
	}
	if item, ok := s.reports[contract]; ok {
		copy := item
		return &copy, nil
	}
	return nil, nil
}

func (s *projectQueryStoreFake) GetProjectChainState(_ context.Context, contract common.Address) (*appstore.ProjectChainState, error) {
	item, ok := s.chainStates[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *projectQueryStoreFake) GetProjectSimulationResult(_ context.Context, contract common.Address) (*appstore.ProjectSimulationResult, error) {
	item, ok := s.simulations[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *projectQueryStoreFake) GetProjectAveDetail(_ context.Context, contract common.Address) (*appstore.ProjectAveDetail, error) {
	item, ok := s.aveDetails[contract]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

func (s *projectQueryStoreFake) ListProjectGenesisWalletsByContract(_ context.Context, contract common.Address) ([]appstore.ProjectGenesisWallet, error) {
	items := s.genesisWallets[contract]
	return append([]appstore.ProjectGenesisWallet(nil), items...), nil
}

func (s *projectQueryStoreFake) ListProjectCreatorHistoricalProjectsByContract(_ context.Context, contract common.Address) ([]appstore.ProjectCreatorHistoricalProject, error) {
	items := s.creatorHistory[contract]
	return append([]appstore.ProjectCreatorHistoricalProject(nil), items...), nil
}

func (s *projectQueryStoreFake) GetProjectComponentState(_ context.Context, contract common.Address, component string) (*appstore.ProjectComponentState, error) {
	item, ok := s.componentStates[componentStateKey(contract, component)]
	if !ok {
		return nil, nil
	}
	copy := item
	return &copy, nil
}

type projectRefreshEventBusFake struct {
	mu     sync.Mutex
	events []appcomponents.Event
	ch     chan struct{}
}

func newProjectRefreshEventBusFake() *projectRefreshEventBusFake {
	return &projectRefreshEventBusFake{ch: make(chan struct{}, 32)}
}

func (b *projectRefreshEventBusFake) Publish(_ context.Context, event appcomponents.Event) error {
	b.mu.Lock()
	b.events = append(b.events, event)
	b.mu.Unlock()
	select {
	case b.ch <- struct{}{}:
	default:
	}
	return nil
}

func (b *projectRefreshEventBusFake) Subscribe(context.Context, string, string, appcomponents.Handler) error {
	return nil
}

func (b *projectRefreshEventBusFake) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.events)
}

func (b *projectRefreshEventBusFake) waitForCount(t *testing.T, expected int, timeout time.Duration) []appcomponents.Event {
	t.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		b.mu.Lock()
		count := len(b.events)
		if count >= expected {
			events := append([]appcomponents.Event(nil), b.events...)
			b.mu.Unlock()
			return events
		}
		b.mu.Unlock()

		select {
		case <-b.ch:
		case <-deadline.C:
			t.Fatalf("timed out waiting for %d events, got %d", expected, count)
		}
	}
}

func newComponentCacheForTest(t *testing.T) appcache.ProjectComponentCache {
	t.Helper()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return appcache.NewProjectComponentCache(redisport.NewGoRedisAdapter(client))
}

func componentStateKey(contract common.Address, component string) string {
	return contract.Hex() + "|" + component
}
