package application

import (
	"sync"

	"github.com/google/uuid"
)

type ProjectRegistry interface {
	Get(projectID uuid.UUID) (*Project, bool, error)
	Set(projectID uuid.UUID, project *Project) error
	Remove(projectID uuid.UUID) error
	List() ([]*Project, error)
}

type projectRegistryImpl struct {
	mu       sync.RWMutex
	Projects map[uuid.UUID]*Project
}

func NewProjectRegistry() ProjectRegistry {
	return &projectRegistryImpl{
		Projects: make(map[uuid.UUID]*Project),
	}
}

func (r *projectRegistryImpl) Get(projectID uuid.UUID) (*Project, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	project, ok := r.Projects[projectID]
	return project, ok, nil
}

func (r *projectRegistryImpl) Set(projectID uuid.UUID, project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Projects[projectID] = project
	return nil
}

func (r *projectRegistryImpl) Remove(projectID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Projects, projectID)
	return nil
}

func (r *projectRegistryImpl) List() ([]*Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	projects := make([]*Project, 0, len(r.Projects))
	for _, project := range r.Projects {
		projects = append(projects, project)
	}
	return projects, nil
}
