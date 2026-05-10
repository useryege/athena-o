package application

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
)

type ProjectStore interface {
	SaveProjectMeta(ctx context.Context, meta ProjectMeta) error
}
type ProjectRegistry interface {
	GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error)
	ListProjects(ctx context.Context) ([]*v1alpha1.ProjectView, error)
	SetProject(ctx context.Context, projectID uuid.UUID, project *Project) error
	RemoveProject(ctx context.Context, projectID uuid.UUID) error
}

var _ ProjectRegistry = &projectRegistryImpl{}

type projectRegistryImpl struct {
	mu                 sync.RWMutex
	ProjectsByContract map[common.Address]struct{}
	Projects           map[uuid.UUID]*Project
	store              ProjectStore
}

func NewProjectRegistry(store ProjectStore) ProjectRegistry {
	return &projectRegistryImpl{
		Projects:           make(map[uuid.UUID]*Project),
		ProjectsByContract: make(map[common.Address]struct{}),
		store:              store,
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

func (r *projectRegistryImpl) ListProjects(ctx context.Context) ([]*v1alpha1.ProjectView, error) {
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

	sort.SliceStable(projects, func(i, j int) bool {
		if projects[i].Meta.BlockNumber != projects[j].Meta.BlockNumber {
			return projects[i].Meta.BlockNumber < projects[j].Meta.BlockNumber
		}
		if projects[i].Meta.TxIndex != projects[j].Meta.TxIndex {
			return projects[i].Meta.TxIndex > projects[j].Meta.TxIndex
		}
		return projects[i].Meta.ProjectID.String() > projects[j].Meta.ProjectID.String()
	})

	projectViews := make([]*v1alpha1.ProjectView, 0, len(projects))
	for _, project := range projects {
		projectViews = append(projectViews, projectToView(project))
	}

	return projectViews, nil
}

func (r *projectRegistryImpl) SetProject(ctx context.Context, projectID uuid.UUID, project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.ProjectsByContract[project.Meta.Contract]; ok {
		return errors.New("project already exists")
	}
	r.ProjectsByContract[project.Meta.Contract] = struct{}{}
	r.Projects[projectID] = project
	return nil
}

func (r *projectRegistryImpl) RemoveProject(ctx context.Context, projectID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.ProjectsByContract, r.Projects[projectID].Meta.Contract)
	delete(r.Projects, projectID)
	return nil
}
