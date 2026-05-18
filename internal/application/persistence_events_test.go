package application

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appstore "github.com/useryege/athena/internal/application/store"
)

type persistenceWriterMock struct {
	metas         []appstore.ProjectMeta
	eventLogs     []appstore.ProjectEventLog
	sourceUpdates []struct {
		contract   common.Address
		sourceCode string
	}
	archives      []common.Address
	unarchives    []common.Address
	blacklistAdds []string
	blacklistDels []string
}

func (m *persistenceWriterMock) WriteProjectMeta(ctx context.Context, meta appstore.ProjectMeta) error {
	m.metas = append(m.metas, meta)
	return nil
}

func (m *persistenceWriterMock) WriteProjectEventLog(ctx context.Context, item appstore.ProjectEventLog) error {
	m.eventLogs = append(m.eventLogs, item)
	return nil
}

func (m *persistenceWriterMock) WriteProjectSourceCode(ctx context.Context, contract common.Address, sourceCode string) error {
	m.sourceUpdates = append(m.sourceUpdates, struct {
		contract   common.Address
		sourceCode string
	}{contract: contract, sourceCode: sourceCode})
	return nil
}

func (m *persistenceWriterMock) ArchiveProject(ctx context.Context, contract common.Address) error {
	m.archives = append(m.archives, contract)
	return nil
}

func (m *persistenceWriterMock) UnarchiveProject(ctx context.Context, contract common.Address) error {
	m.unarchives = append(m.unarchives, contract)
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
	contract := common.HexToAddress("0x1111111111111111111111111111111111111111")
	payload, err := json.Marshal(projectMetaSavePayload{
		BlockTime:   100,
		BlockNumber: 200,
		Contract:    contract.Hex(),
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
	if writer.metas[0].Contract != contract {
		t.Fatalf("contract = %s, want %s", writer.metas[0].Contract, contract)
	}
}

func TestApplyEventProjectSourceCodeUpdate(t *testing.T) {
	contract := common.HexToAddress("0x3333333333333333333333333333333333333333")
	payload, err := json.Marshal(projectSourceCodeUpdatePayload{Contract: contract.Hex(), SourceCode: "code"})
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
	if writer.sourceUpdates[0].contract != contract {
		t.Fatalf("contract = %s, want %s", writer.sourceUpdates[0].contract, contract)
	}
	if writer.sourceUpdates[0].sourceCode != "code" {
		t.Fatalf("source code = %q, want %q", writer.sourceUpdates[0].sourceCode, "code")
	}
}

func TestApplyEventProjectEventLogAdd(t *testing.T) {
	contract := common.HexToAddress("0x7777777777777777777777777777777777777777")
	occurredAt := time.Now().UTC().Truncate(time.Second)
	payload, err := json.Marshal(projectEventLogAddPayload{
		Contract:       contract.Hex(),
		EventType:      2,
		OccurredAt:     occurredAt.Format(time.RFC3339Nano),
		Message:        "Contract source code opened",
		Payload:        "{}",
		IdempotencyKey: "project_source_code_opened",
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	writer := &persistenceWriterMock{}
	bus := &RedisPersistenceEventBus{}
	if err := bus.applyEvent(context.Background(), writer, PersistenceEvent{Op: PersistenceOpProjectEventLogAdd, Payload: payload}); err != nil {
		t.Fatalf("apply event: %v", err)
	}
	if len(writer.eventLogs) != 1 {
		t.Fatalf("event log writes = %d, want 1", len(writer.eventLogs))
	}
	if writer.eventLogs[0].Contract != contract {
		t.Fatalf("contract = %s, want %s", writer.eventLogs[0].Contract, contract)
	}
	if writer.eventLogs[0].EventType != 2 {
		t.Fatalf("event type = %d, want 2", writer.eventLogs[0].EventType)
	}
	if writer.eventLogs[0].IdempotencyKey != "project_source_code_opened" {
		t.Fatalf("idempotency key = %q, want %q", writer.eventLogs[0].IdempotencyKey, "project_source_code_opened")
	}
}

func TestApplyEventArchiveAndUnarchive(t *testing.T) {
	contract := common.HexToAddress("0x4444444444444444444444444444444444444444")
	writer := &persistenceWriterMock{}
	bus := &RedisPersistenceEventBus{}

	if err := bus.applyEvent(context.Background(), writer, PersistenceEvent{Op: PersistenceOpProjectArchive, Contract: contract.Hex()}); err != nil {
		t.Fatalf("apply archive event: %v", err)
	}
	if err := bus.applyEvent(context.Background(), writer, PersistenceEvent{Op: PersistenceOpProjectUnarchive, Contract: contract.Hex()}); err != nil {
		t.Fatalf("apply unarchive event: %v", err)
	}
	if len(writer.archives) != 1 || writer.archives[0] != contract {
		t.Fatalf("archives = %v, want [%s]", writer.archives, contract)
	}
	if len(writer.unarchives) != 1 || writer.unarchives[0] != contract {
		t.Fatalf("unarchives = %v, want [%s]", writer.unarchives, contract)
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
