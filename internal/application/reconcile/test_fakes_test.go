package reconcile

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	appcache "github.com/useryege/athena/internal/application/cache"
	"github.com/useryege/athena/internal/application/redisport"
	appstore "github.com/useryege/athena/internal/application/store"
)

func NewProjectSnapshotCache(client redisport.Client) ProjectSnapshotCache {
	return appcache.NewProjectSnapshotCache(client)
}

type discoveryProjectStoreFake struct {
	metas        []appstore.ProjectMeta
	pairMetas    []appstore.ProjectMeta
	codeBinMetas []appstore.ProjectMeta
	errs         []error
	calls        int
	pairCalls    int
	codeBinCalls int
	maxBlock     uint64
	maxBlockOK   bool
	maxBlockErr  error

	gotCreator     common.Address
	gotBlockNumber uint64
	gotTxIndex     uint64
	gotPairs       []common.Address
	gotCodeBinHash common.Hash
}

func (s *discoveryProjectStoreFake) SaveProjectMeta(context.Context, appstore.ProjectMeta) error {
	return nil
}

func (s *discoveryProjectStoreFake) GetMaxProjectBlockNumber(context.Context) (uint64, bool, error) {
	return s.maxBlock, s.maxBlockOK, s.maxBlockErr
}

func (s *discoveryProjectStoreFake) ListProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (s *discoveryProjectStoreFake) ListAllProjectMetas(context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (s *discoveryProjectStoreFake) ListProjectMetasByPairAddresses(_ context.Context, pairs []common.Address) ([]appstore.ProjectMeta, error) {
	s.pairCalls++
	s.gotPairs = append([]common.Address(nil), pairs...)
	return append([]appstore.ProjectMeta(nil), s.pairMetas...), nil
}

func (s *discoveryProjectStoreFake) ListProjectMetasByCodeBinHash(_ context.Context, codeBinHash common.Hash) ([]appstore.ProjectMeta, error) {
	s.codeBinCalls++
	s.gotCodeBinHash = codeBinHash
	return append([]appstore.ProjectMeta(nil), s.codeBinMetas...), nil
}

func (s *discoveryProjectStoreFake) UpdateProjectSourceCode(context.Context, common.Address, string, string) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpdateProjectCodeBinHash(context.Context, common.Address, common.Hash) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpdateProjectSourceQualityReport(context.Context, common.Address, string, string) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpsertProjectAveDetail(context.Context, common.Address, appstore.ProjectAveDetail) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpdateProjectCreatorResult(context.Context, common.Address, appstore.SimulateResult) error {
	return nil
}

func (s *discoveryProjectStoreFake) UpdateProjectReport(context.Context, common.Address, appstore.ProjectReport) error {
	return nil
}

func (s *discoveryProjectStoreFake) ListProjectMetasByCreator(context.Context, common.Address) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (s *discoveryProjectStoreFake) ListProjectMetasByCreatorBefore(_ context.Context, creator common.Address, blockNumber uint64, txIndex uint64) ([]appstore.ProjectMeta, error) {
	s.calls++
	s.gotCreator = creator
	s.gotBlockNumber = blockNumber
	s.gotTxIndex = txIndex
	if len(s.errs) > 0 {
		err := s.errs[0]
		s.errs = s.errs[1:]
		if err != nil {
			return nil, err
		}
	}
	return append([]appstore.ProjectMeta(nil), s.metas...), nil
}

func (s *discoveryProjectStoreFake) GetProjectMetaByContract(context.Context, common.Address) (*appstore.ProjectMeta, error) {
	return nil, nil
}
