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

type ChainValidator interface {
	ValidateERC20(ctx context.Context, contracts []common.Address) ([]model.TokenValidation, error)
	ValidatePairs(ctx context.Context, pairs []common.Address) ([]model.PairValidation, error)
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
	TxHash common.Hash
}

type Options struct {
	ChainID           int64
	ConfirmationDepth uint64
	StartBlock        uint64
	Reader            Reader
	Producer          Producer
	Store             CheckpointStore
	ChainValidator    ChainValidator
}

type Ingestor struct {
	chainID           int64
	confirmationDepth uint64
	startBlock        uint64
	reader            Reader
	producer          Producer
	store             CheckpointStore
	chainValidator    ChainValidator
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
	if opts.ChainValidator == nil {
		return nil, errors.New("application chain ingestor chain validator is required")
	}
	return &Ingestor{
		chainID:           opts.ChainID,
		confirmationDepth: opts.ConfirmationDepth,
		startBlock:        opts.StartBlock,
		reader:            opts.Reader,
		producer:          opts.Producer,
		store:             opts.Store,
		chainValidator:    opts.ChainValidator,
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
	if validatedCreations, ok := i.validatedContractCreations(ctx, block.ContractCreations); ok {
		for _, item := range validatedCreations {
			txIndex, err := int64FromUint64(item.creation.TxIndex, "tx index")
			if err != nil {
				return err
			}
			envelope, err := appevents.NewEnvelope(appevents.EventTypeContractCreated, i.chainID, appevents.ContractCreatedPayload{
				Contract:    item.creation.Contract.Hex(),
				Creator:     item.creation.Creator.Hex(),
				TxHash:      item.creation.TxHash.Hex(),
				WethPair:    item.validation.WethPair.Hex(),
				UsdtPair:    item.validation.UsdtPair.Hex(),
				BlockNumber: blockNumber,
				BlockTime:   blockTime,
				TxIndex:     txIndex,
			})
			if err != nil {
				return err
			}
			if err := i.producer.Publish(ctx, appevents.TopicContractCreatedV1, appevents.ContractCreatedKey(i.chainID, item.creation.Contract.Hex()), envelope); err != nil {
				return err
			}
		}
	}
	if validatedSwaps, ok := i.validatedDexSwaps(ctx, block.DexSwaps); ok {
		for _, item := range validatedSwaps {
			envelope, err := appevents.NewEnvelope(appevents.EventTypeDexSwap, i.chainID, appevents.DexSwapPayload{
				Pair:        item.swap.Pair.Hex(),
				Token0:      item.validation.Token0.Hex(),
				Token1:      item.validation.Token1.Hex(),
				TxHash:      item.swap.TxHash.Hex(),
				BlockNumber: blockNumber,
			})
			if err != nil {
				return err
			}
			if err := i.producer.Publish(ctx, appevents.TopicDexSwapV1, appevents.DexSwapKey(i.chainID, item.swap.Pair.Hex()), envelope); err != nil {
				return err
			}
		}
	}
	return nil
}

type validatedContractCreation struct {
	creation   ContractCreated
	validation model.TokenValidation
}

func (i *Ingestor) validatedContractCreations(ctx context.Context, items []ContractCreated) ([]validatedContractCreation, bool) {
	if len(items) == 0 {
		return nil, true
	}
	creations := make([]ContractCreated, 0, len(items))
	contracts := make([]common.Address, 0, len(items))
	for _, item := range items {
		if item.Contract == (common.Address{}) {
			continue
		}
		creations = append(creations, item)
		contracts = append(contracts, item.Contract)
	}
	if len(contracts) == 0 {
		return nil, true
	}
	validations, err := i.chainValidator.ValidateERC20(ctx, contracts)
	if err != nil || len(validations) != len(creations) {
		return nil, false
	}
	result := make([]validatedContractCreation, 0, len(creations))
	for idx, validation := range validations {
		if !validation.IsValidERC20 {
			continue
		}
		result = append(result, validatedContractCreation{
			creation:   creations[idx],
			validation: validation,
		})
	}
	return result, true
}

type validatedDexSwap struct {
	swap       DexSwap
	validation model.PairValidation
}

func (i *Ingestor) validatedDexSwaps(ctx context.Context, items []DexSwap) ([]validatedDexSwap, bool) {
	if len(items) == 0 {
		return nil, true
	}
	swaps := make([]DexSwap, 0, len(items))
	pairs := make([]common.Address, 0, len(items))
	seen := make(map[common.Address]struct{}, len(items))
	for _, item := range items {
		if item.Pair == (common.Address{}) {
			continue
		}
		if _, ok := seen[item.Pair]; ok {
			continue
		}
		seen[item.Pair] = struct{}{}
		swaps = append(swaps, item)
		pairs = append(pairs, item.Pair)
	}
	if len(pairs) == 0 {
		return nil, true
	}
	validations, err := i.chainValidator.ValidatePairs(ctx, pairs)
	if err != nil || len(validations) != len(swaps) {
		return nil, false
	}
	result := make([]validatedDexSwap, 0, len(swaps))
	for idx, validation := range validations {
		if !validation.IsValidPancakePair {
			continue
		}
		result = append(result, validatedDexSwap{
			swap:       swaps[idx],
			validation: validation,
		})
	}
	return result, true
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
