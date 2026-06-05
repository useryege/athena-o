package ingest

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	appevents "github.com/useryege/athena/internal/application/events"
	appstore "github.com/useryege/athena/internal/application/store"
)

type readerFake struct {
	latest      uint64
	blocks      map[uint64]Block
	latestCalls *int
	readNumbers *[]uint64
}

func (f *readerFake) LatestBlockNumber(context.Context) (uint64, error) {
	if f.latestCalls != nil {
		*f.latestCalls = *f.latestCalls + 1
	}
	return f.latest, nil
}

func (f *readerFake) ReadBlock(_ context.Context, number uint64) (Block, error) {
	if f.readNumbers != nil {
		*f.readNumbers = append(*f.readNumbers, number)
	}
	return f.blocks[number], nil
}

type producerFake struct {
	failAfter int
	records   []publishedRecord
}

type publishedRecord struct {
	topic string
	key   string
	event appevents.Envelope
}

func (f *producerFake) Publish(_ context.Context, topic string, key string, envelope appevents.Envelope) error {
	if f.failAfter > 0 && len(f.records) >= f.failAfter {
		return errors.New("publish failed")
	}
	f.records = append(f.records, publishedRecord{topic: topic, key: key, event: envelope})
	return nil
}

type checkpointStoreFake struct {
	items map[int64]appstore.ChainIngestCheckpoint
}

func (f *checkpointStoreFake) GetChainIngestCheckpoint(_ context.Context, chainID int64) (*appstore.ChainIngestCheckpoint, error) {
	if f.items == nil {
		return nil, nil
	}
	item, ok := f.items[chainID]
	if !ok {
		return nil, nil
	}
	return &item, nil
}

func (f *checkpointStoreFake) UpsertChainIngestCheckpoint(_ context.Context, item appstore.ChainIngestCheckpoint) (*appstore.ChainIngestCheckpoint, error) {
	if f.items == nil {
		f.items = map[int64]appstore.ChainIngestCheckpoint{}
	}
	f.items[item.ChainID] = item
	return &item, nil
}

func TestIngestorPublishesEventsAndAdvancesCheckpoint(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	pair := common.HexToAddress("0x2000000000000000000000000000000000000002")
	producer := &producerFake{}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		56: {ChainID: 56, Status: appstore.ChainIngestStatusRunning},
	}}
	ingestor, err := NewIngestor(Options{
		ChainID:           56,
		ConfirmationDepth: 2,
		StartBlock:        10,
		Reader: &readerFake{
			latest: 12,
			blocks: map[uint64]Block{
				10: {
					ChainID: 56,
					Number:  10,
					Hash:    common.HexToHash("0x10"),
					Time:    100,
					ContractCreations: []ContractCreated{{
						Contract: contract,
						Creator:  common.HexToAddress("0x3000000000000000000000000000000000000003"),
						TxHash:   common.HexToHash("0xabc"),
						TxIndex:  1,
					}},
					DexSwaps: []DexSwap{{Pair: pair, TxHash: common.HexToHash("0xdef")}},
				},
			},
		},
		Producer: producer,
		Store:    store,
	})
	if err != nil {
		t.Fatalf("NewIngestor: %v", err)
	}

	count, err := ingestor.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessOnce: %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}
	if len(producer.records) != 2 {
		t.Fatalf("published records = %d, want contract+swap", len(producer.records))
	}
	checkpoint := store.items[56]
	if checkpoint.CursorBlockNumber != 10 || checkpoint.FinalizedBlockNumber != 10 {
		t.Fatalf("checkpoint = %#v, want block 10", checkpoint)
	}
}

func TestIngestorDoesNotAdvanceCheckpointWhenKafkaPublishFails(t *testing.T) {
	producer := &producerFake{failAfter: 1}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		1: {ChainID: 1, Status: appstore.ChainIngestStatusRunning},
	}}
	ingestor, err := NewIngestor(Options{
		ChainID:           1,
		ConfirmationDepth: 1,
		StartBlock:        5,
		Reader: &readerFake{latest: 6, blocks: map[uint64]Block{
			5: {
				ChainID: 1,
				Number:  5,
				Hash:    common.HexToHash("0x5"),
				ContractCreations: []ContractCreated{{
					Contract: common.HexToAddress("0x1000000000000000000000000000000000000001"),
				}},
				DexSwaps: []DexSwap{{Pair: common.HexToAddress("0x2000000000000000000000000000000000000002")}},
			},
		}},
		Producer: producer,
		Store:    store,
	})
	if err != nil {
		t.Fatalf("NewIngestor: %v", err)
	}
	if _, err := ingestor.ProcessOnce(context.Background()); err == nil {
		t.Fatalf("ProcessOnce error = nil, want publish failure")
	}
	if checkpoint := store.items[1]; checkpoint.CursorBlockNumber != 0 || checkpoint.FinalizedBlockNumber != 0 {
		t.Fatalf("checkpoint = %#v, want no progress after publish failure", checkpoint)
	}
}

func TestIngestorCheckpointsAreChainScoped(t *testing.T) {
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{}}
	for _, chainID := range []int64{1, 56} {
		store.items[chainID] = appstore.ChainIngestCheckpoint{ChainID: chainID, Status: appstore.ChainIngestStatusRunning}
		ingestor, err := NewIngestor(Options{
			ChainID:           chainID,
			ConfirmationDepth: 1,
			StartBlock:        8,
			Reader:            &readerFake{latest: 9, blocks: map[uint64]Block{8: {ChainID: chainID, Number: 8, Hash: common.HexToHash("0x8")}}},
			Producer:          &producerFake{},
			Store:             store,
		})
		if err != nil {
			t.Fatalf("NewIngestor chain %d: %v", chainID, err)
		}
		if _, err := ingestor.ProcessOnce(context.Background()); err != nil {
			t.Fatalf("ProcessOnce chain %d: %v", chainID, err)
		}
	}
	if len(store.items) != 2 {
		t.Fatalf("checkpoint chains = %d, want 2", len(store.items))
	}
}

func TestIngestorStoppedCheckpointDoesNotReadNodeOrPublish(t *testing.T) {
	latestCalls := 0
	readNumbers := []uint64{}
	producer := &producerFake{}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		56: {ChainID: 56, Status: appstore.ChainIngestStatusStopped},
	}}
	ingestor, err := NewIngestor(Options{
		ChainID:           56,
		ConfirmationDepth: 1,
		StartBlock:        1,
		Reader:            &readerFake{latest: 100, latestCalls: &latestCalls, readNumbers: &readNumbers},
		Producer:          producer,
		Store:             store,
	})
	if err != nil {
		t.Fatalf("NewIngestor: %v", err)
	}

	count, err := ingestor.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessOnce: %v", err)
	}

	if count != 0 || latestCalls != 0 || len(readNumbers) != 0 || len(producer.records) != 0 {
		t.Fatalf("count=%d latestCalls=%d reads=%#v published=%d, want stopped no-op", count, latestCalls, readNumbers, len(producer.records))
	}
}

func TestIngestorContinuesFromCheckpointCursor(t *testing.T) {
	readNumbers := []uint64{}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		1: {ChainID: 1, Status: appstore.ChainIngestStatusRunning, CursorBlockNumber: 7},
	}}
	ingestor, err := NewIngestor(Options{
		ChainID:           1,
		ConfirmationDepth: 1,
		StartBlock:        5,
		Reader: &readerFake{
			latest:      10,
			readNumbers: &readNumbers,
			blocks: map[uint64]Block{
				8: {ChainID: 1, Number: 8, Hash: common.HexToHash("0x8")},
				9: {ChainID: 1, Number: 9, Hash: common.HexToHash("0x9")},
			},
		},
		Producer: &producerFake{},
		Store:    store,
	})
	if err != nil {
		t.Fatalf("NewIngestor: %v", err)
	}

	count, err := ingestor.ProcessOnce(context.Background())
	if err != nil {
		t.Fatalf("ProcessOnce: %v", err)
	}

	if count != 2 || len(readNumbers) != 2 || readNumbers[0] != 8 || readNumbers[1] != 9 {
		t.Fatalf("count=%d reads=%#v, want blocks 8 and 9", count, readNumbers)
	}
	if checkpoint := store.items[1]; checkpoint.CursorBlockNumber != 9 || checkpoint.FinalizedBlockNumber != 9 {
		t.Fatalf("checkpoint = %#v, want block 9", checkpoint)
	}
}
