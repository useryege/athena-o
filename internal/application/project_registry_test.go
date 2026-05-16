package application

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	appstore "github.com/useryege/athena/internal/application/store"
)

type registryPublisherMock struct {
	saveMetas        []appstore.ProjectMeta
	sourceCodeWrites []struct {
		projectID  uuid.UUID
		sourceCode string
	}
	saveErr       error
	sourceCodeErr error
}

func (m *registryPublisherMock) Publish(ctx context.Context, event PersistenceEvent) error {
	return nil
}

func (m *registryPublisherMock) PublishProjectMetaSave(ctx context.Context, meta appstore.ProjectMeta) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saveMetas = append(m.saveMetas, meta)
	return nil
}

func (m *registryPublisherMock) PublishProjectSourceCodeUpdate(ctx context.Context, projectID uuid.UUID, sourceCode string) error {
	if m.sourceCodeErr != nil {
		return m.sourceCodeErr
	}
	m.sourceCodeWrites = append(m.sourceCodeWrites, struct {
		projectID  uuid.UUID
		sourceCode string
	}{projectID: projectID, sourceCode: sourceCode})
	return nil
}

func (m *registryPublisherMock) PublishProjectArchive(ctx context.Context, projectID uuid.UUID) error {
	return nil
}

func (m *registryPublisherMock) PublishProjectUnarchive(ctx context.Context, projectID uuid.UUID) error {
	return nil
}

func (m *registryPublisherMock) PublishSourceCodeBlacklistAdd(ctx context.Context, field string) error {
	return nil
}

func (m *registryPublisherMock) PublishSourceCodeBlacklistDelete(ctx context.Context, field string) error {
	return nil
}

func TestProjectRegistrySetProjectPublishesMetaSave(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	projectID := uuid.New()
	project := &Project{Meta: ProjectMeta{
		ProjectID:   projectID,
		Contract:    common.HexToAddress("0x1111111111111111111111111111111111111111"),
		Creator:     common.HexToAddress("0x2222222222222222222222222222222222222222"),
		BlockNumber: 10,
		BlockTime:   11,
		TxHash:      common.HexToHash("0x1234"),
		TxIndex:     1,
	}}

	if err := registry.SetProject(context.Background(), projectID, project); err != nil {
		t.Fatalf("set project: %v", err)
	}
	if len(publisher.saveMetas) != 1 {
		t.Fatalf("meta save publishes = %d, want 1", len(publisher.saveMetas))
	}
	stored, ok, err := registry.GetProject(context.Background(), projectID)
	if err != nil {
		t.Fatalf("get project: %v", err)
	}
	if !ok || stored == nil {
		t.Fatalf("project not found after set")
	}
}

func TestProjectRegistrySetProjectPublishFailureDoesNotMutateRegistry(t *testing.T) {
	expectedErr := errors.New("publish failed")
	publisher := &registryPublisherMock{saveErr: expectedErr}
	registry := NewProjectRegistry(publisher)

	projectID := uuid.New()
	project := &Project{Meta: ProjectMeta{
		ProjectID: projectID,
		Contract:  common.HexToAddress("0x1111111111111111111111111111111111111111"),
	}}

	err := registry.SetProject(context.Background(), projectID, project)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("set project err = %v, want %v", err, expectedErr)
	}
	_, ok, getErr := registry.GetProject(context.Background(), projectID)
	if getErr != nil {
		t.Fatalf("get project: %v", getErr)
	}
	if ok {
		t.Fatalf("project should not be present when publish fails")
	}
}

func TestProjectRegistryUpdateProjectMetaStatePublishFailureKeepsOldSourceCode(t *testing.T) {
	publisher := &registryPublisherMock{}
	registry := NewProjectRegistry(publisher)

	projectID := uuid.New()
	project := &Project{Meta: ProjectMeta{
		ProjectID:   projectID,
		Contract:    common.HexToAddress("0x1111111111111111111111111111111111111111"),
		SourceCode:  "old",
		BlockNumber: 1,
	}}
	if err := registry.LoadProject(context.Background(), projectID, project); err != nil {
		t.Fatalf("load project: %v", err)
	}

	expectedErr := errors.New("source update failed")
	publisher.sourceCodeErr = expectedErr
	err := registry.UpdateProjectMetaState(context.Background(), projectID, &ProjectMeta{SourceCode: "new"})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("update err = %v, want %v", err, expectedErr)
	}
	stored, ok, getErr := registry.GetProject(context.Background(), projectID)
	if getErr != nil {
		t.Fatalf("get project: %v", getErr)
	}
	if !ok || stored == nil {
		t.Fatalf("project not found")
	}
	if stored.Meta.SourceCode != "old" {
		t.Fatalf("source code = %q, want %q", stored.Meta.SourceCode, "old")
	}
}
