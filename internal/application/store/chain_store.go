package store

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

const (
	ChainIngestStatusStopped = "stopped"
	ChainIngestStatusRunning = "running"
)

func (s *SQLStore) GetChainIngestCheckpoint(ctx context.Context, chainID int64) (*ChainIngestCheckpoint, error) {
	if chainID <= 0 {
		return nil, fmt.Errorf("chain_id must be positive")
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.GetChainIngestCheckpoint(ctx, chainID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get chain ingest checkpoint: %w", err)
	}
	item, err := chainIngestCheckpointFromSQLC(row.ChainID, row.ChainName, row.Enabled, row.FinalizedBlockNumber, row.FinalizedBlockHash, row.CursorBlockNumber, row.CursorBlockHash, row.Status, row.LockedAt, row.LockedBy, row.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *SQLStore) ListChainIngestCheckpoints(ctx context.Context) ([]ChainIngestCheckpoint, error) {
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListChainIngestCheckpoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("list chain ingest checkpoints: %w", err)
	}
	items := make([]ChainIngestCheckpoint, 0, len(rows))
	for _, row := range rows {
		item, err := chainIngestCheckpointFromSQLC(row.ChainID, row.ChainName, row.Enabled, row.FinalizedBlockNumber, row.FinalizedBlockHash, row.CursorBlockNumber, row.CursorBlockHash, row.Status, row.LockedAt, row.LockedBy, row.UpdatedAt)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *SQLStore) UpsertChainIngestCheckpoint(ctx context.Context, item ChainIngestCheckpoint) (*ChainIngestCheckpoint, error) {
	if item.ChainID <= 0 {
		return nil, fmt.Errorf("chain_id must be positive")
	}
	if item.Status == "" {
		item.Status = ChainIngestStatusRunning
	}
	if item.FinalizedBlockNumber > math.MaxInt64 || item.CursorBlockNumber > math.MaxInt64 {
		return nil, fmt.Errorf("chain ingest checkpoint block number exceeds int64")
	}
	queries, err := s.querier()
	if err != nil {
		return nil, err
	}
	row, err := queries.UpsertChainIngestCheckpoint(ctx, appsqlc.UpsertChainIngestCheckpointParams{
		ChainID:              item.ChainID,
		FinalizedBlockNumber: int64(item.FinalizedBlockNumber),
		FinalizedBlockHash:   nullableHashBytes(item.FinalizedBlockHash),
		CursorBlockNumber:    int64(item.CursorBlockNumber),
		CursorBlockHash:      nullableHashBytes(item.CursorBlockHash),
		Status:               strings.TrimSpace(item.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("upsert chain ingest checkpoint: %w", err)
	}
	updated, err := chainIngestCheckpointFromSQLC(row.ChainID, item.ChainName, item.Enabled, row.FinalizedBlockNumber, row.FinalizedBlockHash, row.CursorBlockNumber, row.CursorBlockHash, row.Status, row.LockedAt, row.LockedBy, row.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &updated, nil
}

func chainIngestCheckpointFromSQLC(chainID int64, chainName string, enabled bool, finalizedBlockNumber int64, finalizedBlockHash []byte, cursorBlockNumber int64, cursorBlockHash []byte, status string, lockedAt pgtype.Timestamptz, lockedBy pgtype.Text, updatedAt pgtype.Timestamptz) (ChainIngestCheckpoint, error) {
	if finalizedBlockNumber < 0 || cursorBlockNumber < 0 {
		return ChainIngestCheckpoint{}, fmt.Errorf("chain ingest checkpoint contains negative block number")
	}
	item := ChainIngestCheckpoint{
		ChainID:              chainID,
		ChainName:            chainName,
		Enabled:              enabled,
		FinalizedBlockNumber: uint64(finalizedBlockNumber),
		FinalizedBlockHash:   hashFromNullableBytes(finalizedBlockHash),
		CursorBlockNumber:    uint64(cursorBlockNumber),
		CursorBlockHash:      hashFromNullableBytes(cursorBlockHash),
		Status:               status,
		LockedBy:             lockedBy.String,
	}
	if lockedAt.Valid {
		item.LockedAt = lockedAt.Time
	}
	if updatedAt.Valid {
		item.UpdatedAt = updatedAt.Time
	}
	return item, nil
}

func hashFromNullableBytes(value []byte) common.Hash {
	if len(value) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(value)
}
