package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	appstore "github.com/useryege/athena/internal/application/store"
)

type projectStoreMock struct {
	updateSourceCodeCalls int
	updateProjectID       uuid.UUID
	updateSourceCode      string
	updateSourceCodeErr   error
}

func (m *projectStoreMock) SaveProjectMeta(ctx context.Context, meta appstore.ProjectMeta) error {
	return nil
}

func (m *projectStoreMock) ListProjectMetas(ctx context.Context) ([]appstore.ProjectMeta, error) {
	return nil, nil
}

func (m *projectStoreMock) UpdateProjectSourceCode(ctx context.Context, projectID uuid.UUID, sourceCode string) error {
	m.updateSourceCodeCalls++
	m.updateProjectID = projectID
	m.updateSourceCode = sourceCode
	return m.updateSourceCodeErr
}

func TestUpdateProjectSourceCodeStateReadyPersists(t *testing.T) {
	store := &projectStoreMock{}
	registry := NewProjectRegistry(store)
	projectID := uuid.New()
	project := &Project{
		Meta: ProjectMeta{
			ProjectID: projectID,
			Contract:  common.HexToAddress("0x0000000000000000000000000000000000000001"),
			Creator:   common.HexToAddress("0x0000000000000000000000000000000000000002"),
		},
	}
	if err := registry.SetProject(context.Background(), projectID, project); err != nil {
		t.Fatalf("set project: %v", err)
	}

	state := &ProjectSourceCodeState{}
	state.SourceCode.MarkReady("contract A {}", time.Now())
	if err := registry.UpdateProjectSourceCodeState(context.Background(), projectID, state); err != nil {
		t.Fatalf("update source code state: %v", err)
	}

	if store.updateSourceCodeCalls != 1 {
		t.Fatalf("update source code calls = %d, want 1", store.updateSourceCodeCalls)
	}
	if store.updateProjectID != projectID {
		t.Fatalf("update project id = %s, want %s", store.updateProjectID, projectID)
	}
	if store.updateSourceCode != "contract A {}" {
		t.Fatalf("update source code = %q, want %q", store.updateSourceCode, "contract A {}")
	}

	updated, ok, err := registry.GetProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok {
		t.Fatal("project not found")
	}
	gotSourceCode, gotOK := updated.SourceCode.SourceCode.Get()
	if !gotOK || gotSourceCode != "contract A {}" {
		t.Fatalf("source code = %q, ok = %v, want %q, true", gotSourceCode, gotOK, "contract A {}")
	}
}

func TestUpdateProjectSourceCodeStateNonReadyDoesNotPersist(t *testing.T) {
	store := &projectStoreMock{}
	registry := NewProjectRegistry(store)
	projectID := uuid.New()
	project := &Project{
		Meta: ProjectMeta{
			ProjectID: projectID,
			Contract:  common.HexToAddress("0x0000000000000000000000000000000000000011"),
			Creator:   common.HexToAddress("0x0000000000000000000000000000000000000012"),
		},
	}
	if err := registry.SetProject(context.Background(), projectID, project); err != nil {
		t.Fatalf("set project: %v", err)
	}

	state := &ProjectSourceCodeState{}
	state.SourceCode.MarkFailed(errors.New("fetch failed"), time.Now())
	if err := registry.UpdateProjectSourceCodeState(context.Background(), projectID, state); err != nil {
		t.Fatalf("update source code state: %v", err)
	}

	if store.updateSourceCodeCalls != 0 {
		t.Fatalf("update source code calls = %d, want 0", store.updateSourceCodeCalls)
	}
}

func TestUpdateProjectSourceCodeStateStoreErrorReturned(t *testing.T) {
	store := &projectStoreMock{updateSourceCodeErr: errors.New("db failed")}
	registry := NewProjectRegistry(store)
	projectID := uuid.New()
	project := &Project{
		Meta: ProjectMeta{
			ProjectID: projectID,
			Contract:  common.HexToAddress("0x0000000000000000000000000000000000000021"),
			Creator:   common.HexToAddress("0x0000000000000000000000000000000000000022"),
		},
	}
	if err := registry.SetProject(context.Background(), projectID, project); err != nil {
		t.Fatalf("set project: %v", err)
	}

	state := &ProjectSourceCodeState{}
	state.SourceCode.MarkReady("contract B {}", time.Now())
	err := registry.UpdateProjectSourceCodeState(context.Background(), projectID, state)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db failed" {
		t.Fatalf("error = %v, want db failed", err)
	}
}
