package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	bscswapsqlc "github.com/useryege/athena/internal/bscswap/store/sqlc"
	postgresutil "github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

var ErrCheckpointConflict = errors.New("BSC swap scan checkpoint changed concurrently")

type Checkpoint struct {
	StartBlockNumber     uint64
	CursorBlockNumber    uint64
	CursorBlockHash      common.Hash
	CursorBlockTimestamp uint64
	InitializedAt        time.Time
	UpdatedAt            time.Time
}

type SwapTransaction struct {
	TransactionHash  common.Hash
	BlockNumber      uint64
	BlockHash        common.Hash
	BlockTimestamp   uint64
	TransactionIndex uint64
	FromAddress      common.Address
}

type PageCursor struct {
	BlockNumber      uint64
	TransactionIndex uint64
}

type Store struct {
	pool    *pgxpool.Pool
	queries *bscswapsqlc.Queries
}

func Open(ctx context.Context) (*Store, error) {
	pool, err := postgresutil.ConnectAndMigrate(ctx, postgresutil.Options{
		Module:       "bsc swap",
		DSNEnv:       "ATHENA_BSC_SWAP_POSTGRES_DSN",
		Database:     "bsc_swap",
		Migrations:   migrations,
		MigrationDir: "migrations",
	})
	if err != nil {
		return nil, err
	}
	log.Info("BSC swap postgres migrations are up to date")
	return &Store{pool: pool, queries: bscswapsqlc.New(pool)}, nil
}

func (s *Store) Close() error {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("BSC swap postgres database is not configured")
	}
	return s.pool.Ping(ctx)
}

func (s *Store) GetCheckpoint(ctx context.Context) (Checkpoint, error) {
	if s == nil || s.queries == nil {
		return Checkpoint{}, fmt.Errorf("BSC swap postgres database is not configured")
	}
	row, err := s.queries.GetScanCheckpoint(ctx)
	if err != nil {
		return Checkpoint{}, err
	}
	return checkpointFromRow(row)
}

func (s *Store) InitializeCheckpoint(ctx context.Context, checkpoint Checkpoint) (Checkpoint, error) {
	if s == nil || s.queries == nil {
		return Checkpoint{}, fmt.Errorf("BSC swap postgres database is not configured")
	}
	params, err := checkpointParams(checkpoint)
	if err != nil {
		return Checkpoint{}, err
	}
	row, err := s.queries.InitializeScanCheckpoint(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.GetCheckpoint(ctx)
	}
	if err != nil {
		return Checkpoint{}, fmt.Errorf("initialize BSC swap scan checkpoint: %w", err)
	}
	return checkpointFromRow(row)
}

func (s *Store) CommitBatch(ctx context.Context, expectedCursor uint64, transactions []SwapTransaction, checkpoint Checkpoint) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("BSC swap postgres database is not configured")
	}
	if checkpoint.CursorBlockNumber <= expectedCursor {
		return fmt.Errorf("checkpoint cursor %d must advance past %d", checkpoint.CursorBlockNumber, expectedCursor)
	}
	expected, err := uint64ToInt64("expected_cursor_block_number", expectedCursor)
	if err != nil {
		return err
	}
	checkpointBlock, err := uint64ToInt64("cursor_block_number", checkpoint.CursorBlockNumber)
	if err != nil {
		return err
	}
	checkpointTimestamp, err := uint64ToInt64("cursor_block_timestamp", checkpoint.CursorBlockTimestamp)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin BSC swap batch transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := bscswapsqlc.New(tx)
	for _, transaction := range transactions {
		params, err := transactionParams(transaction)
		if err != nil {
			return err
		}
		if err := queries.InsertSwapTransaction(ctx, params); err != nil {
			return fmt.Errorf("insert BSC swap transaction %s: %w", transaction.TransactionHash.Hex(), err)
		}
	}
	rows, err := queries.AdvanceScanCheckpoint(ctx, bscswapsqlc.AdvanceScanCheckpointParams{
		CursorBlockNumber:         checkpointBlock,
		CursorBlockHash:           checkpoint.CursorBlockHash.Bytes(),
		CursorBlockTimestamp:      checkpointTimestamp,
		ExpectedCursorBlockNumber: expected,
	})
	if err != nil {
		return fmt.Errorf("advance BSC swap scan checkpoint: %w", err)
	}
	if rows != 1 {
		return ErrCheckpointConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit BSC swap scan batch: %w", err)
	}
	return nil
}

func (s *Store) ListSwapTransactions(ctx context.Context, address common.Address, beforeBlock uint64, cursor *PageCursor, limit int32) ([]SwapTransaction, Checkpoint, error) {
	if s == nil || s.pool == nil {
		return nil, Checkpoint{}, fmt.Errorf("BSC swap postgres database is not configured")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, Checkpoint{}, fmt.Errorf("begin BSC swap query transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := bscswapsqlc.New(tx)
	checkpointRow, err := queries.GetScanCheckpoint(ctx)
	if err != nil {
		return nil, Checkpoint{}, err
	}
	checkpoint, err := checkpointFromRow(checkpointRow)
	if err != nil {
		return nil, Checkpoint{}, err
	}

	var items []SwapTransaction
	if cursor == nil {
		blockNumber, conversionErr := uint64ToInt64("before_block_number", beforeBlock)
		if conversionErr != nil {
			return nil, Checkpoint{}, conversionErr
		}
		rows, listErr := queries.ListSwapTransactionsBeforeBlock(ctx, bscswapsqlc.ListSwapTransactionsBeforeBlockParams{
			FromAddress: address.Bytes(), BeforeBlockNumber: blockNumber, ResultLimit: limit,
		})
		if listErr != nil {
			return nil, Checkpoint{}, fmt.Errorf("list BSC swap transactions before block: %w", listErr)
		}
		items, err = transactionsBeforeBlockFromRows(rows)
	} else {
		blockNumber, conversionErr := uint64ToInt64("before_block_number", cursor.BlockNumber)
		if conversionErr != nil {
			return nil, Checkpoint{}, conversionErr
		}
		transactionIndex, conversionErr := uint64ToInt64("before_transaction_index", cursor.TransactionIndex)
		if conversionErr != nil {
			return nil, Checkpoint{}, conversionErr
		}
		rows, listErr := queries.ListSwapTransactionsBeforePosition(ctx, bscswapsqlc.ListSwapTransactionsBeforePositionParams{
			FromAddress: address.Bytes(), BeforeBlockNumber: blockNumber,
			BeforeTransactionIndex: transactionIndex, ResultLimit: limit,
		})
		if listErr != nil {
			return nil, Checkpoint{}, fmt.Errorf("list BSC swap transactions before position: %w", listErr)
		}
		items, err = transactionsBeforePositionFromRows(rows)
	}
	if err != nil {
		return nil, Checkpoint{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, Checkpoint{}, fmt.Errorf("commit BSC swap query transaction: %w", err)
	}
	return items, checkpoint, nil
}

func checkpointParams(checkpoint Checkpoint) (bscswapsqlc.InitializeScanCheckpointParams, error) {
	start, err := uint64ToInt64("start_block_number", checkpoint.StartBlockNumber)
	if err != nil {
		return bscswapsqlc.InitializeScanCheckpointParams{}, err
	}
	cursor, err := uint64ToInt64("cursor_block_number", checkpoint.CursorBlockNumber)
	if err != nil {
		return bscswapsqlc.InitializeScanCheckpointParams{}, err
	}
	timestamp, err := uint64ToInt64("cursor_block_timestamp", checkpoint.CursorBlockTimestamp)
	if err != nil {
		return bscswapsqlc.InitializeScanCheckpointParams{}, err
	}
	return bscswapsqlc.InitializeScanCheckpointParams{
		StartBlockNumber: start, CursorBlockNumber: cursor,
		CursorBlockHash: checkpoint.CursorBlockHash.Bytes(), CursorBlockTimestamp: timestamp,
	}, nil
}

func checkpointFromRow(row bscswapsqlc.BscSwapScanCheckpoint) (Checkpoint, error) {
	start, err := int64ToUint64("start_block_number", row.StartBlockNumber)
	if err != nil {
		return Checkpoint{}, err
	}
	cursor, err := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if err != nil {
		return Checkpoint{}, err
	}
	timestamp, err := int64ToUint64("cursor_block_timestamp", row.CursorBlockTimestamp)
	if err != nil {
		return Checkpoint{}, err
	}
	if len(row.CursorBlockHash) != common.HashLength {
		return Checkpoint{}, fmt.Errorf("cursor block hash must contain %d bytes", common.HashLength)
	}
	return Checkpoint{
		StartBlockNumber: start, CursorBlockNumber: cursor,
		CursorBlockHash: common.BytesToHash(row.CursorBlockHash), CursorBlockTimestamp: timestamp,
		InitializedAt: timestampValue(row.InitializedAt), UpdatedAt: timestampValue(row.UpdatedAt),
	}, nil
}

func transactionParams(transaction SwapTransaction) (bscswapsqlc.InsertSwapTransactionParams, error) {
	blockNumber, err := uint64ToInt64("block_number", transaction.BlockNumber)
	if err != nil {
		return bscswapsqlc.InsertSwapTransactionParams{}, err
	}
	blockTimestamp, err := uint64ToInt64("block_timestamp", transaction.BlockTimestamp)
	if err != nil {
		return bscswapsqlc.InsertSwapTransactionParams{}, err
	}
	transactionIndex, err := uint64ToInt64("transaction_index", transaction.TransactionIndex)
	if err != nil {
		return bscswapsqlc.InsertSwapTransactionParams{}, err
	}
	return bscswapsqlc.InsertSwapTransactionParams{
		TransactionHash: transaction.TransactionHash.Bytes(), BlockNumber: blockNumber,
		BlockHash: transaction.BlockHash.Bytes(), BlockTimestamp: blockTimestamp,
		TransactionIndex: transactionIndex, FromAddress: transaction.FromAddress.Bytes(),
	}, nil
}

func transactionsBeforeBlockFromRows(rows []bscswapsqlc.ListSwapTransactionsBeforeBlockRow) ([]SwapTransaction, error) {
	items := make([]SwapTransaction, 0, len(rows))
	for _, row := range rows {
		item, err := transactionFromValues(row.TransactionHash, row.BlockNumber, row.BlockHash, row.BlockTimestamp, row.TransactionIndex, row.FromAddress)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func transactionsBeforePositionFromRows(rows []bscswapsqlc.ListSwapTransactionsBeforePositionRow) ([]SwapTransaction, error) {
	items := make([]SwapTransaction, 0, len(rows))
	for _, row := range rows {
		item, err := transactionFromValues(row.TransactionHash, row.BlockNumber, row.BlockHash, row.BlockTimestamp, row.TransactionIndex, row.FromAddress)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func transactionFromValues(transactionHash []byte, blockNumber int64, blockHash []byte, blockTimestamp, transactionIndex int64, fromAddress []byte) (SwapTransaction, error) {
	if len(transactionHash) != common.HashLength || len(blockHash) != common.HashLength {
		return SwapTransaction{}, fmt.Errorf("stored BSC swap transaction contains an invalid hash")
	}
	if len(fromAddress) != common.AddressLength {
		return SwapTransaction{}, fmt.Errorf("stored BSC swap transaction contains an invalid address")
	}
	bn, err := int64ToUint64("block_number", blockNumber)
	if err != nil {
		return SwapTransaction{}, err
	}
	bt, err := int64ToUint64("block_timestamp", blockTimestamp)
	if err != nil {
		return SwapTransaction{}, err
	}
	ti, err := int64ToUint64("transaction_index", transactionIndex)
	if err != nil {
		return SwapTransaction{}, err
	}
	return SwapTransaction{
		TransactionHash: common.BytesToHash(transactionHash), BlockNumber: bn,
		BlockHash: common.BytesToHash(blockHash), BlockTimestamp: bt,
		TransactionIndex: ti, FromAddress: common.BytesToAddress(fromAddress),
	}, nil
}

func timestampValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func uint64ToInt64(field string, value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("%s exceeds int64 max", field)
	}
	return int64(value), nil
}

func int64ToUint64(field string, value int64) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("%s is negative", field)
	}
	return uint64(value), nil
}
