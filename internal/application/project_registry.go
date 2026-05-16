package application

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type ProjectRegistry interface {
	GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error)
	ListProjects(ctx context.Context) ([]*Project, error)
	ListProjectContracts(ctx context.Context) ([]ProjectContractRef, error)
	SetProject(ctx context.Context, projectID uuid.UUID, project *Project) error
	LoadProject(ctx context.Context, projectID uuid.UUID, project *Project) error
	UpdateProjectChainStates(ctx context.Context, states map[uuid.UUID]athenacontract.AthenaProject) error
	UpdateProjectMetaState(ctx context.Context, projectID uuid.UUID, state *ProjectMeta) error
	RemoveProject(ctx context.Context, projectID uuid.UUID) error
}

var _ ProjectRegistry = &projectRegistryImpl{}

type ProjectContractRef struct {
	ProjectID uuid.UUID
	Contract  common.Address
	Creator   common.Address
}

type projectRegistryImpl struct {
	mu                        sync.RWMutex
	ProjectsByContract        map[common.Address]struct{}
	Projects                  map[uuid.UUID]*Project
	ProjectContractRefs       []ProjectContractRef
	ProjectContractRefIndexes map[uuid.UUID]int
	publisher                 PersistenceEventPublisher
}

func NewProjectRegistry(publisher PersistenceEventPublisher) ProjectRegistry {
	if publisher == nil {
		panic("persistence publisher is not configured")
	}
	return &projectRegistryImpl{
		Projects:                  make(map[uuid.UUID]*Project),
		ProjectsByContract:        make(map[common.Address]struct{}),
		ProjectContractRefIndexes: make(map[uuid.UUID]int),
		publisher:                 publisher,
	}
}

func (r *projectRegistryImpl) GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, ok := r.Projects[projectID]
	if !ok {
		return nil, false, nil
	}

	return cloneProject(project), true, nil
}

func (r *projectRegistryImpl) ListProjects(ctx context.Context) ([]*Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projects := make([]*Project, 0, len(r.Projects))
	for _, project := range r.Projects {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		projects = append(projects, cloneProject(project))
	}

	return projects, nil
}

func (r *projectRegistryImpl) ListProjectContracts(ctx context.Context) ([]ProjectContractRef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	refs := make([]ProjectContractRef, len(r.ProjectContractRefs))
	copy(refs, r.ProjectContractRefs)
	return refs, nil
}

func (r *projectRegistryImpl) SetProject(ctx context.Context, projectID uuid.UUID, project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, ok := r.ProjectsByContract[project.Meta.Contract]; ok {
		return nil
	}
	if err := r.publisher.PublishProjectMetaSave(ctx, projectMetaToStore(project.Meta)); err != nil {
		return err
	}
	r.setProjectLocked(projectID, project)
	return nil
}

func (r *projectRegistryImpl) LoadProject(ctx context.Context, projectID uuid.UUID, project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if project == nil {
		return nil
	}
	if _, ok := r.ProjectsByContract[project.Meta.Contract]; ok {
		return nil
	}
	r.setProjectLocked(projectID, project)
	return nil
}

func (r *projectRegistryImpl) setProjectLocked(projectID uuid.UUID, project *Project) {
	if project == nil {
		return
	}
	if _, ok := r.ProjectsByContract[project.Meta.Contract]; ok {
		return
	}
	r.ProjectsByContract[project.Meta.Contract] = struct{}{}
	r.Projects[projectID] = cloneProject(project)
	r.ProjectContractRefIndexes[projectID] = len(r.ProjectContractRefs)
	r.ProjectContractRefs = append(r.ProjectContractRefs, ProjectContractRef{
		ProjectID: project.Meta.ProjectID,
		Contract:  project.Meta.Contract,
		Creator:   project.Meta.Creator,
	})
}

func (r *projectRegistryImpl) UpdateProjectChainStates(ctx context.Context, states map[uuid.UUID]athenacontract.AthenaProject) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for projectID, state := range states {
		project, ok := r.Projects[projectID]
		if !ok {
			continue
		}
		project.ChainState = cloneAthenaProject(state)
	}
	return nil
}

func (r *projectRegistryImpl) UpdateProjectMetaState(ctx context.Context, projectID uuid.UUID, state *ProjectMeta) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	project, ok := r.Projects[projectID]
	if !ok {
		return nil
	}
	if state != nil {
		if state.SourceCode != "" {
			if err := r.publisher.PublishProjectSourceCodeUpdate(ctx, projectID, state.SourceCode); err != nil {
				return err
			}
			project.Meta.SourceCode = state.SourceCode
		}
		project.Meta.CreatorResult = state.CreatorResult
		project.Meta.SourceCodeBlacklist = sourcecode.BlacklistReport{
			HasBlacklistFields: state.SourceCodeBlacklist.HasBlacklistFields,
			BlacklistFields:    cloneStringSlice(state.SourceCodeBlacklist.BlacklistFields),
			ResolvedAt:         state.SourceCodeBlacklist.ResolvedAt,
		}
	}
	return nil
}

func (r *projectRegistryImpl) RemoveProject(ctx context.Context, projectID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if project, ok := r.Projects[projectID]; ok {
		delete(r.ProjectsByContract, project.Meta.Contract)
	}
	delete(r.Projects, projectID)
	r.removeProjectContractRef(projectID)
	return nil
}

func (r *projectRegistryImpl) removeProjectContractRef(projectID uuid.UUID) {
	index, ok := r.ProjectContractRefIndexes[projectID]
	if !ok {
		return
	}

	lastIndex := len(r.ProjectContractRefs) - 1
	if index != lastIndex {
		lastRef := r.ProjectContractRefs[lastIndex]
		r.ProjectContractRefs[index] = lastRef
		r.ProjectContractRefIndexes[lastRef.ProjectID] = index
	}
	r.ProjectContractRefs[lastIndex] = ProjectContractRef{}
	r.ProjectContractRefs = r.ProjectContractRefs[:lastIndex]
	delete(r.ProjectContractRefIndexes, projectID)
}

func cloneProject(project *Project) *Project {
	if project == nil {
		return nil
	}
	return &Project{
		Meta:       cloneProjectMeta(project.Meta),
		ChainState: cloneAthenaProject(project.ChainState),
	}
}

func cloneProjectMeta(meta ProjectMeta) ProjectMeta {
	meta.SourceCodeBlacklist = sourcecode.BlacklistReport{
		HasBlacklistFields: meta.SourceCodeBlacklist.HasBlacklistFields,
		BlacklistFields:    cloneStringSlice(meta.SourceCodeBlacklist.BlacklistFields),
		ResolvedAt:         meta.SourceCodeBlacklist.ResolvedAt,
	}
	return meta
}

func projectMetaToStore(meta ProjectMeta) appstore.ProjectMeta {
	txHash := meta.TxHash
	if meta.Tx != nil {
		txHash = meta.Tx.Hash()
	}
	return appstore.ProjectMeta{
		ProjectID:   meta.ProjectID,
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract,
		Creator:     meta.Creator,
		Tx:          meta.Tx,
		TxHash:      txHash,
		TxIndex:     meta.TxIndex,
		SourceCode:  meta.SourceCode,
		IsArchived:  meta.IsArchived,
		ArchivedAt:  meta.ArchivedAt,
	}
}

func projectMetaFromStore(meta appstore.ProjectMeta) ProjectMeta {
	return ProjectMeta{
		ProjectID:   meta.ProjectID,
		BlockTime:   meta.BlockTime,
		BlockNumber: meta.BlockNumber,
		Contract:    meta.Contract,
		Creator:     meta.Creator,
		Tx:          meta.Tx,
		TxHash:      meta.TxHash,
		TxIndex:     meta.TxIndex,
		SourceCode:  meta.SourceCode,
		IsArchived:  meta.IsArchived,
		ArchivedAt:  meta.ArchivedAt,
	}
}

func cloneAthenaProject(project athenacontract.AthenaProject) athenacontract.AthenaProject {
	project.UpdatedAt = cloneBigInt(project.UpdatedAt)
	project.Token.TotalSupply = cloneBigInt(project.Token.TotalSupply)
	project.WethPair.TotalSupply = cloneBigInt(project.WethPair.TotalSupply)
	project.WethPair.LockedLiquidity = cloneBigInt(project.WethPair.LockedLiquidity)
	project.WethPair.BaseBalance = cloneBigInt(project.WethPair.BaseBalance)
	project.WethPair.QuoteBalance = cloneBigInt(project.WethPair.QuoteBalance)
	project.WethPair.QuoteUsdtValue = cloneBigInt(project.WethPair.QuoteUsdtValue)
	project.WethPair.Reserve0 = cloneBigInt(project.WethPair.Reserve0)
	project.WethPair.Reserve1 = cloneBigInt(project.WethPair.Reserve1)
	project.UsdtPair.TotalSupply = cloneBigInt(project.UsdtPair.TotalSupply)
	project.UsdtPair.LockedLiquidity = cloneBigInt(project.UsdtPair.LockedLiquidity)
	project.UsdtPair.BaseBalance = cloneBigInt(project.UsdtPair.BaseBalance)
	project.UsdtPair.QuoteBalance = cloneBigInt(project.UsdtPair.QuoteBalance)
	project.UsdtPair.QuoteUsdtValue = cloneBigInt(project.UsdtPair.QuoteUsdtValue)
	project.UsdtPair.Reserve0 = cloneBigInt(project.UsdtPair.Reserve0)
	project.UsdtPair.Reserve1 = cloneBigInt(project.UsdtPair.Reserve1)
	return project
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneBigInt(value *big.Int) *big.Int {
	if value == nil {
		return nil
	}
	return new(big.Int).Set(value)
}
