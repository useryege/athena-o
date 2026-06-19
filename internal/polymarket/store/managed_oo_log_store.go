package store

import (
	"context"
	"errors"
	"fmt"
	"math"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	polymarketsqlc "github.com/useryege/athena/internal/polymarket/store/sqlc"
)

func (s *SQLStore) GetPolymarketChainLogCursor(ctx context.Context, syncName string) (*ChainLogCursor, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	row, err := s.queries.GetPolymarketChainLogCursor(ctx, syncName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get polymarket chain log cursor: %w", err)
	}
	cursor, err := mapPolymarketChainLogCursor(row)
	if err != nil {
		return nil, err
	}
	return cursor, nil
}

func (s *SQLStore) UpsertPolymarketChainLogCursor(ctx context.Context, cursor ChainLogCursor) (*ChainLogCursor, error) {
	if s == nil || s.queries == nil {
		return nil, fmt.Errorf("polymarket postgres database is not configured")
	}
	params, err := upsertPolymarketChainLogCursorParams(cursor)
	if err != nil {
		return nil, err
	}
	row, err := s.queries.UpsertPolymarketChainLogCursor(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("upsert polymarket chain log cursor: %w", err)
	}
	mapped, err := mapPolymarketChainLogCursor(row)
	if err != nil {
		return nil, err
	}
	return mapped, nil
}

func (s *SQLStore) IngestManagedOOProposePriceLogs(ctx context.Context, cursor ChainLogCursor, logs []ManagedOOProposePriceLog) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("polymarket postgres database is not configured")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin managed oo propose price log sync: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := s.queries.WithTx(tx)
	if len(logs) > 0 {
		params, err := batchUpsertManagedOOProposePriceLogsParams(logs)
		if err != nil {
			return err
		}
		if err := queries.BatchUpsertManagedOOProposePriceLogs(ctx, params); err != nil {
			return fmt.Errorf("batch upsert managed oo propose price logs: %w", err)
		}
	}
	cursorParams, err := upsertPolymarketChainLogCursorParams(cursor)
	if err != nil {
		return err
	}
	if _, err := queries.UpsertPolymarketChainLogCursor(ctx, cursorParams); err != nil {
		return fmt.Errorf("upsert managed oo propose price cursor: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit managed oo propose price log sync: %w", err)
	}
	return nil
}

func upsertPolymarketChainLogCursorParams(cursor ChainLogCursor) (polymarketsqlc.UpsertPolymarketChainLogCursorParams, error) {
	lastBlockNumber, err := uint64ToInt64("last_block_number", cursor.LastBlockNumber)
	if err != nil {
		return polymarketsqlc.UpsertPolymarketChainLogCursorParams{}, err
	}
	return polymarketsqlc.UpsertPolymarketChainLogCursorParams{
		SyncName:        cursor.SyncName,
		ContractAddress: cursor.ContractAddress,
		Topic:           cursor.Topic,
		LastBlockNumber: lastBlockNumber,
		LastPolledAt:    nullableTime(cursor.LastPolledAt),
	}, nil
}

func batchUpsertManagedOOProposePriceLogsParams(logs []ManagedOOProposePriceLog) (polymarketsqlc.BatchUpsertManagedOOProposePriceLogsParams, error) {
	params := polymarketsqlc.BatchUpsertManagedOOProposePriceLogsParams{
		TxHashes:                make([]string, 0, len(logs)),
		LogIndexes:              make([]int64, 0, len(logs)),
		BlockNumbers:            make([]int64, 0, len(logs)),
		BlockHashes:             make([]string, 0, len(logs)),
		TxIndexes:               make([]int64, 0, len(logs)),
		ContractAddresses:       make([]string, 0, len(logs)),
		Topics:                  make([]string, 0, len(logs)),
		Requesters:              make([]string, 0, len(logs)),
		Proposers:               make([]string, 0, len(logs)),
		Identifiers:             make([]string, 0, len(logs)),
		RequestTimestamps:       make([]int64, 0, len(logs)),
		AncillaryDataHexValues:  make([]string, 0, len(logs)),
		AncillaryDataTextValues: make([]string, 0, len(logs)),
		MarketIds:               make([]string, 0, len(logs)),
		ProposedPrices:          make([]string, 0, len(logs)),
		ExpirationTimestamps:    make([]int64, 0, len(logs)),
		Currencies:              make([]string, 0, len(logs)),
		RawTopicsValues:         make([][]byte, 0, len(logs)),
		RawDataValues:           make([]string, 0, len(logs)),
		FetchedAtValues:         make([]pgtype.Timestamptz, 0, len(logs)),
	}
	for _, item := range logs {
		logIndex, err := uintToInt64("log_index", item.LogIndex)
		if err != nil {
			return params, err
		}
		blockNumber, err := uint64ToInt64("block_number", item.BlockNumber)
		if err != nil {
			return params, err
		}
		txIndex, err := uintToInt64("tx_index", item.TxIndex)
		if err != nil {
			return params, err
		}
		requestTimestamp, err := uint64ToInt64("request_timestamp", item.RequestTimestamp)
		if err != nil {
			return params, err
		}
		expirationTimestamp, err := uint64ToInt64("expiration_timestamp", item.ExpirationTimestamp)
		if err != nil {
			return params, err
		}
		params.TxHashes = append(params.TxHashes, item.TxHash)
		params.LogIndexes = append(params.LogIndexes, logIndex)
		params.BlockNumbers = append(params.BlockNumbers, blockNumber)
		params.BlockHashes = append(params.BlockHashes, item.BlockHash)
		params.TxIndexes = append(params.TxIndexes, txIndex)
		params.ContractAddresses = append(params.ContractAddresses, item.ContractAddress)
		params.Topics = append(params.Topics, item.Topic)
		params.Requesters = append(params.Requesters, item.Requester)
		params.Proposers = append(params.Proposers, item.Proposer)
		params.Identifiers = append(params.Identifiers, item.Identifier)
		params.RequestTimestamps = append(params.RequestTimestamps, requestTimestamp)
		params.AncillaryDataHexValues = append(params.AncillaryDataHexValues, item.AncillaryDataHex)
		params.AncillaryDataTextValues = append(params.AncillaryDataTextValues, item.AncillaryDataText)
		params.MarketIds = append(params.MarketIds, item.MarketID)
		params.ProposedPrices = append(params.ProposedPrices, item.ProposedPrice)
		params.ExpirationTimestamps = append(params.ExpirationTimestamps, expirationTimestamp)
		params.Currencies = append(params.Currencies, item.Currency)
		params.RawTopicsValues = append(params.RawTopicsValues, jsonBytes(item.RawTopics, jsonArray))
		params.RawDataValues = append(params.RawDataValues, item.RawData)
		params.FetchedAtValues = append(params.FetchedAtValues, nullableTime(item.FetchedAt))
	}
	return params, nil
}

func mapPolymarketChainLogCursor(row polymarketsqlc.PolymarketChainLogCursor) (*ChainLogCursor, error) {
	lastBlockNumber, err := int64ToUint64("last_block_number", row.LastBlockNumber)
	if err != nil {
		return nil, err
	}
	return &ChainLogCursor{
		SyncName:        row.SyncName,
		ContractAddress: row.ContractAddress,
		Topic:           row.Topic,
		LastBlockNumber: lastBlockNumber,
		LastPolledAt:    timeValue(row.LastPolledAt),
		CreatedAt:       timeValue(row.CreatedAt),
		UpdatedAt:       timeValue(row.UpdatedAt),
	}, nil
}

func uint64ToInt64(name string, value uint64) (int64, error) {
	if value > math.MaxInt64 {
		return 0, fmt.Errorf("%s overflows int64: %d", name, value)
	}
	return int64(value), nil
}

func uintToInt64(name string, value uint) (int64, error) {
	if uint64(value) > math.MaxInt64 {
		return 0, fmt.Errorf("%s overflows int64: %d", name, value)
	}
	return int64(value), nil
}

func int64ToUint64(name string, value int64) (uint64, error) {
	if value < 0 {
		return 0, fmt.Errorf("%s is negative: %d", name, value)
	}
	return uint64(value), nil
}
