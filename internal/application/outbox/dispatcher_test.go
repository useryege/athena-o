package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appevents "github.com/useryege/athena/internal/application/events"
	"github.com/useryege/athena/internal/application/store"
	"github.com/useryege/athena/internal/application/workflows"
)

type fakeOutboxStore struct {
	events             []store.OutboxEvent
	claimedTypes       []string
	processed          []int64
	failed             []int64
	discarded          []int64
	running            []collectionRunningMark
	completed          []collectionCompletedMark
	collectionFailures []collectionFailedMark
	nextAttempt        time.Time
	lastError          string
}

type collectionRunningMark struct {
	chainID    int64
	contract   common.Address
	workflowID string
	startedAt  time.Time
}

type collectionCompletedMark struct {
	chainID     int64
	contract    common.Address
	completedAt time.Time
}

type collectionFailedMark struct {
	chainID   int64
	contract  common.Address
	nextRunAt time.Time
	lastError string
}

func (f *fakeOutboxStore) InsertOutboxEvent(context.Context, store.CreateOutboxEventParams) (*store.OutboxEvent, error) {
	return nil, nil
}

func (f *fakeOutboxStore) ClaimOutboxEvents(context.Context, string, time.Time, int32) ([]store.OutboxEvent, error) {
	return f.events, nil
}

func (f *fakeOutboxStore) ClaimOutboxEventsByTypes(_ context.Context, _ string, _ time.Time, _ int32, types []string) ([]store.OutboxEvent, error) {
	f.claimedTypes = append([]string(nil), types...)
	allowed := map[string]struct{}{}
	for _, eventType := range types {
		allowed[eventType] = struct{}{}
	}
	items := make([]store.OutboxEvent, 0, len(f.events))
	for _, event := range f.events {
		if _, ok := allowed[event.Type]; ok {
			items = append(items, event)
		}
	}
	return items, nil
}

func (f *fakeOutboxStore) MarkOutboxEventProcessed(_ context.Context, id int64) error {
	f.processed = append(f.processed, id)
	return nil
}

func (f *fakeOutboxStore) MarkOutboxEventFailed(_ context.Context, id int64, nextAttemptAt time.Time, lastError string) error {
	f.failed = append(f.failed, id)
	f.nextAttempt = nextAttemptAt
	f.lastError = lastError
	return nil
}

func (f *fakeOutboxStore) MarkOutboxEventDiscarded(_ context.Context, id int64, lastError string) error {
	f.discarded = append(f.discarded, id)
	f.lastError = lastError
	return nil
}

func (f *fakeOutboxStore) GetProjectCollectionState(context.Context, int64, common.Address) (*store.ProjectCollectionState, error) {
	return nil, nil
}

func (f *fakeOutboxStore) ListProjectComponentStates(context.Context, int64, common.Address) ([]store.ProjectComponentState, error) {
	return nil, nil
}

func (f *fakeOutboxStore) MarkProjectCollectionRunning(_ context.Context, chainID int64, contract common.Address, workflowID string, startedAt time.Time) error {
	f.running = append(f.running, collectionRunningMark{chainID: chainID, contract: contract, workflowID: workflowID, startedAt: startedAt})
	return nil
}

func (f *fakeOutboxStore) MarkProjectCollectionCompleted(_ context.Context, chainID int64, contract common.Address, completedAt time.Time) error {
	f.completed = append(f.completed, collectionCompletedMark{chainID: chainID, contract: contract, completedAt: completedAt})
	return nil
}

func (f *fakeOutboxStore) MarkProjectCollectionFailed(_ context.Context, chainID int64, contract common.Address, nextRunAt time.Time, lastError string) error {
	f.collectionFailures = append(f.collectionFailures, collectionFailedMark{chainID: chainID, contract: contract, nextRunAt: nextRunAt, lastError: lastError})
	return nil
}

type fakeWorkflowStarter struct {
	candidates  []workflows.CandidateQualificationInput
	collections []workflows.ProjectCollectionInput
	workflowID  string
	err         error
}

type fakeProjectEventProducer struct {
	records []projectEventRecord
}

type projectEventRecord struct {
	topic    string
	key      string
	envelope appevents.Envelope
}

func (f *fakeProjectEventProducer) Publish(_ context.Context, topic string, key string, envelope appevents.Envelope) error {
	f.records = append(f.records, projectEventRecord{topic: topic, key: key, envelope: envelope})
	return nil
}

func (f *fakeProjectEventProducer) Close() {}

func (f *fakeWorkflowStarter) StartCandidateQualification(_ context.Context, input workflows.CandidateQualificationInput) error {
	f.candidates = append(f.candidates, input)
	return f.err
}

func (f *fakeWorkflowStarter) StartProjectCollection(_ context.Context, input workflows.ProjectCollectionInput) (string, error) {
	f.collections = append(f.collections, input)
	if f.err != nil {
		return "", f.err
	}
	if f.workflowID != "" {
		return f.workflowID, nil
	}
	return workflows.ProjectCollectionWorkflowID(input.Project.ChainID, input.Project.Contract.Hex()), nil
}

func TestDispatcherClaimsWorkflowAndKafkaProjectOutboxTypes(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	storeFake := &fakeOutboxStore{events: []store.OutboxEvent{
		newOutboxEvent(1, store.OutboxTypeCandidateQualificationRequested, 56, map[string]any{
			"project": workflows.ProjectRef{ChainID: 56, Contract: contract},
			"source":  "catch_up",
		}),
		newOutboxEvent(2, store.OutboxTypeProjectCollectionRequested, 56, map[string]any{
			"project": workflows.ProjectRef{ChainID: 56, Contract: contract},
			"reason":  "pair_swap",
		}),
		newOutboxEvent(3, store.OutboxTypeKafkaProjectEventPublish, 56, map[string]any{
			"project": workflows.ProjectRef{ChainID: 56, Contract: contract},
			"action":  "collection_completed",
		}),
	}}
	starter := &fakeWorkflowStarter{}
	producer := &fakeProjectEventProducer{}
	dispatcher, err := NewDispatcher(Options{Store: storeFake, Starter: starter, ProjectEventProducer: producer})
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}

	count, err := dispatcher.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process once: %v", err)
	}

	if count != 3 {
		t.Fatalf("claimed count = %d, want 3 dispatchable events", count)
	}
	if len(storeFake.claimedTypes) != 3 || !contains(storeFake.claimedTypes, store.OutboxTypeKafkaProjectEventPublish) {
		t.Fatalf("claimed types = %#v, want workflow and kafka project event types", storeFake.claimedTypes)
	}
	if len(starter.candidates) != 1 || len(starter.collections) != 1 {
		t.Fatalf("started candidates=%d collections=%d, want 1/1", len(starter.candidates), len(starter.collections))
	}
	if len(storeFake.running) != 1 || storeFake.running[0].chainID != 56 || storeFake.running[0].contract != contract || storeFake.running[0].workflowID == "" {
		t.Fatalf("collection running marks = %#v, want workflow running state for %s", storeFake.running, contract.Hex())
	}
	if len(producer.records) != 1 {
		t.Fatalf("project events published = %d, want 1", len(producer.records))
	}
	if producer.records[0].topic != appevents.TopicApplicationProjectV1 {
		t.Fatalf("topic = %q, want application project topic", producer.records[0].topic)
	}
	if !equalIDs(storeFake.processed, []int64{1, 2, 3}) {
		t.Fatalf("processed ids = %#v, want [1 2 3]", storeFake.processed)
	}
}

func TestDispatcherRetriesTransientWorkflowStartError(t *testing.T) {
	now := time.Date(2026, 6, 5, 12, 0, 0, 0, time.UTC)
	contract := common.HexToAddress("0x1000000000000000000000000000000000000002")
	storeFake := &fakeOutboxStore{events: []store.OutboxEvent{
		newOutboxEvent(10, store.OutboxTypeCandidateQualificationRequested, 1, map[string]any{
			"project": workflows.ProjectRef{ChainID: 1, Contract: contract},
		}),
	}}
	storeFake.events[0].Attempts = 2
	dispatcher, err := NewDispatcher(Options{
		Store:                storeFake,
		Starter:              &fakeWorkflowStarter{err: errors.New("temporal unavailable")},
		Now:                  func() time.Time { return now },
		RetryInitialInterval: time.Second,
		RetryMaxInterval:     time.Minute,
	})
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}

	if _, err := dispatcher.ProcessOnce(context.Background()); err == nil {
		t.Fatalf("process once error = nil, want transient dispatch error")
	}

	if !equalIDs(storeFake.failed, []int64{10}) {
		t.Fatalf("failed ids = %#v, want [10]", storeFake.failed)
	}
	if !storeFake.nextAttempt.Equal(now.Add(2 * time.Second)) {
		t.Fatalf("next attempt = %s, want %s", storeFake.nextAttempt, now.Add(2*time.Second))
	}
}

func TestDispatcherDiscardsBadPayload(t *testing.T) {
	storeFake := &fakeOutboxStore{events: []store.OutboxEvent{{
		ID:      20,
		Type:    store.OutboxTypeProjectCollectionRequested,
		ChainID: 56,
		Payload: []byte("{"),
	}}}
	dispatcher, err := NewDispatcher(Options{Store: storeFake, Starter: &fakeWorkflowStarter{}})
	if err != nil {
		t.Fatalf("new dispatcher: %v", err)
	}

	count, err := dispatcher.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process once: %v", err)
	}

	if count != 1 {
		t.Fatalf("claimed count = %d, want 1", count)
	}
	if !equalIDs(storeFake.discarded, []int64{20}) {
		t.Fatalf("discarded ids = %#v, want [20]", storeFake.discarded)
	}
}

func newOutboxEvent(id int64, eventType string, chainID int64, payload any) store.OutboxEvent {
	raw, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}
	return store.OutboxEvent{ID: id, Type: eventType, ChainID: chainID, Payload: raw}
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func equalIDs(got []int64, want []int64) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
