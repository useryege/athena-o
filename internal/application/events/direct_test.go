package events

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appmodel "github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

type directProducerQuerierFake struct {
	appsqlc.Querier

	upsertProjectCalls     int
	upsertBytecodeCalls    int
	upsertDeploymentCalls  int
	collectionRequestCalls int
	outboxInserts          int

	lastProject    appsqlc.UpsertProjectFromDiscoveryParams
	lastDeployment appsqlc.UpsertContractBytecodeDeploymentParams
	lastCodeHash   []byte
	lastOutbox     appsqlc.InsertOutboxEventParams
}

func (f *directProducerQuerierFake) UpsertProjectFromDiscovery(_ context.Context, arg appsqlc.UpsertProjectFromDiscoveryParams) error {
	f.upsertProjectCalls++
	f.lastProject = arg
	return nil
}

func (f *directProducerQuerierFake) UpsertBytecode(_ context.Context, codeHash []byte) error {
	f.upsertBytecodeCalls++
	f.lastCodeHash = append([]byte(nil), codeHash...)
	return nil
}

func (f *directProducerQuerierFake) UpsertContractBytecodeDeployment(_ context.Context, arg appsqlc.UpsertContractBytecodeDeploymentParams) error {
	f.upsertDeploymentCalls++
	f.lastDeployment = arg
	return nil
}

func (f *directProducerQuerierFake) UpsertProjectCollectionRequest(_ context.Context, arg appsqlc.UpsertProjectCollectionRequestParams) error {
	f.collectionRequestCalls++
	return nil
}

func (f *directProducerQuerierFake) InsertOutboxEvent(_ context.Context, arg appsqlc.InsertOutboxEventParams) (appsqlc.InsertOutboxEventRow, error) {
	f.outboxInserts++
	f.lastOutbox = arg
	return appsqlc.InsertOutboxEventRow{
		ID:            int64(f.outboxInserts),
		Type:          arg.Type,
		AggregateType: arg.AggregateType,
		AggregateID:   arg.AggregateID,
		ChainID:       arg.ChainID,
		DedupKey:      arg.DedupKey,
		Payload:       arg.Payload,
		Status:        appstore.OutboxStatusPending,
	}, nil
}

func TestDirectProducerContractCreatedWritesProjectBytecodeDeploymentAndCollectionOutbox(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	creator := common.HexToAddress("0x2000000000000000000000000000000000000002")
	wethPair := common.HexToAddress("0x3000000000000000000000000000000000000003")
	usdtPair := common.HexToAddress("0x4000000000000000000000000000000000000004")
	txHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	codeHash := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	querier := &directProducerQuerierFake{}
	producer := NewDirectProducer(appstore.NewSQLStoreWithQuerier(querier))
	envelope := testEnvelope(t, EventTypeContractCreated, 56, ContractCreatedPayload{
		Contract:    contract.Hex(),
		Creator:     creator.Hex(),
		TxHash:      txHash.Hex(),
		CodeHash:    codeHash.Hex(),
		WethPair:    wethPair.Hex(),
		UsdtPair:    usdtPair.Hex(),
		BlockNumber: 123,
		BlockTime:   456,
		TxIndex:     7,
	})

	if err := producer.Publish(context.Background(), envelope); err != nil {
		t.Fatalf("publish direct event: %v", err)
	}

	if querier.upsertProjectCalls != 1 || querier.upsertBytecodeCalls != 1 || querier.upsertDeploymentCalls != 1 || querier.collectionRequestCalls != 1 || querier.outboxInserts != 1 {
		t.Fatalf("calls project/bytecode/deployment/collection/outbox = %d/%d/%d/%d/%d, want all 1", querier.upsertProjectCalls, querier.upsertBytecodeCalls, querier.upsertDeploymentCalls, querier.collectionRequestCalls, querier.outboxInserts)
	}
	if querier.lastProject.ChainID != 56 || common.BytesToAddress(querier.lastProject.Contract) != contract || common.BytesToAddress(querier.lastProject.Creator) != creator || common.BytesToHash(querier.lastProject.TxHash) != txHash {
		t.Fatalf("project params = %#v, want decoded contract/creator/tx hash", querier.lastProject)
	}
	if common.BytesToHash(querier.lastCodeHash) != codeHash {
		t.Fatalf("code hash = %s, want %s", common.BytesToHash(querier.lastCodeHash).Hex(), codeHash.Hex())
	}
	if querier.lastDeployment.ChainID != 56 || common.BytesToAddress(querier.lastDeployment.Contract) != contract || common.BytesToHash(querier.lastDeployment.CodeHash) != codeHash {
		t.Fatalf("deployment params = %#v, want chain/contract/code hash", querier.lastDeployment)
	}
	if querier.lastOutbox.Type != appstore.OutboxTypeProjectCollectionRequested || querier.lastOutbox.ChainID != 56 {
		t.Fatalf("outbox params = %#v, want project collection request on chain 56", querier.lastOutbox)
	}
	var outboxPayload struct {
		Project appmodel.ProjectRef `json:"project"`
		Reason  string              `json:"reason"`
	}
	if err := json.Unmarshal(querier.lastOutbox.Payload, &outboxPayload); err != nil {
		t.Fatalf("decode outbox payload: %v", err)
	}
	if outboxPayload.Project.ChainID != 56 || outboxPayload.Project.Contract != contract || outboxPayload.Reason != string(appmodel.ProjectDiscoverySourceFollowHeads) {
		t.Fatalf("outbox payload = %#v, want project ref and follow_heads reason", outboxPayload)
	}
}
