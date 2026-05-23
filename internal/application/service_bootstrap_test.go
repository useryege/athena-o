package application

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type bootstrapProjectStoreFake struct {
	metas          []appstore.ProjectMeta
	genesis        map[common.Address][]appstore.ProjectGenesisWallet
	creatorHistory map[common.Address][]appstore.ProjectCreatorHistoricalProject
}

func (s *bootstrapProjectStoreFake) SaveProjectMeta(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (s *bootstrapProjectStoreFake) ListProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return append([]appstore.ProjectMeta(nil), s.metas...), nil
}

func (s *bootstrapProjectStoreFake) ListAllProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return append([]appstore.ProjectMeta(nil), s.metas...), nil
}

func (s *bootstrapProjectStoreFake) UpdateProjectSourceCode(context.Context, common.Address, string) error {
	return nil
}

func (s *bootstrapProjectStoreFake) UpdateProjectCodeBinHash(context.Context, common.Address, common.Hash) error {
	return nil
}

func (s *bootstrapProjectStoreFake) UpdateProjectSourceQualityReport(context.Context, common.Address, string) error {
	return nil
}

func (s *bootstrapProjectStoreFake) UpdateProjectCreatorResult(context.Context, common.Address, appstore.SimulateResult) error {
	return nil
}

func (s *bootstrapProjectStoreFake) ListProjectMetasByCreator(_ context.Context, creator common.Address) ([]appstore.ProjectMeta, error) {
	metas := make([]appstore.ProjectMeta, 0)
	for _, meta := range s.metas {
		if meta.Creator == creator {
			metas = append(metas, meta)
		}
	}
	return metas, nil
}

func (s *bootstrapProjectStoreFake) ListProjectMetasByCreatorBefore(_ context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]appstore.ProjectMeta, error) {
	metas := make([]appstore.ProjectMeta, 0)
	for _, meta := range s.metas {
		if meta.Creator != creator {
			continue
		}
		if meta.BlockNumber < blockNumber || (meta.BlockNumber == blockNumber && meta.TxIndex < txIndex) {
			metas = append(metas, meta)
		}
	}
	return metas, nil
}

func (s *bootstrapProjectStoreFake) GetProjectMetaByContract(_ context.Context, contract common.Address) (*appstore.ProjectMeta, error) {
	for _, meta := range s.metas {
		if meta.Contract == contract {
			item := meta
			return &item, nil
		}
	}
	return nil, nil
}

func (s *bootstrapProjectStoreFake) ReplaceProjectGenesisWallets(context.Context, common.Address, []appstore.ProjectGenesisWallet) error {
	return nil
}

func (s *bootstrapProjectStoreFake) ListProjectGenesisWalletsByContract(_ context.Context, contract common.Address) ([]appstore.ProjectGenesisWallet, error) {
	return append([]appstore.ProjectGenesisWallet(nil), s.genesis[contract]...), nil
}

func (s *bootstrapProjectStoreFake) ListProjectGenesisWalletsByContracts(_ context.Context, contracts []common.Address) (map[common.Address][]appstore.ProjectGenesisWallet, error) {
	result := make(map[common.Address][]appstore.ProjectGenesisWallet, len(contracts))
	for _, contract := range contracts {
		result[contract] = append([]appstore.ProjectGenesisWallet(nil), s.genesis[contract]...)
	}
	return result, nil
}

func (s *bootstrapProjectStoreFake) ListProjectGenesisWalletsByWallet(_ context.Context, wallet common.Address) ([]appstore.ProjectGenesisWallet, error) {
	items := make([]appstore.ProjectGenesisWallet, 0)
	for _, records := range s.genesis {
		for _, item := range records {
			if item.Wallet == wallet {
				items = append(items, item)
			}
		}
	}
	return items, nil
}

func (s *bootstrapProjectStoreFake) ReplaceProjectCreatorHistoricalProjects(context.Context, common.Address, []appstore.ProjectCreatorHistoricalProject) error {
	return nil
}

func (s *bootstrapProjectStoreFake) ListProjectCreatorHistoricalProjectsByContract(_ context.Context, contract common.Address) ([]appstore.ProjectCreatorHistoricalProject, error) {
	return append([]appstore.ProjectCreatorHistoricalProject(nil), s.creatorHistory[contract]...), nil
}

func (s *bootstrapProjectStoreFake) ListProjectCreatorHistoricalProjectsByContracts(_ context.Context, contracts []common.Address) (map[common.Address][]appstore.ProjectCreatorHistoricalProject, error) {
	result := make(map[common.Address][]appstore.ProjectCreatorHistoricalProject, len(contracts))
	for _, contract := range contracts {
		result[contract] = append([]appstore.ProjectCreatorHistoricalProject(nil), s.creatorHistory[contract]...)
	}
	return result, nil
}

type bootstrapProjectStoreWithoutGenesisFake struct {
	metas []appstore.ProjectMeta
}

func (s *bootstrapProjectStoreWithoutGenesisFake) SaveProjectMeta(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (s *bootstrapProjectStoreWithoutGenesisFake) ListProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return append([]appstore.ProjectMeta(nil), s.metas...), nil
}

func (s *bootstrapProjectStoreWithoutGenesisFake) ListAllProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return append([]appstore.ProjectMeta(nil), s.metas...), nil
}

func (s *bootstrapProjectStoreWithoutGenesisFake) UpdateProjectSourceCode(context.Context, common.Address, string) error {
	return nil
}

func (s *bootstrapProjectStoreWithoutGenesisFake) UpdateProjectCodeBinHash(context.Context, common.Address, common.Hash) error {
	return nil
}

func (s *bootstrapProjectStoreWithoutGenesisFake) UpdateProjectSourceQualityReport(context.Context, common.Address, string) error {
	return nil
}

func (s *bootstrapProjectStoreWithoutGenesisFake) UpdateProjectCreatorResult(context.Context, common.Address, appstore.SimulateResult) error {
	return nil
}

func (s *bootstrapProjectStoreWithoutGenesisFake) ListProjectMetasByCreator(_ context.Context, creator common.Address) ([]appstore.ProjectMeta, error) {
	metas := make([]appstore.ProjectMeta, 0)
	for _, meta := range s.metas {
		if meta.Creator == creator {
			metas = append(metas, meta)
		}
	}
	return metas, nil
}

func (s *bootstrapProjectStoreWithoutGenesisFake) ListProjectMetasByCreatorBefore(_ context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]appstore.ProjectMeta, error) {
	metas := make([]appstore.ProjectMeta, 0)
	for _, meta := range s.metas {
		if meta.Creator != creator {
			continue
		}
		if meta.BlockNumber < blockNumber || (meta.BlockNumber == blockNumber && meta.TxIndex < txIndex) {
			metas = append(metas, meta)
		}
	}
	return metas, nil
}

func (s *bootstrapProjectStoreWithoutGenesisFake) GetProjectMetaByContract(_ context.Context, contract common.Address) (*appstore.ProjectMeta, error) {
	for _, meta := range s.metas {
		if meta.Contract == contract {
			item := meta
			return &item, nil
		}
	}
	return nil, nil
}

type bootstrapProjectCacheFake struct {
	replaced []*Project
}

func (c *bootstrapProjectCacheFake) ReplaceAll(_ context.Context, projects []*Project) error {
	c.replaced = append([]*Project(nil), projects...)
	return nil
}

func (c *bootstrapProjectCacheFake) SetProject(context.Context, *Project) error { return nil }

func (c *bootstrapProjectCacheFake) UpdateProject(context.Context, common.Address, ProjectUpdater) (bool, error) {
	return false, nil
}

func (c *bootstrapProjectCacheFake) DeleteProject(context.Context, common.Address) error {
	return nil
}

func (c *bootstrapProjectCacheFake) GetProject(context.Context, common.Address) (*Project, bool, error) {
	return nil, false, nil
}

func (c *bootstrapProjectCacheFake) GetMaxProjectBlockNumber(context.Context) (uint64, bool, error) {
	return 0, false, nil
}

func (c *bootstrapProjectCacheFake) ListProjects(context.Context) ([]*Project, error) {
	return nil, nil
}

func (c *bootstrapProjectCacheFake) ListProjectsPage(context.Context, int32, int32) ([]*Project, int64, int32, int32, error) {
	return nil, 0, 1, 1, nil
}

func TestBootstrapProjectCachesRestoresDBProjectsOnly(t *testing.T) {
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000b1")
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	creatorResultFetchedAt := time.Date(2026, 5, 23, 6, 0, 0, 0, time.UTC)
	store := &bootstrapProjectStoreFake{
		metas: []appstore.ProjectMeta{{
			BlockTime:              11,
			BlockNumber:            22,
			Contract:               contract,
			Creator:                creator,
			TxHash:                 common.HexToHash("0x01"),
			TxIndex:                3,
			SourceCode:             "contract Source {}",
			CreatorResult:          appstore.SimulateResult{CanMintViaTransferToWethPair: true},
			CreatorResultFetchedAt: creatorResultFetchedAt,
		}},
		genesis: map[common.Address][]appstore.ProjectGenesisWallet{
			contract: {{
				ProjectContract: contract,
				Wallet:          wallet,
				NetAmount:       big.NewInt(100),
				RatioBPS:        2500,
				RankIndex:       1,
			}},
		},
	}
	cache := &bootstrapProjectCacheFake{}
	service := &Service{store: store, projectCache: cache}

	if err := service.bootstrapProjectCaches(context.Background()); err != nil {
		t.Fatalf("bootstrapProjectCaches: %v", err)
	}
	if len(cache.replaced) != 1 {
		t.Fatalf("replaced project count = %d, want 1", len(cache.replaced))
	}
	project := cache.replaced[0]
	if project.Meta.Contract != contract || project.Meta.Creator != creator {
		t.Fatalf("restored meta = %+v", project.Meta)
	}
	if len(project.Meta.GenesisWallets) != 1 || project.Meta.GenesisWallets[0].Wallet != wallet || project.Meta.GenesisWallets[0].RatioBPS != 2500 {
		t.Fatalf("restored genesis wallets = %+v", project.Meta.GenesisWallets)
	}
	if project.Meta.ChainState.TokenContract != (common.Address{}) {
		t.Fatalf("bootstrap populated chain state, want zero runtime state")
	}
	if !project.Meta.CreatorResult.CanMintViaTransferToWethPair {
		t.Fatalf("restored creator result = %+v, want weth transfer mint flag", project.Meta.CreatorResult)
	}
	if !project.Meta.CreatorResultFetchedAt.Equal(creatorResultFetchedAt) {
		t.Fatalf("restored creator result fetched at = %s, want %s", project.Meta.CreatorResultFetchedAt, creatorResultFetchedAt)
	}
}

func TestBootstrapProjectCachesRequiresProjectStore(t *testing.T) {
	service := &Service{projectCache: &bootstrapProjectCacheFake{}}

	err := service.bootstrapProjectCaches(context.Background())
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("bootstrapProjectCaches error code = %v, want %v: %v", status.Code(err), codes.FailedPrecondition, err)
	}
}

func TestBootstrapProjectCachesRequiresGenesisWalletStore(t *testing.T) {
	service := &Service{
		store:        &bootstrapProjectStoreWithoutGenesisFake{},
		projectCache: &bootstrapProjectCacheFake{},
	}

	err := service.bootstrapProjectCaches(context.Background())
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("bootstrapProjectCaches error code = %v, want %v: %v", status.Code(err), codes.FailedPrecondition, err)
	}
}

func TestBootstrapProjectCachesRequiresProjectCache(t *testing.T) {
	service := &Service{store: &bootstrapProjectStoreFake{}}

	err := service.bootstrapProjectCaches(context.Background())
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("bootstrapProjectCaches error code = %v, want %v: %v", status.Code(err), codes.FailedPrecondition, err)
	}
}

func TestBootstrapProjectCachesRequiresProjectCacheRedisClient(t *testing.T) {
	service := &Service{
		store:        &bootstrapProjectStoreFake{},
		projectCache: NewProjectSnapshotCache(nil),
	}

	err := service.bootstrapProjectCaches(context.Background())
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("bootstrapProjectCaches error code = %v, want %v: %v", status.Code(err), codes.FailedPrecondition, err)
	}
}
