package store

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	bscsqlc "github.com/useryege/athena/internal/bscinbound/store/sqlc"
	postgresutil "github.com/useryege/athena/util/db/postgres"
)

//go:embed migrations/*.sql
var migrations embed.FS

var ErrCheckpointConflict = errors.New("BSC inbound scan checkpoint changed concurrently")

type Checkpoint struct {
	StartBlockNumber     uint64
	CursorBlockNumber    uint64
	CursorBlockHash      common.Hash
	CursorBlockTimestamp uint64
	InitializedAt        time.Time
	UpdatedAt            time.Time
}

type InboundNormalTransaction struct {
	TransactionHash  common.Hash
	BlockNumber      uint64
	BlockHash        common.Hash
	BlockTimestamp   uint64
	TransactionIndex uint64
	FromAddress      common.Address
	ToAddress        common.Address
	ValueWei         *big.Int
}

type PageCursor struct {
	BlockNumber      uint64
	TransactionIndex uint64
}

type Store struct {
	pool    *pgxpool.Pool
	queries *bscsqlc.Queries
}

func Open(ctx context.Context) (*Store, error) {
	pool, err := postgresutil.ConnectAndMigrate(ctx, postgresutil.Options{
		Module:       "bsc inbound",
		DSNEnv:       "ATHENA_BSC_INBOUND_POSTGRES_DSN",
		Database:     "bsc_inbound",
		Migrations:   migrations,
		MigrationDir: "migrations",
	})
	if err != nil {
		return nil, err
	}
	log.Info("BSC inbound postgres migrations are up to date")
	return &Store{pool: pool, queries: bscsqlc.New(pool)}, nil
}

func (s *Store) Close() error {
	if s != nil && s.pool != nil {
		s.pool.Close()
	}
	return nil
}

func (s *Store) Ping(ctx context.Context) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("BSC inbound postgres database is not configured")
	}
	return s.pool.Ping(ctx)
}

func (s *Store) GetCheckpoint(ctx context.Context) (Checkpoint, error) {
	if s == nil || s.queries == nil {
		return Checkpoint{}, fmt.Errorf("BSC inbound postgres database is not configured")
	}
	row, err := s.queries.GetScanCheckpoint(ctx)
	if err != nil {
		return Checkpoint{}, err
	}
	return checkpointFromRow(row)
}

func (s *Store) InitializeCheckpoint(ctx context.Context, checkpoint Checkpoint) (Checkpoint, error) {
	if s == nil || s.queries == nil {
		return Checkpoint{}, fmt.Errorf("BSC inbound postgres database is not configured")
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
		return Checkpoint{}, fmt.Errorf("initialize BSC inbound scan checkpoint: %w", err)
	}
	return checkpointFromRow(row)
}

func (s *Store) CommitBatch(ctx context.Context, expectedCursor uint64, transactions []InboundNormalTransaction, checkpoint Checkpoint) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("BSC inbound postgres database is not configured")
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
		return fmt.Errorf("begin BSC inbound batch transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := bscsqlc.New(tx)
	for _, transaction := range transactions {
		params, err := transactionParams(transaction)
		if err != nil {
			return err
		}
		if err := queries.InsertInboundNormalTransaction(ctx, params); err != nil {
			return fmt.Errorf("insert inbound transaction %s: %w", transaction.TransactionHash.Hex(), err)
		}
	}
	rows, err := queries.AdvanceScanCheckpoint(ctx, bscsqlc.AdvanceScanCheckpointParams{
		CursorBlockNumber:         checkpointBlock,
		CursorBlockHash:           checkpoint.CursorBlockHash.Bytes(),
		CursorBlockTimestamp:      checkpointTimestamp,
		ExpectedCursorBlockNumber: expected,
	})
	if err != nil {
		return fmt.Errorf("advance BSC inbound scan checkpoint: %w", err)
	}
	if rows != 1 {
		return ErrCheckpointConflict
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit BSC inbound scan batch: %w", err)
	}
	return nil
}

func (s *Store) ListInboundNormalTransactions(ctx context.Context, address common.Address, cursor *PageCursor, limit int32) ([]InboundNormalTransaction, Checkpoint, error) {
	if s == nil || s.pool == nil {
		return nil, Checkpoint{}, fmt.Errorf("BSC inbound postgres database is not configured")
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, Checkpoint{}, fmt.Errorf("begin BSC inbound query transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := bscsqlc.New(tx)
	checkpointRow, err := queries.GetScanCheckpoint(ctx)
	if err != nil {
		return nil, Checkpoint{}, err
	}
	checkpoint, err := checkpointFromRow(checkpointRow)
	if err != nil {
		return nil, Checkpoint{}, err
	}
	var (
		items    []InboundNormalTransaction
		queryErr error
	)
	if cursor == nil {
		rows, listErr := queries.ListLatestInboundNormalTransactions(ctx, bscsqlc.ListLatestInboundNormalTransactionsParams{
			ToAddress: address.Bytes(), ResultLimit: limit,
		})
		if listErr != nil {
			queryErr = listErr
		} else {
			items, queryErr = latestTransactionsFromRows(rows)
		}
	} else {
		blockNumber, conversionErr := uint64ToInt64("before_block_number", cursor.BlockNumber)
		if conversionErr != nil {
			return nil, Checkpoint{}, conversionErr
		}
		transactionIndex, conversionErr := uint64ToInt64("before_transaction_index", cursor.TransactionIndex)
		if conversionErr != nil {
			return nil, Checkpoint{}, conversionErr
		}
		rows, listErr := queries.ListInboundNormalTransactionsBefore(ctx, bscsqlc.ListInboundNormalTransactionsBeforeParams{
			ToAddress: address.Bytes(), BeforeBlockNumber: blockNumber,
			BeforeTransactionIndex: transactionIndex, ResultLimit: limit,
		})
		if listErr != nil {
			queryErr = listErr
		} else {
			items, queryErr = transactionsBeforeFromRows(rows)
		}
	}
	if queryErr != nil {
		return nil, Checkpoint{}, fmt.Errorf("list BSC inbound normal transactions: %w", queryErr)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, Checkpoint{}, fmt.Errorf("commit BSC inbound query transaction: %w", err)
	}
	return items, checkpoint, nil
}

func checkpointParams(checkpoint Checkpoint) (bscsqlc.InitializeScanCheckpointParams, error) {
	start, err := uint64ToInt64("start_block_number", checkpoint.StartBlockNumber)
	if err != nil {
		return bscsqlc.InitializeScanCheckpointParams{}, err
	}
	cursor, err := uint64ToInt64("cursor_block_number", checkpoint.CursorBlockNumber)
	if err != nil {
		return bscsqlc.InitializeScanCheckpointParams{}, err
	}
	timestamp, err := uint64ToInt64("cursor_block_timestamp", checkpoint.CursorBlockTimestamp)
	if err != nil {
		return bscsqlc.InitializeScanCheckpointParams{}, err
	}
	return bscsqlc.InitializeScanCheckpointParams{
		StartBlockNumber: start, CursorBlockNumber: cursor,
		CursorBlockHash: checkpoint.CursorBlockHash.Bytes(), CursorBlockTimestamp: timestamp,
	}, nil
}

func checkpointFromRow(row bscsqlc.BscInboundScanCheckpoint) (Checkpoint, error) {
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

func transactionParams(transaction InboundNormalTransaction) (bscsqlc.InsertInboundNormalTransactionParams, error) {
	blockNumber, err := uint64ToInt64("block_number", transaction.BlockNumber)
	if err != nil {
		return bscsqlc.InsertInboundNormalTransactionParams{}, err
	}
	blockTimestamp, err := uint64ToInt64("block_timestamp", transaction.BlockTimestamp)
	if err != nil {
		return bscsqlc.InsertInboundNormalTransactionParams{}, err
	}
	transactionIndex, err := uint64ToInt64("transaction_index", transaction.TransactionIndex)
	if err != nil {
		return bscsqlc.InsertInboundNormalTransactionParams{}, err
	}
	if transaction.ValueWei == nil || transaction.ValueWei.Sign() <= 0 {
		return bscsqlc.InsertInboundNormalTransactionParams{}, fmt.Errorf("transaction %s value must be positive", transaction.TransactionHash.Hex())
	}
	return bscsqlc.InsertInboundNormalTransactionParams{
		TransactionHash: transaction.TransactionHash.Bytes(), BlockNumber: blockNumber,
		BlockHash: transaction.BlockHash.Bytes(), BlockTimestamp: blockTimestamp,
		TransactionIndex: transactionIndex, FromAddress: transaction.FromAddress.Bytes(),
		ToAddress: transaction.ToAddress.Bytes(), ValueWei: numericFromBigInt(transaction.ValueWei),
	}, nil
}

func latestTransactionsFromRows(rows []bscsqlc.ListLatestInboundNormalTransactionsRow) ([]InboundNormalTransaction, error) {
	items := make([]InboundNormalTransaction, 0, len(rows))
	for _, row := range rows {
		item, err := transactionFromValues(row.TransactionHash, row.BlockNumber, row.BlockHash, row.BlockTimestamp, row.TransactionIndex, row.FromAddress, row.ToAddress, row.ValueWei)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func transactionsBeforeFromRows(rows []bscsqlc.ListInboundNormalTransactionsBeforeRow) ([]InboundNormalTransaction, error) {
	items := make([]InboundNormalTransaction, 0, len(rows))
	for _, row := range rows {
		item, err := transactionFromValues(row.TransactionHash, row.BlockNumber, row.BlockHash, row.BlockTimestamp, row.TransactionIndex, row.FromAddress, row.ToAddress, row.ValueWei)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func transactionFromValues(transactionHash []byte, blockNumber int64, blockHash []byte, blockTimestamp, transactionIndex int64, fromAddress, toAddress []byte, value pgtype.Numeric) (InboundNormalTransaction, error) {
	if len(transactionHash) != common.HashLength || len(blockHash) != common.HashLength {
		return InboundNormalTransaction{}, fmt.Errorf("stored transaction contains an invalid hash")
	}
	if len(fromAddress) != common.AddressLength || len(toAddress) != common.AddressLength {
		return InboundNormalTransaction{}, fmt.Errorf("stored transaction contains an invalid address")
	}
	bn, err := int64ToUint64("block_number", blockNumber)
	if err != nil {
		return InboundNormalTransaction{}, err
	}
	bt, err := int64ToUint64("block_timestamp", blockTimestamp)
	if err != nil {
		return InboundNormalTransaction{}, err
	}
	ti, err := int64ToUint64("transaction_index", transactionIndex)
	if err != nil {
		return InboundNormalTransaction{}, err
	}
	return InboundNormalTransaction{
		TransactionHash: common.BytesToHash(transactionHash), BlockNumber: bn,
		BlockHash: common.BytesToHash(blockHash), BlockTimestamp: bt, TransactionIndex: ti,
		FromAddress: common.BytesToAddress(fromAddress), ToAddress: common.BytesToAddress(toAddress),
		ValueWei: bigIntFromNumeric(value),
	}, nil
}

func numericFromBigInt(value *big.Int) pgtype.Numeric {
	return pgtype.Numeric{Int: new(big.Int).Set(value), Exp: 0, Valid: true}
}

func bigIntFromNumeric(value pgtype.Numeric) *big.Int {
	if !value.Valid || value.Int == nil {
		return new(big.Int)
	}
	result := new(big.Int).Set(value.Int)
	if value.Exp > 0 {
		return result.Mul(result, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(value.Exp)), nil))
	}
	if value.Exp < 0 {
		return result.Quo(result, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-value.Exp)), nil))
	}
	return result
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
