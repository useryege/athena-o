package store

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appmodel "github.com/useryege/athena/internal/application/model"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

type projectIntakeQuerierFake struct {
	appsqlc.Querier

	upsertProjectCalls     int
	collectionRequestCalls int
	newOutboxInserts       int
	outboxByDedup          map[string]appsqlc.InsertOutboxEventRow

	lastProject    appsqlc.UpsertProjectFromDiscoveryParams
	lastOutbox     appsqlc.InsertOutboxEventParams
	lastCollection appsqlc.UpsertProjectCollectionRequestParams
}

func (f *projectIntakeQuerierFake) UpsertProjectFromDiscovery(_ context.Context, arg appsqlc.UpsertProjectFromDiscoveryParams) error {
	f.upsertProjectCalls++
	f.lastProject = arg
	return nil
}

func (f *projectIntakeQuerierFake) UpsertProjectCollectionRequest(_ context.Context, arg appsqlc.UpsertProjectCollectionRequestParams) error {
	f.collectionRequestCalls++
	f.lastCollection = arg
	return nil
}

func (f *projectIntakeQuerierFake) InsertOutboxEvent(_ context.Context, arg appsqlc.InsertOutboxEventParams) (appsqlc.InsertOutboxEventRow, error) {
	f.lastOutbox = arg
	if f.outboxByDedup == nil {
		f.outboxByDedup = map[string]appsqlc.InsertOutboxEventRow{}
	}
	key := arg.Type + "|" + arg.DedupKey
	if row, ok := f.outboxByDedup[key]; ok {
		return row, nil
	}
	f.newOutboxInserts++
	row := appsqlc.InsertOutboxEventRow{
		ID:            int64(f.newOutboxInserts),
		Type:          arg.Type,
		AggregateType: arg.AggregateType,
		AggregateID:   arg.AggregateID,
		ChainID:       arg.ChainID,
		DedupKey:      arg.DedupKey,
		Payload:       arg.Payload,
		Status:        "pending",
	}
	f.outboxByDedup[key] = row
	return row, nil
}

func TestUpsertProjectCandidateAndEnqueueQualificationWritesProjectAndCollectionOutbox(t *testing.T) {
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	creator := common.HexToAddress("0x1000000000000000000000000000000000000002")
	wethPair := common.HexToAddress("0x1000000000000000000000000000000000000003")
	usdtPair := common.HexToAddress("0x1000000000000000000000000000000000000004")
	txHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	querier := &projectIntakeQuerierFake{}
	store := NewSQLStoreWithQuerier(querier)

	candidate := appmodel.DiscoveredProjectCandidate{
		ChainID:     56,
		Contract:    contract,
		Creator:     creator,
		TxHash:      txHash,
		BlockNumber: 10,
		BlockTime:   20,
		TxIndex:     1,
		WethPair:    wethPair,
		UsdtPair:    usdtPair,
		Source:      appmodel.ProjectDiscoverySourceCatchUp,
	}
	if err := store.UpsertProjectCandidateAndEnqueueQualification(ctx, candidate); err != nil {
		t.Fatalf("first intake: %v", err)
	}
	if err := store.UpsertProjectCandidateAndEnqueueQualification(ctx, candidate); err != nil {
		t.Fatalf("second intake: %v", err)
	}

	if querier.upsertProjectCalls != 2 {
		t.Fatalf("project upserts = %d, want 2", querier.upsertProjectCalls)
	}
	if querier.collectionRequestCalls != 2 {
		t.Fatalf("collection request calls = %d, want 2", querier.collectionRequestCalls)
	}
	if querier.newOutboxInserts != 1 {
		t.Fatalf("new outbox inserts = %d, want 1 deduped pending insert", querier.newOutboxInserts)
	}
	if querier.lastProject.ChainID != 56 || common.BytesToAddress(querier.lastProject.Contract) != contract {
		t.Fatalf("project params = %#v, want chain/contract", querier.lastProject)
	}
	if common.BytesToAddress(querier.lastProject.Creator) != creator || common.BytesToHash(querier.lastProject.TxHash) != txHash {
		t.Fatalf("project base params = %#v, want creator/tx hash", querier.lastProject)
	}
	if common.BytesToAddress(querier.lastProject.WethPair) != wethPair || common.BytesToAddress(querier.lastProject.UsdtPair) != usdtPair {
		t.Fatalf("project pair params = %#v, want weth/usdt pairs", querier.lastProject)
	}
	if querier.lastOutbox.Type != OutboxTypeProjectCollectionRequested || querier.lastOutbox.ChainID != 56 {
		t.Fatalf("outbox params = %#v, want project collection on chain 56", querier.lastOutbox)
	}
	if querier.lastOutbox.DedupKey != projectDedupKey(contract) {
		t.Fatalf("dedup key = %q, want contract key", querier.lastOutbox.DedupKey)
	}
	var outboxPayload struct {
		Project appmodel.ProjectRef `json:"project"`
		Reason  string              `json:"reason"`
	}
	if err := json.Unmarshal(querier.lastOutbox.Payload, &outboxPayload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if outboxPayload.Project.ChainID != 56 || outboxPayload.Project.Contract != contract || outboxPayload.Reason != string(appmodel.ProjectDiscoverySourceCatchUp) {
		t.Fatalf("outbox payload = %#v, want project ref and discovery source reason", outboxPayload)
	}
}

func TestEnqueueProjectCollectionWritesCollectionStateAndOutbox(t *testing.T) {
	ctx := context.Background()
	contract := common.HexToAddress("0x1000000000000000000000000000000000000003")
	querier := &projectIntakeQuerierFake{}
	store := NewSQLStoreWithQuerier(querier)

	if err := store.EnqueueProjectCollection(ctx, appmodel.ProjectRef{ChainID: 1, Contract: contract}, "pair_swap"); err != nil {
		t.Fatalf("enqueue collection: %v", err)
	}

	if querier.collectionRequestCalls != 1 {
		t.Fatalf("collection request calls = %d, want 1", querier.collectionRequestCalls)
	}
	if querier.newOutboxInserts != 1 {
		t.Fatalf("outbox inserts = %d, want 1", querier.newOutboxInserts)
	}
	if querier.lastCollection.ChainID != 1 || common.BytesToAddress(querier.lastCollection.ProjectContract) != contract {
		t.Fatalf("collection params = %#v, want chain/contract", querier.lastCollection)
	}
	if querier.lastOutbox.Type != OutboxTypeProjectCollectionRequested || querier.lastOutbox.ChainID != 1 {
		t.Fatalf("outbox params = %#v, want project collection on chain 1", querier.lastOutbox)
	}
}
