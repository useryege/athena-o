package application

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
)

type bootstrapProjectStoreFake struct {
	metas   []appstore.ProjectMeta
	genesis map[common.Address][]appstore.ProjectGenesisWallet
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

func (s *bootstrapProjectStoreFake) UpdateProjectSourceQualityReport(context.Context, common.Address, string) error {
	return nil
}

func (s *bootstrapProjectStoreFake) ArchiveProjectByContract(context.Context, common.Address) error {
	return nil
}

func (s *bootstrapProjectStoreFake) UnarchiveProjectByContract(context.Context, common.Address) error {
	return nil
}

func (s *bootstrapProjectStoreFake) ListArchivedProjectMetas(context.Context, int32, int32) ([]appstore.ProjectMeta, int64, int32, int32, error) {
	return nil, 0, 1, 1, nil
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

func (s *bootstrapProjectStoreFake) GetArchivedProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
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

func (s *bootstrapProjectStoreFake) ListSourceCodeBlacklistFields(context.Context) ([]string, error) {
	return nil, nil
}

func (s *bootstrapProjectStoreFake) AddSourceCodeBlacklistField(context.Context, string) error {
	return nil
}

func (s *bootstrapProjectStoreFake) DeleteSourceCodeBlacklistField(context.Context, string) error {
	return nil
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

func (c *bootstrapProjectCacheFake) ListActiveProjects(context.Context) ([]*Project, error) {
	return nil, nil
}

func (c *bootstrapProjectCacheFake) ListActiveProjectsPage(context.Context, int32, int32) ([]*Project, int64, int32, int32, error) {
	return nil, 0, 1, 1, nil
}

func (c *bootstrapProjectCacheFake) ListArchivedProjects(context.Context, int32, int32) ([]*Project, int64, int32, int32, error) {
	return nil, 0, 1, 1, nil
}

func TestBootstrapProjectCachesRestoresDBProjectsOnly(t *testing.T) {
	contract := common.HexToAddress("0x00000000000000000000000000000000000000a1")
	creator := common.HexToAddress("0x00000000000000000000000000000000000000b1")
	wallet := common.HexToAddress("0x00000000000000000000000000000000000000c1")
	archivedAt := time.Unix(123, 0).UTC()
	store := &bootstrapProjectStoreFake{
		metas: []appstore.ProjectMeta{{
			BlockTime:   11,
			BlockNumber: 22,
			Contract:    contract,
			Creator:     creator,
			TxHash:      common.HexToHash("0x01"),
			TxIndex:     3,
			SourceCode:  "contract Source {}",
			IsArchived:  true,
			ArchivedAt:  archivedAt,
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
	if project.Meta.Contract != contract || project.Meta.Creator != creator || !project.Meta.IsArchived || !project.Meta.ArchivedAt.Equal(archivedAt) {
		t.Fatalf("restored meta = %+v", project.Meta)
	}
	if len(project.Meta.GenesisWallets) != 1 || project.Meta.GenesisWallets[0].Wallet != wallet || project.Meta.GenesisWallets[0].RatioBPS != 2500 {
		t.Fatalf("restored genesis wallets = %+v", project.Meta.GenesisWallets)
	}
	if project.Runtime.ChainState.TokenContract != (common.Address{}) {
		t.Fatalf("bootstrap populated chain state, want zero runtime state")
	}
	if project.Runtime.CreatorResult.HasMintRisk() {
		t.Fatal("bootstrap populated simulation result, want zero runtime simulation")
	}
}

func TestBootstrapProjectCachesRequiresProjectStore(t *testing.T) {
	service := &Service{projectCache: &bootstrapProjectCacheFake{}}

	if err := service.bootstrapProjectCaches(context.Background()); err == nil {
		t.Fatal("bootstrapProjectCaches error = nil, want failed precondition")
	}
}
