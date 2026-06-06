package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/application/model"
	appsqlc "github.com/useryege/athena/internal/application/store/sqlc"
)

const (
	outboxAggregateProject = "project"
)

func (s *SQLStore) UpsertProjectCandidateAndEnqueueQualification(ctx context.Context, candidate model.DiscoveredProjectCandidate) error {
	if candidate.Contract == (common.Address{}) {
		return errors.New("project discovery contract is empty")
	}
	if candidate.TxHash == (common.Hash{}) {
		return errors.New("project discovery tx hash is empty")
	}
	if err := validateProjectBaseNumbers(candidate.BlockNumber, candidate.BlockTime, candidate.TxIndex); err != nil {
		return err
	}
	chainID := s.chainIDForProject(candidate.ChainID)
	if candidate.Source == "" {
		candidate.Source = model.ProjectDiscoverySourceCatchUp
	}
	now := time.Now().UTC()
	return s.withProjectIntakeQuerier(ctx, "project discovery intake", func(queries appsqlc.Querier) error {
		payload, err := json.Marshal(candidatePayload(candidate))
		if err != nil {
			return fmt.Errorf("marshal project discovery payload: %w", err)
		}
		if candidate.CodeHash != (common.Hash{}) {
			if err := queries.UpsertBytecode(ctx, candidate.CodeHash.Bytes()); err != nil {
				return fmt.Errorf("upsert discovered project bytecode: %w", err)
			}
			if err := queries.UpsertContractBytecodeDeployment(ctx, appsqlc.UpsertContractBytecodeDeploymentParams{
				ChainID:  chainID,
				Contract: candidate.Contract.Bytes(),
				CodeHash: candidate.CodeHash.Bytes(),
			}); err != nil {
				return fmt.Errorf("upsert discovered project bytecode deployment: %w", err)
			}
		}
		if err := queries.UpsertProjectFromDiscovery(ctx, appsqlc.UpsertProjectFromDiscoveryParams{
			ChainID:     chainID,
			Contract:    candidate.Contract.Bytes(),
			Creator:     candidate.Creator.Bytes(),
			TxHash:      candidate.TxHash.Bytes(),
			BlockNumber: int64(candidate.BlockNumber),
			BlockTime:   int64(candidate.BlockTime),
			TxIndex:     int64(candidate.TxIndex),
			WethPair:    nullableAddressBytes(candidate.WethPair),
			UsdtPair:    nullableAddressBytes(candidate.UsdtPair),
			Source:      string(candidate.Source),
			Reason:      pgtype.Text{},
			Payload:     payload,
		}); err != nil {
			return fmt.Errorf("upsert discovered project: %w", err)
		}
		ref := model.ProjectRef{ChainID: chainID, Contract: candidate.Contract}
		if err := queries.UpsertProjectCollectionRequest(ctx, appsqlc.UpsertProjectCollectionRequestParams{
			RequestedAt:     pgTime(now),
			NextRunAt:       pgTime(now),
			ChainID:         chainID,
			ProjectContract: candidate.Contract.Bytes(),
		}); err != nil {
			return fmt.Errorf("upsert project collection request: %w", err)
		}
		_, err = insertOutboxEvent(ctx, queries, CreateOutboxEventParams{
			Type:          OutboxTypeProjectCollectionRequested,
			AggregateType: outboxAggregateProject,
			AggregateID:   projectDedupKey(candidate.Contract),
			ChainID:       chainID,
			DedupKey:      projectDedupKey(candidate.Contract),
			Payload: map[string]any{
				"project": ref,
				"reason":  string(candidate.Source),
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
		"code_hash":    optionalHashHex(candidate.CodeHash),
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
