package store

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/application/model"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

const (
	outboxAggregateProjectCandidate = "project_candidate"
	outboxAggregateProject          = "project"
)

func (s *SQLStore) UpsertProjectCandidateAndEnqueueQualification(ctx context.Context, candidate model.DiscoveredProjectCandidate) error {
	if candidate.Contract == (common.Address{}) {
		return nil
	}
	chainID := s.chainIDForProject(candidate.ChainID)
	if candidate.Source == "" {
		candidate.Source = model.ProjectDiscoverySourceCatchUp
	}
	return s.withProjectIntakeQuerier(ctx, "project candidate intake", func(queries appsqlc.Querier) error {
		payload, err := json.Marshal(candidatePayload(candidate))
		if err != nil {
			return fmt.Errorf("marshal project candidate payload: %w", err)
		}
		if err := queries.UpsertProjectCandidate(ctx, appsqlc.UpsertProjectCandidateParams{
			ChainID:     chainID,
			Contract:    candidate.Contract.Bytes(),
			Creator:     nullableAddressBytes(candidate.Creator),
			TxHash:      nullableHashBytes(candidate.TxHash),
			BlockNumber: nullableUint64(candidate.BlockNumber),
			BlockTime:   nullableUint64(candidate.BlockTime),
			TxIndex:     nullableUint64(candidate.TxIndex),
			WethPair:    nullableAddressBytes(candidate.WethPair),
			UsdtPair:    nullableAddressBytes(candidate.UsdtPair),
			Source:      string(candidate.Source),
			Payload:     payload,
		}); err != nil {
			return fmt.Errorf("upsert project candidate: %w", err)
		}
		_, err = insertOutboxEvent(ctx, queries, CreateOutboxEventParams{
			Type:          OutboxTypeCandidateQualificationRequested,
			AggregateType: outboxAggregateProjectCandidate,
			AggregateID:   projectDedupKey(candidate.Contract),
			ChainID:       chainID,
			DedupKey:      projectDedupKey(candidate.Contract),
			Payload: map[string]any{
				"project":   model.ProjectRef{ChainID: chainID, Contract: candidate.Contract},
				"source":    candidate.Source,
				"candidate": candidatePayload(candidate),
			},
		})
		if err != nil {
			return err
		}
		return nil
	})
}

func (s *SQLStore) EnqueueProjectCollection(ctx context.Context, ref model.ProjectRef, reason string) error {
	if ref.Contract == (common.Address{}) {
		return nil
	}
	chainID := s.chainIDForProject(ref.ChainID)
	ref.ChainID = chainID
	now := time.Now().UTC()
	return s.withProjectIntakeQuerier(ctx, "project collection intake", func(queries appsqlc.Querier) error {
		if err := queries.UpsertProjectCollectionRequest(ctx, appsqlc.UpsertProjectCollectionRequestParams{
			RequestedAt:     pgTime(now),
			NextRunAt:       pgTime(now),
			ChainID:         chainID,
			ProjectContract: ref.Contract.Bytes(),
		}); err != nil {
			return fmt.Errorf("upsert project collection request: %w", err)
		}
		_, err := insertOutboxEvent(ctx, queries, CreateOutboxEventParams{
			Type:          OutboxTypeProjectCollectionRequested,
			AggregateType: outboxAggregateProject,
			AggregateID:   projectDedupKey(ref.Contract),
			ChainID:       chainID,
			DedupKey:      projectDedupKey(ref.Contract),
			Payload: map[string]any{
				"project": ref,
				"reason":  strings.TrimSpace(reason),
			},
		})
		return err
	})
}

func (s *SQLStore) withProjectIntakeQuerier(ctx context.Context, operation string, fn func(appsqlc.Querier) error) error {
	if fn == nil {
		return nil
	}
	if s.pool == nil {
		queries, err := s.querier()
		if err != nil {
			return err
		}
		return fn(queries)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin %s tx: %w", operation, err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	if err := fn(appsqlc.New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit %s tx: %w", operation, err)
	}
	return nil
}

func projectDedupKey(contract common.Address) string {
	return strings.ToLower(contract.Hex())
}

func candidatePayload(candidate model.DiscoveredProjectCandidate) map[string]any {
	return map[string]any{
		"chain_id":     candidate.ChainID,
		"contract":     candidate.Contract.Hex(),
		"creator":      optionalAddressHex(candidate.Creator),
		"tx_hash":      optionalHashHex(candidate.TxHash),
		"block_number": candidate.BlockNumber,
		"block_time":   candidate.BlockTime,
		"tx_index":     candidate.TxIndex,
		"weth_pair":    optionalAddressHex(candidate.WethPair),
		"usdt_pair":    optionalAddressHex(candidate.UsdtPair),
		"source":       candidate.Source,
	}
}

func optionalAddressHex(value common.Address) string {
	if value == (common.Address{}) {
		return ""
	}
	return value.Hex()
}

func optionalHashHex(value common.Hash) string {
	if value == (common.Hash{}) {
		return ""
	}
	return value.Hex()
}

func nullableUint64(value uint64) pgtype.Int8 {
	if value > math.MaxInt64 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: int64(value), Valid: true}
}
