package application

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type ProjectRegistry interface {
	// Returned projects/snapshots are shared mutable references.
	// Callers must avoid uncontrolled concurrent writes.
	GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error)
	ListProjects(ctx context.Context) ([]*Project, error)
	ListProjectContracts(ctx context.Context) (ProjectContractSnapshot, error)
	SetProject(ctx context.Context, projectID uuid.UUID, project *Project) error
	LoadProject(ctx context.Context, projectID uuid.UUID, project *Project) error
	UpdateProjectChainStates(ctx context.Context, states map[uuid.UUID]athenacontract.AthenaProject) error
	UpdateProjectMetaState(ctx context.Context, projectID uuid.UUID, state *ProjectMeta) error
	RemoveProject(ctx context.Context, projectID uuid.UUID) error
}

var _ ProjectRegistry = &projectRegistryImpl{}

type ProjectContractSnapshot struct {
	ProjectIDs       []uuid.UUID
	ProjectContracts []common.Address
	ProjectQueries   []athenacontract.AthenaProjectQuery
}

type projectRegistryImpl struct {
	mu                 sync.RWMutex
	ProjectsByContract map[common.Address]struct{}
	Projects           map[uuid.UUID]*Project
	ProjectIDs         []uuid.UUID
	ProjectContracts   []common.Address
	ProjectQueries     []athenacontract.AthenaProjectQuery
	ProjectIndexes     map[uuid.UUID]int
	publisher          PersistenceEventPublisher
}

func NewProjectRegistry(publisher PersistenceEventPublisher) ProjectRegistry {
	if publisher == nil {
		panic("persistence publisher is not configured")
	}
	return &projectRegistryImpl{
		Projects:           make(map[uuid.UUID]*Project),
		ProjectsByContract: make(map[common.Address]struct{}),
		ProjectIndexes:     make(map[uuid.UUID]int),
		publisher:          publisher,
	}
}

func (r *projectRegistryImpl) GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, ok := r.Projects[projectID]
	if !ok {
		return nil, false, nil
	}

	return project, true, nil
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
		projects = append(projects, project)
	}

	return projects, nil
}

func (r *projectRegistryImpl) ListProjectContracts(ctx context.Context) (ProjectContractSnapshot, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := ctx.Err(); err != nil {
		return ProjectContractSnapshot{}, err
	}

	return ProjectContractSnapshot{
		ProjectIDs:       r.ProjectIDs,
		ProjectContracts: r.ProjectContracts,
		ProjectQueries:   r.ProjectQueries,
	}, nil
}

func (r *projectRegistryImpl) SetProject(ctx context.Context, projectID uuid.UUID, project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.setProjectWithOptionsLocked(ctx, projectID, project, true)
}

func (r *projectRegistryImpl) LoadProject(ctx context.Context, projectID uuid.UUID, project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.setProjectWithOptionsLocked(ctx, projectID, project, false)
}

func (r *projectRegistryImpl) setProjectWithOptionsLocked(ctx context.Context, projectID uuid.UUID, project *Project, persist bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if project == nil {
		return nil
	}
	if _, ok := r.ProjectsByContract[project.Meta.Contract]; ok {
		return nil
	}
	if persist {
		if err := r.publisher.PublishProjectMetaSave(ctx, projectMetaToStore(project.Meta)); err != nil {
			return err
		}
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
	r.Projects[projectID] = project
	r.ProjectIndexes[projectID] = len(r.ProjectIDs)
	r.ProjectIDs = append(r.ProjectIDs, project.Meta.ProjectID)
	r.ProjectContracts = append(r.ProjectContracts, project.Meta.Contract)
	r.ProjectQueries = append(r.ProjectQueries, athenacontract.AthenaProjectQuery{
		TokenContract: project.Meta.Contract,
		MsgCaller:     project.Meta.Creator,
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
		project.ChainState = state
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
			BlacklistFields:    state.SourceCodeBlacklist.BlacklistFields,
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
	r.removeProjectContractSnapshot(projectID)
	return nil
}

func (r *projectRegistryImpl) removeProjectContractSnapshot(projectID uuid.UUID) {
	index, ok := r.ProjectIndexes[projectID]
	if !ok {
		return
	}

	lastIndex := len(r.ProjectIDs) - 1
	if index != lastIndex {
		lastProjectID := r.ProjectIDs[lastIndex]
		r.ProjectIDs[index] = lastProjectID
		r.ProjectContracts[index] = r.ProjectContracts[lastIndex]
		r.ProjectQueries[index] = r.ProjectQueries[lastIndex]
		r.ProjectIndexes[lastProjectID] = index
	}
	r.ProjectIDs[lastIndex] = uuid.UUID{}
	r.ProjectContracts[lastIndex] = common.Address{}
	r.ProjectQueries[lastIndex] = athenacontract.AthenaProjectQuery{}
	r.ProjectIDs = r.ProjectIDs[:lastIndex]
	r.ProjectContracts = r.ProjectContracts[:lastIndex]
	r.ProjectQueries = r.ProjectQueries[:lastIndex]
	delete(r.ProjectIndexes, projectID)
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
