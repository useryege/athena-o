package application

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

type StaticFieldStore interface {
	GetStaticState(ctx context.Context, projectID uuid.UUID) (*ProjectStaticState, error)
	SaveStaticState(ctx context.Context, projectID uuid.UUID, static *ProjectStaticState) error
}

type ProjectRegistry interface {
	GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error)
	SetProject(ctx context.Context, projectID uuid.UUID, project *Project) error
	RemoveProject(ctx context.Context, projectID uuid.UUID) error
	ListProjects(ctx context.Context) ([]*Project, error)
	GetStaticState(ctx context.Context, projectID uuid.UUID) (*ProjectStaticState, error)
	SaveStaticState(ctx context.Context, projectID uuid.UUID, static *ProjectStaticState) error
	ListDueStaticFields(ctx context.Context, now time.Time, limit int) ([]StaticFieldResolveRequest, error)
}

var _ ProjectRegistry = &projectRegistryImpl{}

type projectRegistryImpl struct {
	mu       sync.RWMutex
	Projects map[uuid.UUID]*Project
}

func NewProjectRegistry() ProjectRegistry {
	return &projectRegistryImpl{
		Projects: make(map[uuid.UUID]*Project),
	}
}

func (r *projectRegistryImpl) GetProject(ctx context.Context, projectID uuid.UUID) (*Project, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	project, ok := r.Projects[projectID]
	if !ok {
		return nil, false, nil
	}

	projectCopy := *project
	return &projectCopy, true, nil
}

func (r *projectRegistryImpl) SetProject(ctx context.Context, projectID uuid.UUID, project *Project) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Projects[projectID] = project
	return nil
}

func (r *projectRegistryImpl) RemoveProject(ctx context.Context, projectID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.Projects, projectID)
	return nil
}

func (r *projectRegistryImpl) ListProjects(ctx context.Context) ([]*Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	projects := make([]*Project, 0, len(r.Projects))
	for _, project := range r.Projects {
		projectCopy := *project
		projects = append(projects, &projectCopy)
	}

	return projects, nil
}

func (s *projectRegistryImpl) GetStaticState(
	ctx context.Context,
	projectID uuid.UUID,
) (*ProjectStaticState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.Projects[projectID]
	if !ok {
		return nil, fmt.Errorf("project not found: %s", projectID)
	}

	static := project.StaticState
	return &static, nil
}

func (s *projectRegistryImpl) SaveStaticState(
	ctx context.Context,
	projectID uuid.UUID,
	static *ProjectStaticState,
) error {
	if static == nil {
		return fmt.Errorf("static state is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.Projects[projectID]
	if !ok {
		return fmt.Errorf("project not found: %s", projectID)
	}

	project.StaticState = *static
	return nil
}

func (s *projectRegistryImpl) ListDueStaticFields(
	ctx context.Context,
	now time.Time,
	limit int,
) ([]StaticFieldResolveRequest, error) {
	if limit <= 0 {
		limit = 1000
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	requests := make([]StaticFieldResolveRequest, 0, limit)

	for projectID, project := range s.Projects {
		fields := DueStaticFields(project.StaticState, now)

		for _, field := range fields {
			requests = append(requests, StaticFieldResolveRequest{
				ProjectID: projectID,
				Field:     field,
				Force:     false,
				Reason:    StaticResolveReasonRetry,
				CreatedAt: now,
			})

			if len(requests) >= limit {
				return requests, nil
			}
		}
	}

	return requests, nil
}

func DueStaticFields(static ProjectStaticState, now time.Time) []StaticField {
	due := make([]StaticField, 0, 5)
	if static.Name.ShouldAttempt(now) {
		due = append(due, StaticFieldName)
	}
	if static.Symbol.ShouldAttempt(now) {
		due = append(due, StaticFieldSymbol)
	}
	if static.Decimals.ShouldAttempt(now) {
		due = append(due, StaticFieldDecimals)
	}
	if static.SourceCode.ShouldAttempt(now) {
		due = append(due, StaticFieldSourceCode)
	}
	if static.SourceCodeABI.ShouldAttempt(now) {
		due = append(due, StaticFieldSourceCodeABI)
	}

	return due
}
