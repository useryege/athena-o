package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	ethereumapisqlc "github.com/useryege/athena/internal/ethereumapi/store/sqlc"
)

const (
	ReceiptStatusUnknown int16 = -1
	ReceiptStatusFailed  int16 = 0
	ReceiptStatusSuccess int16 = 1
)

type NormalTransactionQuery struct {
	ChainID    int64
	Address    ethcommon.Address
	StartBlock uint64
	EndBlock   uint64
	Page       int32
	PageSize   int32
	SortOrder  string
}

func (q NormalTransactionQuery) CacheKey() string {
	return fmt.Sprintf(
		"%d:%s:%d:%d:%d:%d:%s",
		q.ChainID,
		q.Address.Hex(),
		q.StartBlock,
		q.EndBlock,
		q.Page,
		q.PageSize,
		q.SortOrder,
	)
}

type NormalTransaction struct {
	ChainID           int64
	TxHash            ethcommon.Hash
	BlockNumber       uint64
	BlockHash         ethcommon.Hash
	BlockTimestamp    uint64
	Nonce             uint64
	TransactionIndex  uint64
	FromAddress       ethcommon.Address
	ToAddress         *ethcommon.Address
	Value             string
	Gas               uint64
	GasPrice          string
	Input             string
	MethodID          []byte
	FunctionName      string
	ContractAddress   *ethcommon.Address
	CumulativeGasUsed uint64
	ReceiptStatus     int16
	GasUsed           uint64
	Confirmations     uint64
	IsError           bool
	FetchedAt         time.Time
}

type NormalTransactionCacheEntry struct {
	Query        NormalTransactionQuery
	Transactions []*NormalTransaction
	FetchedAt    time.Time
	ExpiresAt    time.Time
}

func (s *SQLStore) GetNormalTransactionCache(ctx context.Context, query NormalTransactionQuery) (*NormalTransactionCacheEntry, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("ethereumapi postgres database is not configured")
	}
	params, err := getNormalTransactionQueryCacheParams(query)
	if err != nil {
		return nil, err
	}
	cache, err := s.queries.GetNormalTransactionQueryCache(ctx, params)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get normal transaction query cache: %w", err)
	}
	rows, err := s.queries.ListNormalTransactionQueryItems(ctx, cache.ID)
	if err != nil {
		return nil, fmt.Errorf("list normal transaction query items: %w", err)
	}
	transactions := make([]*NormalTransaction, 0, len(rows))
	for _, row := range rows {
		transaction, err := normalTransactionFromRow(row)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, transaction)
	}
	return &NormalTransactionCacheEntry{
		Query:        query,
		Transactions: transactions,
		FetchedAt:    timestamptzValue(cache.FetchedAt),
		ExpiresAt:    timestamptzValue(cache.ExpiresAt),
	}, nil
}

func (s *SQLStore) ReplaceNormalTransactionCache(
	ctx context.Context,
	query NormalTransactionQuery,
	transactions []*NormalTransaction,
	fetchedAt time.Time,
	expiresAt time.Time,
) (*NormalTransactionCacheEntry, error) {
	if s == nil || s.pool == nil || s.queries == nil {
		return nil, fmt.Errorf("ethereumapi postgres database is not configured")
	}
	queryParams, err := upsertNormalTransactionQueryCacheParams(query, fetchedAt, expiresAt)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin normal transaction cache replacement: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := ethereumapisqlc.New(tx)
	if len(transactions) > 0 {
		params, err := batchUpsertNormalTransactionsParams(transactions, fetchedAt)
		if err != nil {
			return nil, err
		}
		if err := queries.BatchUpsertNormalTransactions(ctx, params); err != nil {
			return nil, fmt.Errorf("batch upsert normal transactions: %w", err)
		}
	}
	cache, err := queries.UpsertNormalTransactionQueryCache(ctx, queryParams)
	if err != nil {
		return nil, fmt.Errorf("upsert normal transaction query cache: %w", err)
	}
	if err := queries.DeleteNormalTransactionQueryItems(ctx, cache.ID); err != nil {
		return nil, fmt.Errorf("delete normal transaction query items: %w", err)
	}
	if len(transactions) > 0 {
		params := batchCreateNormalTransactionQueryItemsParams(cache.ID, transactions)
		if err := queries.BatchCreateNormalTransactionQueryItems(ctx, params); err != nil {
			return nil, fmt.Errorf("batch create normal transaction query items: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit normal transaction cache replacement: %w", err)
	}
	return &NormalTransactionCacheEntry{
		Query:        query,
		Transactions: transactions,
		FetchedAt:    fetchedAt,
		ExpiresAt:    expiresAt,
	}, nil
}

func (s *SQLStore) DeleteExpiredNormalTransactionCaches(ctx context.Context, deleteBefore time.Time) (int64, error) {
	if s == nil || s.queries == nil {
		return 0, fmt.Errorf("ethereumapi postgres database is not configured")
	}
	rows, err := s.queries.DeleteExpiredNormalTransactionQueryCaches(ctx, timestamptz(deleteBefore))
	if err != nil {
		return 0, fmt.Errorf("delete expired normal transaction query caches: %w", err)
	}
	return rows, nil
}

func getNormalTransactionQueryCacheParams(query NormalTransactionQuery) (ethereumapisqlc.GetNormalTransactionQueryCacheParams, error) {
	startBlock, err := uint64ToInt64("start_block", query.StartBlock)
	if err != nil {
		return ethereumapisqlc.GetNormalTransactionQueryCacheParams{}, err
	}
	endBlock, err := uint64ToInt64("end_block", query.EndBlock)
	if err != nil {
		return ethereumapisqlc.GetNormalTransactionQueryCacheParams{}, err
	}
	return ethereumapisqlc.GetNormalTransactionQueryCacheParams{
		ChainID:    query.ChainID,
		Address:    query.Address.Bytes(),
		StartBlock: startBlock,
		EndBlock:   endBlock,
		Page:       query.Page,
		PageSize:   query.PageSize,
		SortOrder:  query.SortOrder,
	}, nil
}

func upsertNormalTransactionQueryCacheParams(
	query NormalTransactionQuery,
	fetchedAt time.Time,
	expiresAt time.Time,
) (ethereumapisqlc.UpsertNormalTransactionQueryCacheParams, error) {
	params, err := getNormalTransactionQueryCacheParams(query)
	if err != nil {
		return ethereumapisqlc.UpsertNormalTransactionQueryCacheParams{}, err
	}
	return ethereumapisqlc.UpsertNormalTransactionQueryCacheParams{
		ChainID:    params.ChainID,
		Address:    params.Address,
		StartBlock: params.StartBlock,
		EndBlock:   params.EndBlock,
		Page:       params.Page,
		PageSize:   params.PageSize,
		SortOrder:  params.SortOrder,
		FetchedAt:  timestamptz(fetchedAt),
		ExpiresAt:  timestamptz(expiresAt),
	}, nil
}

func batchUpsertNormalTransactionsParams(
	transactions []*NormalTransaction,
	defaultFetchedAt time.Time,
) (ethereumapisqlc.BatchUpsertNormalTransactionsParams, error) {
	params := ethereumapisqlc.BatchUpsertNormalTransactionsParams{
		ChainIDValues:           make([]int64, 0, len(transactions)),
		TxHashValues:            make([][]byte, 0, len(transactions)),
		BlockNumberValues:       make([]int64, 0, len(transactions)),
		BlockHashValues:         make([][]byte, 0, len(transactions)),
		BlockTimestampValues:    make([]int64, 0, len(transactions)),
		NonceValues:             make([]pgtype.Numeric, 0, len(transactions)),
		TransactionIndexValues:  make([]int64, 0, len(transactions)),
		FromAddressValues:       make([][]byte, 0, len(transactions)),
		ToAddressValues:         make([][]byte, 0, len(transactions)),
		ValueValues:             make([]pgtype.Numeric, 0, len(transactions)),
		GasValues:               make([]pgtype.Numeric, 0, len(transactions)),
		GasPriceValues:          make([]pgtype.Numeric, 0, len(transactions)),
		InputValues:             make([]string, 0, len(transactions)),
		MethodIDValues:          make([][]byte, 0, len(transactions)),
		FunctionNameValues:      make([]string, 0, len(transactions)),
		ContractAddressValues:   make([][]byte, 0, len(transactions)),
		CumulativeGasUsedValues: make([]pgtype.Numeric, 0, len(transactions)),
		TxReceiptStatusValues:   make([]int16, 0, len(transactions)),
		GasUsedValues:           make([]pgtype.Numeric, 0, len(transactions)),
		ConfirmationsValues:     make([]pgtype.Numeric, 0, len(transactions)),
		IsErrorValues:           make([]bool, 0, len(transactions)),
		FetchedAtValues:         make([]pgtype.Timestamptz, 0, len(transactions)),
	}
	for index, transaction := range transactions {
		if transaction == nil {
			return ethereumapisqlc.BatchUpsertNormalTransactionsParams{}, fmt.Errorf("normal transaction %d is nil", index)
		}
		blockNumber, err := uint64ToInt64("block_number", transaction.BlockNumber)
		if err != nil {
			return ethereumapisqlc.BatchUpsertNormalTransactionsParams{}, err
		}
		blockTimestamp, err := uint64ToInt64("block_timestamp", transaction.BlockTimestamp)
		if err != nil {
			return ethereumapisqlc.BatchUpsertNormalTransactionsParams{}, err
		}
		transactionIndex, err := uint64ToInt64("transaction_index", transaction.TransactionIndex)
		if err != nil {
			return ethereumapisqlc.BatchUpsertNormalTransactionsParams{}, err
		}
		value, err := numericFromDecimalString("value", transaction.Value)
		if err != nil {
			return ethereumapisqlc.BatchUpsertNormalTransactionsParams{}, err
		}
		gasPrice, err := numericFromDecimalString("gas_price", transaction.GasPrice)
		if err != nil {
			return ethereumapisqlc.BatchUpsertNormalTransactionsParams{}, err
		}
		fetchedAt := transaction.FetchedAt
		if fetchedAt.IsZero() {
			fetchedAt = defaultFetchedAt
		}
		params.ChainIDValues = append(params.ChainIDValues, transaction.ChainID)
		params.TxHashValues = append(params.TxHashValues, transaction.TxHash.Bytes())
		params.BlockNumberValues = append(params.BlockNumberValues, blockNumber)
		params.BlockHashValues = append(params.BlockHashValues, transaction.BlockHash.Bytes())
		params.BlockTimestampValues = append(params.BlockTimestampValues, blockTimestamp)
		params.NonceValues = append(params.NonceValues, numericFromUint64(transaction.Nonce))
		params.TransactionIndexValues = append(params.TransactionIndexValues, transactionIndex)
		params.FromAddressValues = append(params.FromAddressValues, transaction.FromAddress.Bytes())
		params.ToAddressValues = append(params.ToAddressValues, optionalAddressBytes(transaction.ToAddress))
		params.ValueValues = append(params.ValueValues, value)
		params.GasValues = append(params.GasValues, numericFromUint64(transaction.Gas))
		params.GasPriceValues = append(params.GasPriceValues, gasPrice)
		params.InputValues = append(params.InputValues, transaction.Input)
		params.MethodIDValues = append(params.MethodIDValues, optionalBytes(transaction.MethodID))
		params.FunctionNameValues = append(params.FunctionNameValues, transaction.FunctionName)
		params.ContractAddressValues = append(params.ContractAddressValues, optionalAddressBytes(transaction.ContractAddress))
		params.CumulativeGasUsedValues = append(params.CumulativeGasUsedValues, numericFromUint64(transaction.CumulativeGasUsed))
		params.TxReceiptStatusValues = append(params.TxReceiptStatusValues, transaction.ReceiptStatus)
		params.GasUsedValues = append(params.GasUsedValues, numericFromUint64(transaction.GasUsed))
		params.ConfirmationsValues = append(params.ConfirmationsValues, numericFromUint64(transaction.Confirmations))
		params.IsErrorValues = append(params.IsErrorValues, transaction.IsError)
		params.FetchedAtValues = append(params.FetchedAtValues, timestamptz(fetchedAt))
	}
	return params, nil
}

func batchCreateNormalTransactionQueryItemsParams(
	queryID int64,
	transactions []*NormalTransaction,
) ethereumapisqlc.BatchCreateNormalTransactionQueryItemsParams {
	params := ethereumapisqlc.BatchCreateNormalTransactionQueryItemsParams{
		QueryIDValues:  make([]int64, 0, len(transactions)),
		PositionValues: make([]int32, 0, len(transactions)),
		ChainIDValues:  make([]int64, 0, len(transactions)),
		TxHashValues:   make([][]byte, 0, len(transactions)),
	}
	for index, transaction := range transactions {
		params.QueryIDValues = append(params.QueryIDValues, queryID)
		params.PositionValues = append(params.PositionValues, int32(index))
		params.ChainIDValues = append(params.ChainIDValues, transaction.ChainID)
		params.TxHashValues = append(params.TxHashValues, transaction.TxHash.Bytes())
	}
	return params
}

func normalTransactionFromRow(row ethereumapisqlc.ListNormalTransactionQueryItemsRow) (*NormalTransaction, error) {
	blockNumber, err := int64ToUint64("block_number", row.BlockNumber)
	if err != nil {
		return nil, err
	}
	blockTimestamp, err := int64ToUint64("block_timestamp", row.BlockTimestamp)
	if err != nil {
		return nil, err
	}
	transactionIndex, err := int64ToUint64("transaction_index", row.TransactionIndex)
	if err != nil {
		return nil, err
	}
	nonce, err := numericToUint64("nonce", row.Nonce)
	if err != nil {
		return nil, err
	}
	gas, err := numericToUint64("gas", row.Gas)
	if err != nil {
		return nil, err
	}
	cumulativeGasUsed, err := numericToUint64("cumulative_gas_used", row.CumulativeGasUsed)
	if err != nil {
		return nil, err
	}
	gasUsed, err := numericToUint64("gas_used", row.GasUsed)
	if err != nil {
		return nil, err
	}
	confirmations, err := numericToUint64("confirmations", row.Confirmations)
	if err != nil {
		return nil, err
	}
	return &NormalTransaction{
		ChainID:           row.ChainID,
		TxHash:            ethcommon.BytesToHash(row.TxHash),
		BlockNumber:       blockNumber,
		BlockHash:         ethcommon.BytesToHash(row.BlockHash),
		BlockTimestamp:    blockTimestamp,
		Nonce:             nonce,
		TransactionIndex:  transactionIndex,
		FromAddress:       ethcommon.BytesToAddress(row.FromAddress),
		ToAddress:         addressPointer(row.ToAddress),
		Value:             numericToDecimalString(row.Value),
		Gas:               gas,
		GasPrice:          numericToDecimalString(row.GasPrice),
		Input:             row.Input,
		MethodID:          optionalBytes(row.MethodID),
		FunctionName:      row.FunctionName,
		ContractAddress:   addressPointer(row.ContractAddress),
		CumulativeGasUsed: cumulativeGasUsed,
		ReceiptStatus:     row.TxReceiptStatus,
		GasUsed:           gasUsed,
		Confirmations:     confirmations,
		IsError:           row.IsError,
		FetchedAt:         timestamptzValue(row.FetchedAt),
	}, nil
}

func uint64ToInt64(field string, value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("%s exceeds supported range", field)
	}
	return int64(value), nil
}

func int64ToUint64(field string, value int64) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("%s must not be negative", field)
	}
	return uint64(value), nil
}

func numericFromUint64(value uint64) pgtype.Numeric {
	return pgtype.Numeric{
		Int:   new(big.Int).SetUint64(value),
		Exp:   0,
		Valid: true,
	}
}

func numericFromDecimalString(field string, value string) (pgtype.Numeric, error) {
	integer, ok := new(big.Int).SetString(value, 10)
	if !ok || integer.Sign() < 0 {
		return pgtype.Numeric{}, fmt.Errorf("%s must be a nonnegative decimal integer", field)
	}
	return pgtype.Numeric{Int: integer, Exp: 0, Valid: true}, nil
}

func numericToUint64(field string, value pgtype.Numeric) (uint64, error) {
	integer := bigIntFromNumeric(value)
	if integer.Sign() < 0 || !integer.IsUint64() {
		return 0, fmt.Errorf("%s exceeds uint64 range", field)
	}
	return integer.Uint64(), nil
}

func numericToDecimalString(value pgtype.Numeric) string {
	return bigIntFromNumeric(value).String()
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

func optionalAddressBytes(value *ethcommon.Address) []byte {
	if value == nil {
		return nil
	}
	return value.Bytes()
}

func addressPointer(value []byte) *ethcommon.Address {
	if len(value) == 0 {
		return nil
	}
	address := ethcommon.BytesToAddress(value)
	return &address
}

func optionalBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}
	result := make([]byte, len(value))
	copy(result, value)
	return result
}

func timestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func timestamptzValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}
