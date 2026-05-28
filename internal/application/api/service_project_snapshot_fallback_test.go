package api

import (
	"context"
	"errors"
	"math/big"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/redis/go-redis/v9"
	applicationpkg "github.com/useryege/athena/internal/application/apiclient"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServiceGetProjectUsesCachedSnapshot(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
	contract := common.BigToAddress(big.NewInt(201))
	if err := cache.SetProject(ctx, &Project{Meta: ProjectMeta{
		Contract: contract,
		ChainState: athenacontract.AthenaProject{Token: athenacontract.AthenaToken{
			Name:         "Cached",
			Symbol:       "CCH",
			IsValidERC20: true,
		}},
	}}); err != nil {
		t.Fatalf("set cache: %v", err)
	}
	store := &projectSnapshotFallbackStore{getErr: errors.New("store should not be called")}
	fetcher := &projectSnapshotFetcherFake{err: errors.New("fetcher should not be called")}
	service := &Service{projectCache: cache, store: store, athenaFetcher: fetcher}

	resp, err := service.GetProject(ctx, &applicationpkg.GetProjectRequest{Contract: contract.Hex()})
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if resp.GetItem().Meta.Token.Name != "Cached" {
		t.Fatalf("token name = %q, want Cached", resp.GetItem().Meta.Token.Name)
	}
	if store.getCalls != 0 {
		t.Fatalf("store get calls = %d, want 0", store.getCalls)
	}
	if fetcher.fetchProjectsCalls != 0 {
		t.Fatalf("fetcher calls = %d, want 0", fetcher.fetchProjectsCalls)
	}
}

func TestServiceGetProjectFallbackLoadsDBFetchesChainAndRecaches(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
	contract := common.BigToAddress(big.NewInt(202))
	creator := common.BigToAddress(big.NewInt(303))
	wallet := common.BigToAddress(big.NewInt(404))
	previous := common.BigToAddress(big.NewInt(505))
	fetchedAt := time.Date(2026, 5, 23, 1, 2, 3, 0, time.UTC)
	store := &projectSnapshotFallbackStore{
		metas: []appstore.ProjectMeta{{
			BlockNumber: 123,
			BlockTime:   456,
			Contract:    contract,
			Creator:     creator,
			TxHash:      common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			TxIndex:     7,
			Report: appstore.ProjectReport{
				IsPolicyEvaluated:          true,
				IsBlacklistedCreatorWallet: true,
				HasMintRisk:                true,
			},
			GenesisWalletsFetchedAt:            fetchedAt,
			CreatorHistoricalProjectsFetchedAt: fetchedAt,
		}},
		genesisWallets: map[common.Address][]appstore.ProjectGenesisWallet{
			contract: {{
				ProjectContract: contract,
				Wallet:          wallet,
				NetAmount:       big.NewInt(1000),
				RatioBPS:        2500,
				RankIndex:       1,
			}},
		},
		creatorHistoricalProjects: map[common.Address][]appstore.ProjectCreatorHistoricalProject{
			contract: {{
				ProjectContract:           contract,
				HistoricalProjectContract: previous,
				RankIndex:                 0,
			}},
		},
	}
	fetcher := &projectSnapshotFetcherFake{projects: []athenacontract.AthenaProject{{
		TokenContract: contract,
		Token: athenacontract.AthenaToken{
			Name:         "Fetched",
			Symbol:       "FTD",
			IsValidERC20: true,
		},
	}}}
	service := &Service{projectCache: cache, store: store, athenaFetcher: fetcher}

	resp, err := service.GetProject(ctx, &applicationpkg.GetProjectRequest{Contract: contract.Hex()})
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if resp.GetItem().Meta.Token.Name != "Fetched" {
		t.Fatalf("token name = %q, want Fetched", resp.GetItem().Meta.Token.Name)
	}
	if len(resp.GetItem().Meta.GenesisWallets) != 1 || resp.GetItem().Meta.GenesisWallets[0].Wallet != wallet.Hex() {
		t.Fatalf("genesis wallets = %+v, want %s", resp.GetItem().Meta.GenesisWallets, wallet.Hex())
	}
	if len(resp.GetItem().Meta.CreatorHistoricalProjects) != 1 || resp.GetItem().Meta.CreatorHistoricalProjects[0] != previous.Hex() {
		t.Fatalf("creator historical projects = %v, want %s", resp.GetItem().Meta.CreatorHistoricalProjects, previous.Hex())
	}

	cached, ok, err := cache.GetProject(ctx, contract)
	if err != nil {
		t.Fatalf("get recached project: %v", err)
	}
	if !ok || cached.Meta.ChainState.Token.Name != "Fetched" {
		t.Fatalf("cached project = %+v ok %t, want fetched token", cached, ok)
	}
	wantReport := ProjectReport{
		IsPolicyEvaluated:          true,
		IsBlacklistedCreatorWallet: true,
		HasMintRisk:                true,
	}
	if cached.Report != wantReport {
		t.Fatalf("cached report = %+v, want %+v", cached.Report, wantReport)
	}
}

func TestServiceGetProjectFallbackNotFound(t *testing.T) {
	ctx := context.Background()
	service := &Service{store: &projectSnapshotFallbackStore{}}
	contract := common.BigToAddress(big.NewInt(203))

	_, err := service.GetProject(ctx, &applicationpkg.GetProjectRequest{Contract: contract.Hex()})
	if status.Code(err) != codes.NotFound {
		t.Fatalf("status = %s, want NotFound (err %v)", status.Code(err), err)
	}
}

func TestServiceGetProjectFallbackPropagatesFetcherError(t *testing.T) {
	ctx := context.Background()
	contract := common.BigToAddress(big.NewInt(204))
	wantErr := errors.New("fetch failed")
	service := &Service{
		store: &projectSnapshotFallbackStore{metas: []appstore.ProjectMeta{{
			Contract: contract,
			Creator:  common.BigToAddress(big.NewInt(1)),
		}}},
		athenaFetcher: &projectSnapshotFetcherFake{err: wantErr},
	}

	_, err := service.GetProject(ctx, &applicationpkg.GetProjectRequest{Contract: contract.Hex()})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}

func TestServiceListProjectsFallbackLoadsDBPageAndRecaches(t *testing.T) {
	ctx := context.Background()
	mini := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mini.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	cache := NewProjectSnapshotCache(redisport.NewGoRedisAdapter(client))
	first := common.BigToAddress(big.NewInt(205))
	second := common.BigToAddress(big.NewInt(206))
	store := &projectSnapshotFallbackStore{metas: []appstore.ProjectMeta{
		{BlockNumber: 1, Contract: first, Creator: common.BigToAddress(big.NewInt(1))},
		{BlockNumber: 2, Contract: second, Creator: common.BigToAddress(big.NewInt(2))},
	}}
	fetcher := &projectSnapshotFetcherFake{projects: []athenacontract.AthenaProject{{
		TokenContract: second,
		Token: athenacontract.AthenaToken{
			Name:         "Second",
			Symbol:       "SND",
			IsValidERC20: true,
		},
	}}}
	service := &Service{projectCache: cache, store: store, athenaFetcher: fetcher}

	resp, err := service.ListProjects(ctx, &applicationpkg.ListProjectsRequest{Page: 2, PageSize: 1})
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if resp.GetTotal() != 2 || resp.GetPage() != 2 || resp.GetPageSize() != 1 {
		t.Fatalf("pagination = total %d page %d pageSize %d, want 2/2/1", resp.GetTotal(), resp.GetPage(), resp.GetPageSize())
	}
	if len(resp.GetItems()) != 1 || resp.GetItems()[0].Contract != second.String() || resp.GetItems()[0].Name != "Second" {
		t.Fatalf("items = %+v, want second project", resp.GetItems())
	}
	if _, ok, err := cache.GetProject(ctx, second); err != nil || !ok {
		t.Fatalf("recached second ok=%t err=%v, want ok", ok, err)
	}
}

type projectSnapshotFetcherFake struct {
	projects           []athenacontract.AthenaProject
	err                error
	fetchProjectsCalls int
}

func (f *projectSnapshotFetcherFake) FetchProject(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaProject, error) {
	if f.err != nil {
		return athenacontract.AthenaProject{}, f.err
	}
	if len(f.projects) == 0 {
		return athenacontract.AthenaProject{}, nil
	}
	return f.projects[0], nil
}

func (f *projectSnapshotFetcherFake) FetchProjects(_ context.Context, queries []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProject, error) {
	f.fetchProjectsCalls++
	if f.err != nil {
		return nil, f.err
	}
	if len(f.projects) > 0 {
		return append([]athenacontract.AthenaProject(nil), f.projects...), nil
	}
	projects := make([]athenacontract.AthenaProject, 0, len(queries))
	for _, query := range queries {
		projects = append(projects, athenacontract.AthenaProject{
			TokenContract: query.TokenContract,
			Token: athenacontract.AthenaToken{
				IsValidERC20: true,
			},
		})
	}
	return projects, nil
}

func (f *projectSnapshotFetcherFake) FetchProjectsWithSimulationState(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaProjectWithSimulationState, error) {
	return nil, f.err
}

func (f *projectSnapshotFetcherFake) FetchSimulationState(context.Context, athenacontract.AthenaProjectQuery) (athenacontract.AthenaSimulationState, error) {
	return athenacontract.AthenaSimulationState{}, f.err
}

func (f *projectSnapshotFetcherFake) FetchSimulationStates(context.Context, []athenacontract.AthenaProjectQuery) ([]athenacontract.AthenaSimulationState, error) {
	return nil, f.err
}

type projectSnapshotFallbackStore struct {
	metas                     []appstore.ProjectMeta
	genesisWallets            map[common.Address][]appstore.ProjectGenesisWallet
	creatorHistoricalProjects map[common.Address][]appstore.ProjectCreatorHistoricalProject
	getErr                    error
	listErr                   error
	getCalls                  int
}

func (s *projectSnapshotFallbackStore) SaveProjectMeta(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (s *projectSnapshotFallbackStore) GetMaxProjectBlockNumber(context.Context) (uint64, bool, error) {
	if len(s.metas) == 0 {
		return 0, false, nil
	}
	maxBlock := s.metas[0].BlockNumber
	for _, meta := range s.metas[1:] {
		if meta.BlockNumber > maxBlock {
			maxBlock = meta.BlockNumber
		}
	}
	return maxBlock, true, nil
}

func (s *projectSnapshotFallbackStore) ListProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return append([]appstore.ProjectMeta(nil), s.metas...), nil
}

func (s *projectSnapshotFallbackStore) ListAllProjectMetas(ctx context.Context) ([]appstore.ProjectMeta, error) {
	return s.ListProjectMetas(ctx)
}

func (s *projectSnapshotFallbackStore) ListProjectMetasByPairAddresses(context.Context, []common.Address) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (s *projectSnapshotFallbackStore) ListProjectMetasByCodeBinHash(context.Context, common.Hash) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (s *projectSnapshotFallbackStore) UpdateProjectSourceCode(context.Context, common.Address, string, string) error {
	return nil
}

func (s *projectSnapshotFallbackStore) UpdateProjectCodeBinHash(context.Context, common.Address, common.Hash) error {
	return nil
}

func (s *projectSnapshotFallbackStore) UpdateProjectSourceQualityReport(context.Context, common.Address, string, string) error {
	return nil
}

func (s *projectSnapshotFallbackStore) UpsertProjectAveDetail(context.Context, common.Address, appstore.ProjectAveDetail) error {
	return nil
}

func (s *projectSnapshotFallbackStore) UpdateProjectCreatorResult(context.Context, common.Address, appstore.SimulateResult) error {
	return nil
}

func (s *projectSnapshotFallbackStore) UpdateProjectReport(context.Context, common.Address, appstore.ProjectReport) error {
	return nil
}

func (s *projectSnapshotFallbackStore) ListProjectMetasByCreator(context.Context, common.Address) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (s *projectSnapshotFallbackStore) ListProjectMetasByCreatorBefore(context.Context, common.Address, uint64, uint64) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (s *projectSnapshotFallbackStore) GetProjectMetaByContract(_ context.Context, contract common.Address) (*appstore.ProjectMeta, error) {
	s.getCalls++
	if s.getErr != nil {
		return nil, s.getErr
	}
	for _, meta := range s.metas {
		if meta.Contract == contract {
			metaCopy := meta
			return &metaCopy, nil
		}
	}
	return nil, nil
}

func (s *projectSnapshotFallbackStore) ReplaceProjectGenesisWallets(context.Context, common.Address, []appstore.ProjectGenesisWallet) error {
	return nil
}

func (s *projectSnapshotFallbackStore) ListProjectGenesisWalletsByContract(_ context.Context, contract common.Address) ([]appstore.ProjectGenesisWallet, error) {
	return append([]appstore.ProjectGenesisWallet(nil), s.genesisWallets[contract]...), nil
}

func (s *projectSnapshotFallbackStore) ListProjectGenesisWalletsByContracts(_ context.Context, contracts []common.Address) (map[common.Address][]appstore.ProjectGenesisWallet, error) {
	result := make(map[common.Address][]appstore.ProjectGenesisWallet)
	for _, contract := range contracts {
		result[contract] = append([]appstore.ProjectGenesisWallet(nil), s.genesisWallets[contract]...)
	}
	return result, nil
}

func (s *projectSnapshotFallbackStore) ListProjectGenesisWalletsByWallet(context.Context, common.Address) ([]appstore.ProjectGenesisWallet, error) {
	return nil, nil
}

func (s *projectSnapshotFallbackStore) ReplaceProjectCreatorHistoricalProjects(context.Context, common.Address, []appstore.ProjectCreatorHistoricalProject) error {
	return nil
}

func (s *projectSnapshotFallbackStore) ListProjectCreatorHistoricalProjectsByContract(_ context.Context, contract common.Address) ([]appstore.ProjectCreatorHistoricalProject, error) {
	return append([]appstore.ProjectCreatorHistoricalProject(nil), s.creatorHistoricalProjects[contract]...), nil
}

func (s *projectSnapshotFallbackStore) ListProjectCreatorHistoricalProjectsByContracts(_ context.Context, contracts []common.Address) (map[common.Address][]appstore.ProjectCreatorHistoricalProject, error) {
	result := make(map[common.Address][]appstore.ProjectCreatorHistoricalProject)
	for _, contract := range contracts {
		result[contract] = append([]appstore.ProjectCreatorHistoricalProject(nil), s.creatorHistoricalProjects[contract]...)
	}
	return result, nil
}
