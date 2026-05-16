package application

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/useryege/athena/internal/application/sourcecode"
	appstore "github.com/useryege/athena/internal/application/store"
	athenacontract "github.com/useryege/athena/pkg/abi/ATHENA"
)

type ProjectRegistry interface {
	// Returned projects/snapshots are shared mutable references.
	// Callers must avoid uncontrolled concurrent writes.
	GetProject(ctx context.Context, contract common.Address) (*Project, bool, error)
	ListProjects(ctx context.Context) ([]*Project, error)
	ListProjectContracts(ctx context.Context) (ProjectContractSnapshot, error)
	SetProject(ctx context.Context, project *Project) error
	LoadProject(ctx context.Context, project *Project) error
	UpdateProjectChainStates(ctx context.Context, states map[common.Address]athenacontract.AthenaProject) error
	UpdateProjectMetaState(ctx context.Context, contract common.Address, state *ProjectMeta) error
	RemoveProject(ctx context.Context, contract common.Address) error
}

var _ ProjectRegistry = &projectRegistryImpl{}

type ProjectContractSnapshot struct {
	ProjectContracts []common.Address
	ProjectQueries   []athenacontract.AthenaProjectQuery
}

type projectRegistryImpl struct {
	mu               sync.RWMutex
	Projects         map[common.Address]*Project
	ProjectContracts []common.Address
	ProjectQueries   []athenacontract.AthenaProjectQuery
	ProjectIndexes   map[common.Address]int
	publisher        PersistenceEventPublisher
}

func NewProjectRegistry(publisher PersistenceEventPublisher) ProjectRegistry {
	if publisher == nil {
		panic("persistence publisher is not configured")
	}
	return &projectRegistryImpl{
		Projects:       make(map[common.Address]*Project),
		ProjectIndexes: make(map[common.Address]int),
		publisher:      publisher,
	}
}

func (r *projectRegistryImpl) GetProject(ctx context.Context, contract common.Address) (*Project, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, ok := r.Projects[contract]
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
		ProjectContracts: r.ProjectContracts,
		ProjectQueries:   r.ProjectQueries,
	}, nil
}

func (r *projectRegistryImpl) SetProject(ctx context.Context, project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.setProjectWithOptionsLocked(ctx, project, true)
}

func (r *projectRegistryImpl) LoadProject(ctx context.Context, project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.setProjectWithOptionsLocked(ctx, project, false)
}

func (r *projectRegistryImpl) setProjectWithOptionsLocked(ctx context.Context, project *Project, persist bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if project == nil {
		return nil
	}
	contract := project.Meta.Contract
	if _, ok := r.Projects[contract]; ok {
		return nil
	}
	if persist {
		if err := r.publisher.PublishProjectMetaSave(ctx, projectMetaToStore(project.Meta)); err != nil {
			return err
		}
	}
	r.setProjectLocked(project)
	return nil
}

func (r *projectRegistryImpl) setProjectLocked(project *Project) {
	if project == nil {
		return
	}
	contract := project.Meta.Contract
	if _, ok := r.Projects[contract]; ok {
		return
	}
	r.Projects[contract] = project
	r.ProjectIndexes[contract] = len(r.ProjectContracts)
	r.ProjectContracts = append(r.ProjectContracts, contract)
	r.ProjectQueries = append(r.ProjectQueries, athenacontract.AthenaProjectQuery{
		TokenContract: contract,
		MsgCaller:     project.Meta.Creator,
	})
}

func (r *projectRegistryImpl) UpdateProjectChainStates(ctx context.Context, states map[common.Address]athenacontract.AthenaProject) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for contract, state := range states {
		project, ok := r.Projects[contract]
		if !ok {
			continue
		}
		project.ChainState = state
	}
	return nil
}

func (r *projectRegistryImpl) UpdateProjectMetaState(ctx context.Context, contract common.Address, state *ProjectMeta) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	project, ok := r.Projects[contract]
	if !ok {
		return nil
	}
	if state != nil {
		if state.SourceCode != "" {
			if err := r.publisher.PublishProjectSourceCodeUpdate(ctx, contract, state.SourceCode); err != nil {
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

func (r *projectRegistryImpl) RemoveProject(ctx context.Context, contract common.Address) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(r.Projects, contract)
	r.removeProjectContractSnapshot(contract)
	return nil
}

func (r *projectRegistryImpl) removeProjectContractSnapshot(contract common.Address) {
	index, ok := r.ProjectIndexes[contract]
	if !ok {
		return
	}

	lastIndex := len(r.ProjectContracts) - 1
	if index != lastIndex {
		lastContract := r.ProjectContracts[lastIndex]
		r.ProjectContracts[index] = lastContract
		r.ProjectQueries[index] = r.ProjectQueries[lastIndex]
		r.ProjectIndexes[lastContract] = index
	}
	r.ProjectContracts[lastIndex] = common.Address{}
	r.ProjectQueries[lastIndex] = athenacontract.AthenaProjectQuery{}
	r.ProjectContracts = r.ProjectContracts[:lastIndex]
	r.ProjectQueries = r.ProjectQueries[:lastIndex]
	delete(r.ProjectIndexes, contract)
}

func projectMetaToStore(meta ProjectMeta) appstore.ProjectMeta {
	txHash := meta.TxHash
	if meta.Tx != nil {
		txHash = meta.Tx.Hash()
	}
	return appstore.ProjectMeta{
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
