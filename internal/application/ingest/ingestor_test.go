package ingest

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	appevents "github.com/useryege/athena/internal/application/events"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
)

type readerFake struct {
	latest uint64
	blocks map[uint64]Block
	codes  map[common.Address][]byte
}

func (f *readerFake) LatestBlockNumber(context.Context) (uint64, error) {
	return f.latest, nil
}

func (f *readerFake) ReadBlock(_ context.Context, number uint64) (Block, error) {
	return f.blocks[number], nil
}

func (f *readerFake) ReadContractCode(_ context.Context, contract common.Address) ([]byte, error) {
	return append([]byte(nil), f.codes[contract]...), nil
}

type chainValidatorFake struct {
	tokens []model.TokenValidation
	pairs  []model.PairValidation
}

func (f *chainValidatorFake) ValidateERC20(context.Context, []common.Address) ([]model.TokenValidation, error) {
	return append([]model.TokenValidation(nil), f.tokens...), nil
}

func (f *chainValidatorFake) ValidatePairs(context.Context, []common.Address) ([]model.PairValidation, error) {
	return append([]model.PairValidation(nil), f.pairs...), nil
}

type checkpointStoreFake struct {
	checkpoint *appstore.ChainIngestCheckpoint
	upserts    []appstore.ChainIngestCheckpoint
}

func (f *checkpointStoreFake) GetChainIngestCheckpoint(context.Context, int64) (*appstore.ChainIngestCheckpoint, error) {
	if f.checkpoint == nil {
		return nil, nil
	}
	item := *f.checkpoint
	return &item, nil
}

func (f *checkpointStoreFake) UpsertChainIngestCheckpoint(_ context.Context, item appstore.ChainIngestCheckpoint) (*appstore.ChainIngestCheckpoint, error) {
	f.upserts = append(f.upserts, item)
	f.checkpoint = &item
	return &item, nil
}

type projectEventStoreFake struct {
	candidates        []model.DiscoveredProjectCandidate
	collections       []model.ProjectRef
	collectionReasons []string
	projectsByPair    map[common.Address][]model.Project
	err               error
}

func (f *projectEventStoreFake) UpsertProjectCandidateAndEnqueueQualification(_ context.Context, candidate model.DiscoveredProjectCandidate) error {
	if f.err != nil {
		return f.err
	}
	f.candidates = append(f.candidates, candidate)
	return nil
}

func (f *projectEventStoreFake) EnqueueProjectCollection(_ context.Context, ref model.ProjectRef, reason string) error {
	if f.err != nil {
		return f.err
	}
	f.collections = append(f.collections, ref)
	f.collectionReasons = append(f.collectionReasons, reason)
	return nil
}

func (f *projectEventStoreFake) ListProjectMetasByPairAddresses(_ context.Context, _ int64, pairs []common.Address) ([]model.Project, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(pairs) == 0 {
		return nil, nil
	}
	return append([]model.Project(nil), f.projectsByPair[pairs[0]]...), nil
}

func TestIngestorDirectProducerStoresProjectAndAdvancesCheckpoint(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	creator := common.HexToAddress("0x2000000000000000000000000000000000000002")
	txHash := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	wethPair := common.HexToAddress("0x3000000000000000000000000000000000000003")
	usdtPair := common.HexToAddress("0x4000000000000000000000000000000000000004")
	code := []byte{0x60, 0x00, 0x60, 0x01}
	reader := &readerFake{
		latest: 11,
		blocks: map[uint64]Block{
			10: {
				ChainID: 56,
				Number:  10,
				Hash:    common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				Time:    123,
				ContractCreations: []ContractCreated{{
					Contract: contract,
					Creator:  creator,
					TxHash:   txHash,
					TxIndex:  7,
				}},
			},
		},
		codes: map[common.Address][]byte{contract: code},
	}
	checkpoints := &checkpointStoreFake{checkpoint: &appstore.ChainIngestCheckpoint{
		ChainID:           56,
		CursorBlockNumber: 9,
		Status:            appstore.ChainIngestStatusRunning,
	}}
	projectEvents := &projectEventStoreFake{}
	ingestor, err := NewIngestor(Options{
		ChainID:           56,
		ConfirmationDepth: 1,
		StartBlock:        10,
		Reader:            reader,
		Producer:          appevents.NewDirectProducer(projectEvents),
		Store:             checkpoints,
		ChainValidator: &chainValidatorFake{tokens: []model.TokenValidation{{
			IsValidERC20: true,
			WethPair:     wethPair,
			UsdtPair:     usdtPair,
		}}},
	})
	if err != nil {
		t.Fatalf("new ingestor: %v", err)
	}

	processed, err := ingestor.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("process once: %v", err)
	}

	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}
	if len(projectEvents.candidates) != 1 {
		t.Fatalf("candidates = %d, want 1", len(projectEvents.candidates))
	}
	candidate := projectEvents.candidates[0]
	if candidate.ChainID != 56 || candidate.Contract != contract || candidate.Creator != creator || candidate.TxHash != txHash || candidate.CodeHash != crypto.Keccak256Hash(code) {
		t.Fatalf("candidate = %#v, want decoded direct event fields", candidate)
	}
	if candidate.WethPair != wethPair || candidate.UsdtPair != usdtPair || candidate.Source != model.ProjectDiscoverySourceFollowHeads {
		t.Fatalf("candidate pairs/source = %#v, want validated pairs and follow_heads", candidate)
	}
	if len(checkpoints.upserts) != 1 || checkpoints.upserts[0].CursorBlockNumber != 10 {
		t.Fatalf("checkpoint upserts = %#v, want cursor advanced to block 10", checkpoints.upserts)
	}
}

func TestIngestorDirectProducerStoreErrorDoesNotAdvanceCheckpoint(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	storeErr := errors.New("store unavailable")
	reader := &readerFake{
		latest: 11,
		blocks: map[uint64]Block{
			10: {
				ChainID: 56,
				Number:  10,
				Hash:    common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
				Time:    123,
				ContractCreations: []ContractCreated{{
					Contract: contract,
					Creator:  common.HexToAddress("0x2000000000000000000000000000000000000002"),
					TxHash:   common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111"),
				}},
			},
		},
		codes: map[common.Address][]byte{contract: []byte{0x60, 0x00}},
	}
	checkpoints := &checkpointStoreFake{checkpoint: &appstore.ChainIngestCheckpoint{
		ChainID:           56,
		CursorBlockNumber: 9,
		Status:            appstore.ChainIngestStatusRunning,
	}}
	ingestor, err := NewIngestor(Options{
		ChainID:           56,
		ConfirmationDepth: 1,
		StartBlock:        10,
		Reader:            reader,
		Producer:          appevents.NewDirectProducer(&projectEventStoreFake{err: storeErr}),
		Store:             checkpoints,
		ChainValidator: &chainValidatorFake{tokens: []model.TokenValidation{{
			IsValidERC20: true,
			WethPair:     common.HexToAddress("0x3000000000000000000000000000000000000003"),
			UsdtPair:     common.HexToAddress("0x4000000000000000000000000000000000000004"),
		}}},
	})
	if err != nil {
		t.Fatalf("new ingestor: %v", err)
	}

	processed, err := ingestor.ProcessOnce(context.Background())
	if err == nil {
		t.Fatalf("process once error = nil, want store error")
	}
	if !errors.Is(err, storeErr) {
		t.Fatalf("process once error = %v, want %v", err, storeErr)
	}
	if processed != 0 {
		t.Fatalf("processed = %d, want 0", processed)
	}
	if len(checkpoints.upserts) != 0 {
		t.Fatalf("checkpoint upserts = %#v, want none after store error", checkpoints.upserts)
	}
}
