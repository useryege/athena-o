package ingest

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	appevents "github.com/useryege/athena/internal/application/events"
	"github.com/useryege/athena/internal/application/model"
	appstore "github.com/useryege/athena/internal/application/store"
)

type Reader interface {
	LatestBlockNumber(ctx context.Context) (uint64, error)
	ReadBlock(ctx context.Context, number uint64) (Block, error)
}

type Producer interface {
	Publish(ctx context.Context, topic string, key string, envelope appevents.Envelope) error
}

type CheckpointStore interface {
	GetChainIngestCheckpoint(ctx context.Context, chainID int64) (*appstore.ChainIngestCheckpoint, error)
	UpsertChainIngestCheckpoint(ctx context.Context, item appstore.ChainIngestCheckpoint) (*appstore.ChainIngestCheckpoint, error)
}

type Block struct {
	ChainID           int64
	Number            uint64
	Hash              common.Hash
	Time              uint64
	ContractCreations []ContractCreated
	DexSwaps          []DexSwap
}

type ContractCreated struct {
	Contract common.Address
	Creator  common.Address
	TxHash   common.Hash
	TxIndex  uint64
}

type DexSwap struct {
	Pair   common.Address
	Token0 common.Address
	Token1 common.Address
	TxHash common.Hash
}

type Options struct {
	ChainID           int64
	ConfirmationDepth uint64
	StartBlock        uint64
	Reader            Reader
	Producer          Producer
	Store             CheckpointStore
}

type Ingestor struct {
	chainID           int64
	confirmationDepth uint64
	startBlock        uint64
	reader            Reader
	producer          Producer
	store             CheckpointStore
}

func NewIngestor(opts Options) (*Ingestor, error) {
	if opts.ChainID <= 0 {
		return nil, errors.New("application chain ingestor chain_id must be positive")
	}
	if opts.ConfirmationDepth == 0 {
		opts.ConfirmationDepth = DefaultConfirmationDepth(opts.ChainID)
	}
	if opts.StartBlock == 0 {
		opts.StartBlock = 1
	}
	if opts.Reader == nil {
		return nil, errors.New("application chain ingestor reader is required")
	}
	if opts.Producer == nil {
		return nil, errors.New("application chain ingestor kafka producer is required")
	}
	if opts.Store == nil {
		return nil, errors.New("application chain ingestor checkpoint store is required")
	}
	return &Ingestor{
		chainID:           opts.ChainID,
		confirmationDepth: opts.ConfirmationDepth,
		startBlock:        opts.StartBlock,
		reader:            opts.Reader,
		producer:          opts.Producer,
		store:             opts.Store,
	}, nil
}

func DefaultConfirmationDepth(chainID int64) uint64 {
	switch chainID {
	case model.ChainIDEthereumMainnet:
		return model.EthereumMainnetConfirmationDepth
	case model.ChainIDBSCMainnet:
		return model.BSCMainnetConfirmationDepth
	default:
		return 6
	}
}

func (i *Ingestor) ProcessOnce(ctx context.Context) (int, error) {
	if i == nil {
		return 0, nil
	}
	checkpoint, err := i.store.GetChainIngestCheckpoint(ctx, i.chainID)
	if err != nil {
		return 0, err
	}
	if !checkpointIsRunning(checkpoint) {
		return 0, nil
	}
	latest, err := i.reader.LatestBlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	if latest <= i.confirmationDepth {
		return 0, nil
	}
	finalized := latest - i.confirmationDepth
	next := i.startBlock
	if checkpoint != nil && checkpoint.CursorBlockNumber >= next {
		next = checkpoint.CursorBlockNumber + 1
	}
	if next > finalized {
		return 0, nil
	}
	processed := 0
	for number := next; number <= finalized; number++ {
		if err := ctx.Err(); err != nil {
			return processed, err
		}
		checkpoint, err := i.store.GetChainIngestCheckpoint(ctx, i.chainID)
		if err != nil {
			return processed, err
		}
		if !checkpointIsRunning(checkpoint) {
			return processed, nil
		}
		block, err := i.reader.ReadBlock(ctx, number)
		if err != nil {
			return processed, err
		}
		if block.ChainID == 0 {
			block.ChainID = i.chainID
		}
		if block.ChainID != i.chainID {
			return processed, fmt.Errorf("reader returned chain_id %d for ingestor chain_id %d", block.ChainID, i.chainID)
		}
		if err := i.publishBlock(ctx, block); err != nil {
			return processed, err
		}
		checkpoint, err = i.store.GetChainIngestCheckpoint(ctx, i.chainID)
		if err != nil {
			return processed, err
		}
		if !checkpointIsRunning(checkpoint) {
			return processed, nil
		}
		if _, err := i.store.UpsertChainIngestCheckpoint(ctx, appstore.ChainIngestCheckpoint{
			ChainID:              i.chainID,
			FinalizedBlockNumber: block.Number,
			FinalizedBlockHash:   block.Hash,
			CursorBlockNumber:    block.Number,
			CursorBlockHash:      block.Hash,
			Status:               appstore.ChainIngestStatusRunning,
		}); err != nil {
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func checkpointIsRunning(checkpoint *appstore.ChainIngestCheckpoint) bool {
	return checkpoint != nil && checkpoint.Status == appstore.ChainIngestStatusRunning
}

func (i *Ingestor) publishBlock(ctx context.Context, block Block) error {
	blockNumber, err := int64FromUint64(block.Number, "block number")
	if err != nil {
		return err
	}
	blockTime, err := int64FromUint64(block.Time, "block time")
	if err != nil {
		return err
	}
	for _, item := range block.ContractCreations {
		if item.Contract == (common.Address{}) {
			continue
		}
		txIndex, err := int64FromUint64(item.TxIndex, "tx index")
		if err != nil {
			return err
		}
		envelope, err := appevents.NewEnvelope(appevents.EventTypeContractCreated, i.chainID, appevents.ContractCreatedPayload{
			Contract:    item.Contract.Hex(),
			Creator:     item.Creator.Hex(),
			TxHash:      item.TxHash.Hex(),
			BlockNumber: blockNumber,
			BlockTime:   blockTime,
			TxIndex:     txIndex,
		})
		if err != nil {
			return err
		}
		if err := i.producer.Publish(ctx, appevents.TopicContractCreatedV1, appevents.ContractCreatedKey(i.chainID, item.Contract.Hex()), envelope); err != nil {
			return err
		}
	}
	for _, item := range block.DexSwaps {
		if item.Pair == (common.Address{}) {
			continue
		}
		envelope, err := appevents.NewEnvelope(appevents.EventTypeDexSwap, i.chainID, appevents.DexSwapPayload{
			Pair:        item.Pair.Hex(),
			Token0:      item.Token0.Hex(),
			Token1:      item.Token1.Hex(),
			TxHash:      item.TxHash.Hex(),
			BlockNumber: blockNumber,
		})
		if err != nil {
			return err
		}
		if err := i.producer.Publish(ctx, appevents.TopicDexSwapV1, appevents.DexSwapKey(i.chainID, item.Pair.Hex()), envelope); err != nil {
			return err
		}
	}
	return nil
}

func int64FromUint64(value uint64, name string) (int64, error) {
	const maxInt64 = uint64(1<<63 - 1)
	if value > maxInt64 {
		return 0, fmt.Errorf("%s exceeds int64", name)
	}
	return int64(value), nil
}

type LoopOptions struct {
	PollInterval time.Duration
}

func (i *Ingestor) Run(ctx context.Context, opts LoopOptions) error {
	if opts.PollInterval <= 0 {
		opts.PollInterval = 3 * time.Second
	}
	ticker := time.NewTicker(opts.PollInterval)
	defer ticker.Stop()
	for {
		if _, err := i.ProcessOnce(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
