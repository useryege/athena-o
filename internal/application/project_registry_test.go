package application

import (
	"context"
	"errors"
	"testing"

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

func TestUpdateProjectMetaStateSourceCodeReadyPersists(t *testing.T) {
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

	state := &ProjectMeta{SourceCode: "contract A {}"}
	if err := registry.UpdateProjectMetaState(context.Background(), projectID, state); err != nil {
		t.Fatalf("update project meta state: %v", err)
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
	if updated.Meta.SourceCode != "contract A {}" {
		t.Fatalf("source code = %q, want %q", updated.Meta.SourceCode, "contract A {}")
	}
}

func TestUpdateProjectMetaStateEmptySourceCodeDoesNotPersist(t *testing.T) {
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

	state := &ProjectMeta{SourceCode: ""}
	if err := registry.UpdateProjectMetaState(context.Background(), projectID, state); err != nil {
		t.Fatalf("update project meta state: %v", err)
	}

	if store.updateSourceCodeCalls != 0 {
		t.Fatalf("update source code calls = %d, want 0", store.updateSourceCodeCalls)
	}
}

func TestUpdateProjectMetaStateSourceCodeStoreErrorReturned(t *testing.T) {
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

	state := &ProjectMeta{SourceCode: "contract B {}"}
	err := registry.UpdateProjectMetaState(context.Background(), projectID, state)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "db failed" {
		t.Fatalf("error = %v, want db failed", err)
	}
}
