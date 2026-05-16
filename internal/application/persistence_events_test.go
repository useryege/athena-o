package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	appstore "github.com/useryege/athena/internal/application/store"
)

type persistenceWriterMock struct {
	metas         []appstore.ProjectMeta
	sourceUpdates []struct {
		projectID  uuid.UUID
		sourceCode string
	}
	archives      []uuid.UUID
	unarchives    []uuid.UUID
	blacklistAdds []string
	blacklistDels []string
}

func (m *persistenceWriterMock) WriteProjectMeta(ctx context.Context, meta appstore.ProjectMeta) error {
	m.metas = append(m.metas, meta)
	return nil
}

func (m *persistenceWriterMock) WriteProjectSourceCode(ctx context.Context, projectID uuid.UUID, sourceCode string) error {
	m.sourceUpdates = append(m.sourceUpdates, struct {
		projectID  uuid.UUID
		sourceCode string
	}{projectID: projectID, sourceCode: sourceCode})
	return nil
}

func (m *persistenceWriterMock) ArchiveProject(ctx context.Context, projectID uuid.UUID) error {
	m.archives = append(m.archives, projectID)
	return nil
}

func (m *persistenceWriterMock) UnarchiveProject(ctx context.Context, projectID uuid.UUID) error {
	m.unarchives = append(m.unarchives, projectID)
	return nil
}

func (m *persistenceWriterMock) AddSourceCodeBlacklistField(ctx context.Context, field string) error {
	m.blacklistAdds = append(m.blacklistAdds, field)
	return nil
}

func (m *persistenceWriterMock) DeleteSourceCodeBlacklistField(ctx context.Context, field string) error {
	m.blacklistDels = append(m.blacklistDels, field)
	return nil
}

func TestApplyEventProjectMetaSave(t *testing.T) {
	projectID := uuid.New()
	payload, err := json.Marshal(projectMetaSavePayload{
		ProjectID:   projectID.String(),
		BlockTime:   100,
		BlockNumber: 200,
		Contract:    common.HexToAddress("0x1111111111111111111111111111111111111111").Hex(),
		Creator:     common.HexToAddress("0x2222222222222222222222222222222222222222").Hex(),
		TxHash:      common.HexToHash("0x1234").Hex(),
		TxIndex:     9,
		SourceCode:  "contract A {}",
		IsArchived:  true,
		ArchivedAt:  time.Now().UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	writer := &persistenceWriterMock{}
	bus := &RedisPersistenceEventBus{}
	if err := bus.applyEvent(context.Background(), writer, PersistenceEvent{Op: PersistenceOpProjectMetaSave, Payload: payload}); err != nil {
		t.Fatalf("apply event: %v", err)
	}
	if len(writer.metas) != 1 {
		t.Fatalf("meta writes = %d, want 1", len(writer.metas))
	}
	if writer.metas[0].ProjectID != projectID {
		t.Fatalf("project id = %s, want %s", writer.metas[0].ProjectID, projectID)
	}
}

func TestApplyEventProjectSourceCodeUpdate(t *testing.T) {
	projectID := uuid.New()
	payload, err := json.Marshal(projectSourceCodeUpdatePayload{ProjectID: projectID.String(), SourceCode: "code"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	writer := &persistenceWriterMock{}
	bus := &RedisPersistenceEventBus{}
	if err := bus.applyEvent(context.Background(), writer, PersistenceEvent{Op: PersistenceOpProjectSourceCode, Payload: payload}); err != nil {
		t.Fatalf("apply event: %v", err)
	}
	if len(writer.sourceUpdates) != 1 {
		t.Fatalf("source updates = %d, want 1", len(writer.sourceUpdates))
	}
	if writer.sourceUpdates[0].projectID != projectID {
		t.Fatalf("project id = %s, want %s", writer.sourceUpdates[0].projectID, projectID)
	}
	if writer.sourceUpdates[0].sourceCode != "code" {
		t.Fatalf("source code = %q, want %q", writer.sourceUpdates[0].sourceCode, "code")
	}
}

func TestApplyEventArchiveAndUnarchive(t *testing.T) {
	projectID := uuid.New()
	writer := &persistenceWriterMock{}
	bus := &RedisPersistenceEventBus{}

	if err := bus.applyEvent(context.Background(), writer, PersistenceEvent{Op: PersistenceOpProjectArchive, ProjectID: projectID.String()}); err != nil {
		t.Fatalf("apply archive event: %v", err)
	}
	if err := bus.applyEvent(context.Background(), writer, PersistenceEvent{Op: PersistenceOpProjectUnarchive, ProjectID: projectID.String()}); err != nil {
		t.Fatalf("apply unarchive event: %v", err)
	}
	if len(writer.archives) != 1 || writer.archives[0] != projectID {
		t.Fatalf("archives = %v, want [%s]", writer.archives, projectID)
	}
	if len(writer.unarchives) != 1 || writer.unarchives[0] != projectID {
		t.Fatalf("unarchives = %v, want [%s]", writer.unarchives, projectID)
	}
}

func TestApplyEventBlacklistOps(t *testing.T) {
	writer := &persistenceWriterMock{}
	bus := &RedisPersistenceEventBus{}

	if err := bus.applyEvent(context.Background(), writer, PersistenceEvent{Op: PersistenceOpSourceBlacklistAdd, Field: "owner"}); err != nil {
		t.Fatalf("apply add event: %v", err)
	}
	payload, err := json.Marshal(blacklistFieldPayload{Field: "admin"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	if err := bus.applyEvent(context.Background(), writer, PersistenceEvent{Op: PersistenceOpSourceBlacklistDelete, Payload: payload}); err != nil {
		t.Fatalf("apply delete event: %v", err)
	}
	if len(writer.blacklistAdds) != 1 || writer.blacklistAdds[0] != "owner" {
		t.Fatalf("blacklist adds = %v, want [owner]", writer.blacklistAdds)
	}
	if len(writer.blacklistDels) != 1 || writer.blacklistDels[0] != "admin" {
		t.Fatalf("blacklist dels = %v, want [admin]", writer.blacklistDels)
	}
}
