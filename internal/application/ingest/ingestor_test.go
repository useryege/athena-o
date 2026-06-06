package ingest

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	appevents "github.com/useryege/athena/internal/application/events"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
)

type readerFake struct {
	latest       uint64
	blocks       map[uint64]Block
	latestCalls  *int
	readNumbers  *[]uint64
	codeByAddr   map[common.Address][]byte
	codeErr      error
	codeRequests []readContractCodeRequest
}

type readContractCodeRequest struct {
	contract    common.Address
	blockNumber uint64
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

func (f *readerFake) ReadContractCode(_ context.Context, contract common.Address) ([]byte, error) {
	f.codeRequests = append(f.codeRequests, readContractCodeRequest{contract: contract})
	if f.codeErr != nil {
		return nil, f.codeErr
	}
	if f.codeByAddr != nil {
		return append([]byte(nil), f.codeByAddr[contract]...), nil
	}
	return []byte{0x60, 0x00}, nil
}

type producerFake struct {
	failAfter int
	failTopic string
	records   []publishedRecord
}

type publishedRecord struct {
	topic string
	key   string
	event appevents.Envelope
}

func (f *producerFake) Publish(_ context.Context, topic string, key string, envelope appevents.Envelope) error {
	if f.failTopic != "" && topic == f.failTopic {
		return errors.New("publish failed")
	}
	if f.failAfter > 0 && len(f.records) >= f.failAfter {
		return errors.New("publish failed")
	}
	f.records = append(f.records, publishedRecord{topic: topic, key: key, event: envelope})
	return nil
}

type chainValidatorFake struct {
	tokenErr          error
	tokenResults      []model.TokenValidation
	contracts         []common.Address
	defaultWeth       common.Address
	defaultUsdt       common.Address
	pairErr           error
	pairResults       []model.PairValidation
	pairs             []common.Address
	defaultPairToken0 common.Address
	defaultPairToken1 common.Address
}

func (f *chainValidatorFake) ValidateERC20(_ context.Context, contracts []common.Address) ([]model.TokenValidation, error) {
	f.contracts = append(f.contracts, contracts...)
	if f.tokenErr != nil {
		return nil, f.tokenErr
	}
	if f.tokenResults != nil {
		return append([]model.TokenValidation(nil), f.tokenResults...), nil
	}
	results := make([]model.TokenValidation, 0, len(contracts))
	wethPair := f.defaultWeth
	if wethPair == (common.Address{}) {
		wethPair = common.HexToAddress("0x4000000000000000000000000000000000000004")
	}
	usdtPair := f.defaultUsdt
	if usdtPair == (common.Address{}) {
		usdtPair = common.HexToAddress("0x5000000000000000000000000000000000000005")
	}
	for range contracts {
		results = append(results, model.TokenValidation{
			IsValidERC20: true,
			WethPair:     wethPair,
			UsdtPair:     usdtPair,
		})
	}
	return results, nil
}

func (f *chainValidatorFake) ValidatePairs(_ context.Context, pairs []common.Address) ([]model.PairValidation, error) {
	f.pairs = append(f.pairs, pairs...)
	if f.pairErr != nil {
		return nil, f.pairErr
	}
	if f.pairResults != nil {
		return append([]model.PairValidation(nil), f.pairResults...), nil
	}
	results := make([]model.PairValidation, 0, len(pairs))
	token0 := f.defaultPairToken0
	if token0 == (common.Address{}) {
		token0 = common.HexToAddress("0x6000000000000000000000000000000000000006")
	}
	token1 := f.defaultPairToken1
	if token1 == (common.Address{}) {
		token1 = common.HexToAddress("0x7000000000000000000000000000000000000007")
	}
	for range pairs {
		results = append(results, model.PairValidation{
			IsValidPancakePair: true,
			Token0:             token0,
			Token1:             token1,
		})
	}
	return results, nil
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
	wethPair := common.HexToAddress("0x4000000000000000000000000000000000000004")
	usdtPair := common.HexToAddress("0x5000000000000000000000000000000000000005")
	producer := &producerFake{}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		56: {ChainID: 56, Status: appstore.ChainIngestStatusRunning},
	}}
	validator := &chainValidatorFake{defaultWeth: wethPair, defaultUsdt: usdtPair}
	code := []byte{0x60, 0x01, 0x60, 0x02}
	reader := &readerFake{
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
		codeByAddr: map[common.Address][]byte{contract: code},
	}
	ingestor, err := NewIngestor(Options{
		ChainID:           56,
		ConfirmationDepth: 2,
		StartBlock:        10,
		Reader:            reader,
		Producer:          producer,
		Store:             store,
		ChainValidator:    validator,
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
	if len(validator.contracts) != 1 || validator.contracts[0] != contract {
		t.Fatalf("validated contracts = %#v, want contract", validator.contracts)
	}
	if len(reader.codeRequests) != 1 || reader.codeRequests[0].contract != contract || reader.codeRequests[0].blockNumber != 10 {
		t.Fatalf("code requests = %#v, want contract at block 10", reader.codeRequests)
	}
	var payload appevents.ContractCreatedPayload
	if err := json.Unmarshal(producer.records[0].event.Payload, &payload); err != nil {
		t.Fatalf("decode contract payload: %v", err)
	}
	if wantHash := crypto.Keccak256Hash(code).Hex(); payload.CodeHash != wantHash {
		t.Fatalf("payload code hash = %s, want %s", payload.CodeHash, wantHash)
	}
	if payload.WethPair != wethPair.Hex() || payload.UsdtPair != usdtPair.Hex() {
		t.Fatalf("payload pairs = %s/%s, want %s/%s", payload.WethPair, payload.UsdtPair, wethPair.Hex(), usdtPair.Hex())
	}
	checkpoint := store.items[56]
	if checkpoint.CursorBlockNumber != 10 || checkpoint.FinalizedBlockNumber != 10 {
		t.Fatalf("checkpoint = %#v, want block 10", checkpoint)
	}
}

func TestIngestorSkipsContractCreationWhenBytecodeUnavailable(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	for _, tc := range []struct {
		name   string
		reader *readerFake
	}{
		{
			name:   "read error",
			reader: &readerFake{codeErr: errors.New("code unavailable")},
		},
		{
			name:   "empty code",
			reader: &readerFake{codeByAddr: map[common.Address][]byte{contract: nil}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			producer := &producerFake{}
			store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
				1: {ChainID: 1, Status: appstore.ChainIngestStatusRunning},
			}}
			tc.reader.latest = 6
			tc.reader.blocks = map[uint64]Block{
				5: {
					ChainID:           1,
					Number:            5,
					Hash:              common.HexToHash("0x5"),
					ContractCreations: []ContractCreated{{Contract: contract}},
				},
			}
			ingestor, err := NewIngestor(Options{
				ChainID:           1,
				ConfirmationDepth: 1,
				StartBlock:        5,
				Reader:            tc.reader,
				Producer:          producer,
				Store:             store,
				ChainValidator:    &chainValidatorFake{},
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
			if len(producer.records) != 0 {
				t.Fatalf("published records = %#v, want none", producer.records)
			}
			if checkpoint := store.items[1]; checkpoint.CursorBlockNumber != 5 || checkpoint.FinalizedBlockNumber != 5 {
				t.Fatalf("checkpoint = %#v, want block 5", checkpoint)
			}
		})
	}
}

func TestIngestorDoesNotAdvanceCheckpointWhenKafkaPublishFails(t *testing.T) {
	producer := &producerFake{failTopic: appevents.TopicContractCreatedV1}
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
		Producer:       producer,
		Store:          store,
		ChainValidator: &chainValidatorFake{},
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

func TestIngestorDoesNotAdvanceCheckpointWhenDexSwapPublishFails(t *testing.T) {
	producer := &producerFake{failTopic: appevents.TopicDexSwapV1}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		1: {ChainID: 1, Status: appstore.ChainIngestStatusRunning},
	}}
	ingestor, err := NewIngestor(Options{
		ChainID:           1,
		ConfirmationDepth: 1,
		StartBlock:        5,
		Reader: &readerFake{latest: 6, blocks: map[uint64]Block{
			5: {
				ChainID:  1,
				Number:   5,
				Hash:     common.HexToHash("0x5"),
				DexSwaps: []DexSwap{{Pair: common.HexToAddress("0x2000000000000000000000000000000000000002")}},
			},
		}},
		Producer:       producer,
		Store:          store,
		ChainValidator: &chainValidatorFake{},
	})
	if err != nil {
		t.Fatalf("NewIngestor: %v", err)
	}
	if _, err := ingestor.ProcessOnce(context.Background()); err == nil {
		t.Fatalf("ProcessOnce error = nil, want dex swap publish failure")
	}
	if checkpoint := store.items[1]; checkpoint.CursorBlockNumber != 0 || checkpoint.FinalizedBlockNumber != 0 {
		t.Fatalf("checkpoint = %#v, want no progress after dex publish failure", checkpoint)
	}
}

func TestIngestorSkipsInvalidERC20ContractsAndAdvancesCheckpoint(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	pair := common.HexToAddress("0x2000000000000000000000000000000000000002")
	producer := &producerFake{}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		1: {ChainID: 1, Status: appstore.ChainIngestStatusRunning},
	}}
	ingestor, err := NewIngestor(Options{
		ChainID:           1,
		ConfirmationDepth: 1,
		StartBlock:        5,
		Reader: &readerFake{latest: 6, blocks: map[uint64]Block{
			5: {
				ChainID:           1,
				Number:            5,
				Hash:              common.HexToHash("0x5"),
				ContractCreations: []ContractCreated{{Contract: contract}},
				DexSwaps:          []DexSwap{{Pair: pair}},
			},
		}},
		Producer: producer,
		Store:    store,
		ChainValidator: &chainValidatorFake{tokenResults: []model.TokenValidation{{
			IsValidERC20: false,
		}}},
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
	if len(producer.records) != 1 || producer.records[0].topic != appevents.TopicDexSwapV1 {
		t.Fatalf("published records = %#v, want only dex swap", producer.records)
	}
	if checkpoint := store.items[1]; checkpoint.CursorBlockNumber != 5 || checkpoint.FinalizedBlockNumber != 5 {
		t.Fatalf("checkpoint = %#v, want block 5", checkpoint)
	}
}

func TestIngestorSkipsContractCreationsWhenValidationFails(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	pair := common.HexToAddress("0x2000000000000000000000000000000000000002")
	producer := &producerFake{}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		1: {ChainID: 1, Status: appstore.ChainIngestStatusRunning},
	}}
	ingestor, err := NewIngestor(Options{
		ChainID:           1,
		ConfirmationDepth: 1,
		StartBlock:        5,
		Reader: &readerFake{latest: 6, blocks: map[uint64]Block{
			5: {
				ChainID:           1,
				Number:            5,
				Hash:              common.HexToHash("0x5"),
				ContractCreations: []ContractCreated{{Contract: contract}},
				DexSwaps:          []DexSwap{{Pair: pair}},
			},
		}},
		Producer:       producer,
		Store:          store,
		ChainValidator: &chainValidatorFake{tokenErr: errors.New("validate failed")},
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
	if len(producer.records) != 1 || producer.records[0].topic != appevents.TopicDexSwapV1 {
		t.Fatalf("published records = %#v, want only dex swap", producer.records)
	}
	if checkpoint := store.items[1]; checkpoint.CursorBlockNumber != 5 || checkpoint.FinalizedBlockNumber != 5 {
		t.Fatalf("checkpoint = %#v, want block 5", checkpoint)
	}
}

func TestIngestorSkipsContractCreationsWhenValidationLengthMismatches(t *testing.T) {
	contract := common.HexToAddress("0x1000000000000000000000000000000000000001")
	pair := common.HexToAddress("0x2000000000000000000000000000000000000002")
	producer := &producerFake{}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		1: {ChainID: 1, Status: appstore.ChainIngestStatusRunning},
	}}
	ingestor, err := NewIngestor(Options{
		ChainID:           1,
		ConfirmationDepth: 1,
		StartBlock:        5,
		Reader: &readerFake{latest: 6, blocks: map[uint64]Block{
			5: {
				ChainID:           1,
				Number:            5,
				Hash:              common.HexToHash("0x5"),
				ContractCreations: []ContractCreated{{Contract: contract}},
				DexSwaps:          []DexSwap{{Pair: pair}},
			},
		}},
		Producer:       producer,
		Store:          store,
		ChainValidator: &chainValidatorFake{tokenResults: []model.TokenValidation{}},
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
	if len(producer.records) != 1 || producer.records[0].topic != appevents.TopicDexSwapV1 {
		t.Fatalf("published records = %#v, want only dex swap", producer.records)
	}
	if checkpoint := store.items[1]; checkpoint.CursorBlockNumber != 5 || checkpoint.FinalizedBlockNumber != 5 {
		t.Fatalf("checkpoint = %#v, want block 5", checkpoint)
	}
}

func TestIngestorDeduplicatesDexSwapsByPairAndUsesValidatedTokens(t *testing.T) {
	pairA := common.HexToAddress("0x2000000000000000000000000000000000000002")
	pairB := common.HexToAddress("0x3000000000000000000000000000000000000003")
	tokenA0 := common.HexToAddress("0x4000000000000000000000000000000000000004")
	tokenA1 := common.HexToAddress("0x5000000000000000000000000000000000000005")
	tokenB0 := common.HexToAddress("0x6000000000000000000000000000000000000006")
	tokenB1 := common.HexToAddress("0x7000000000000000000000000000000000000007")
	txA := common.HexToHash("0xabc")
	txADuplicate := common.HexToHash("0xdef")
	txB := common.HexToHash("0x123")
	producer := &producerFake{}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		1: {ChainID: 1, Status: appstore.ChainIngestStatusRunning},
	}}
	validator := &chainValidatorFake{pairResults: []model.PairValidation{
		{IsValidPancakePair: true, Token0: tokenA0, Token1: tokenA1},
		{IsValidPancakePair: true, Token0: tokenB0, Token1: tokenB1},
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
				DexSwaps: []DexSwap{
					{Pair: pairA, TxHash: txA},
					{Pair: pairA, TxHash: txADuplicate},
					{Pair: pairB, TxHash: txB},
				},
			},
		}},
		Producer:       producer,
		Store:          store,
		ChainValidator: validator,
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
	if len(validator.pairs) != 2 || validator.pairs[0] != pairA || validator.pairs[1] != pairB {
		t.Fatalf("validated pairs = %#v, want pairA,pairB", validator.pairs)
	}
	if len(producer.records) != 2 {
		t.Fatalf("published records = %d, want two unique dex swaps", len(producer.records))
	}
	var first appevents.DexSwapPayload
	if err := json.Unmarshal(producer.records[0].event.Payload, &first); err != nil {
		t.Fatalf("decode first dex payload: %v", err)
	}
	if first.Pair != pairA.Hex() || first.Token0 != tokenA0.Hex() || first.Token1 != tokenA1.Hex() || first.TxHash != txA.Hex() {
		t.Fatalf("first dex payload = %#v, want pairA validated tokens and first tx hash", first)
	}
	var second appevents.DexSwapPayload
	if err := json.Unmarshal(producer.records[1].event.Payload, &second); err != nil {
		t.Fatalf("decode second dex payload: %v", err)
	}
	if second.Pair != pairB.Hex() || second.Token0 != tokenB0.Hex() || second.Token1 != tokenB1.Hex() || second.TxHash != txB.Hex() {
		t.Fatalf("second dex payload = %#v, want pairB validated tokens", second)
	}
	if checkpoint := store.items[1]; checkpoint.CursorBlockNumber != 5 || checkpoint.FinalizedBlockNumber != 5 {
		t.Fatalf("checkpoint = %#v, want block 5", checkpoint)
	}
}

func TestIngestorSkipsInvalidDexSwapPairsAndAdvancesCheckpoint(t *testing.T) {
	pair := common.HexToAddress("0x2000000000000000000000000000000000000002")
	producer := &producerFake{}
	store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
		1: {ChainID: 1, Status: appstore.ChainIngestStatusRunning},
	}}
	ingestor, err := NewIngestor(Options{
		ChainID:           1,
		ConfirmationDepth: 1,
		StartBlock:        5,
		Reader: &readerFake{latest: 6, blocks: map[uint64]Block{
			5: {
				ChainID:  1,
				Number:   5,
				Hash:     common.HexToHash("0x5"),
				DexSwaps: []DexSwap{{Pair: pair}},
			},
		}},
		Producer: producer,
		Store:    store,
		ChainValidator: &chainValidatorFake{pairResults: []model.PairValidation{{
			IsValidPancakePair: false,
		}}},
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
	if len(producer.records) != 0 {
		t.Fatalf("published records = %#v, want no dex swaps", producer.records)
	}
	if checkpoint := store.items[1]; checkpoint.CursorBlockNumber != 5 || checkpoint.FinalizedBlockNumber != 5 {
		t.Fatalf("checkpoint = %#v, want block 5", checkpoint)
	}
}

func TestIngestorSkipsDexSwapsWhenPairValidationFails(t *testing.T) {
	for _, tc := range []struct {
		name      string
		validator *chainValidatorFake
	}{
		{name: "error", validator: &chainValidatorFake{pairErr: errors.New("validate pairs failed")}},
		{name: "length mismatch", validator: &chainValidatorFake{pairResults: []model.PairValidation{}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pair := common.HexToAddress("0x2000000000000000000000000000000000000002")
			producer := &producerFake{}
			store := &checkpointStoreFake{items: map[int64]appstore.ChainIngestCheckpoint{
				1: {ChainID: 1, Status: appstore.ChainIngestStatusRunning},
			}}
			ingestor, err := NewIngestor(Options{
				ChainID:           1,
				ConfirmationDepth: 1,
				StartBlock:        5,
				Reader: &readerFake{latest: 6, blocks: map[uint64]Block{
					5: {
						ChainID:  1,
						Number:   5,
						Hash:     common.HexToHash("0x5"),
						DexSwaps: []DexSwap{{Pair: pair}},
					},
				}},
				Producer:       producer,
				Store:          store,
				ChainValidator: tc.validator,
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
			if len(producer.records) != 0 {
				t.Fatalf("published records = %#v, want no dex swaps", producer.records)
			}
			if checkpoint := store.items[1]; checkpoint.CursorBlockNumber != 5 || checkpoint.FinalizedBlockNumber != 5 {
				t.Fatalf("checkpoint = %#v, want block 5", checkpoint)
			}
		})
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
			ChainValidator:    &chainValidatorFake{},
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
		ChainValidator:    &chainValidatorFake{},
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
		Producer:       &producerFake{},
		Store:          store,
		ChainValidator: &chainValidatorFake{},
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
