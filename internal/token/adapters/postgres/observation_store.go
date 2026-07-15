package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/research"
)

func (s *Database) CompleteProjectAveDataCollection(ctx context.Context, task research.ProjectDataCollectionTask, observation research.AveObservationV1, checkedAt time.Time) (*research.ProjectObservation, error) {
	return s.completeTypedObservation(ctx, task, observation, nil, checkedAt, false)
}
func (s *Database) CompleteProjectChainStateCollection(ctx context.Context, task research.ProjectDataCollectionTask, observation research.ChainStateObservationV1, blockNumber uint64, checkedAt time.Time) (*research.ProjectObservation, error) {
	return s.completeTypedObservation(ctx, task, observation, &blockNumber, checkedAt, false)
}
func (s *Database) CompleteProjectWalletAssetStateCollection(ctx context.Context, task research.ProjectDataCollectionTask, observation research.WalletAssetObservationV1, blockNumber uint64, checkedAt time.Time) error {
	_, e := s.completeTypedObservation(ctx, task, observation, &blockNumber, checkedAt, false)
	return e
}
func (s *Database) CompleteProjectSimulationResultCollection(ctx context.Context, task research.ProjectDataCollectionTask, observation research.SimulationObservationV1, blockNumber uint64, checkedAt time.Time) error {
	_, e := s.completeTypedObservation(ctx, task, observation, &blockNumber, checkedAt, false)
	return e
}

func (s *Database) CompleteProjectContractCodeSourceCollection(ctx context.Context, task research.ProjectDataCollectionTask, codeHash common.Hash, sourceCode string, checkedAt time.Time) error {
	if strings.TrimSpace(sourceCode) == "" {
		return s.completeUnavailableContractSource(ctx, task, checkedAt)
	}
	if s == nil || s.pool == nil {
		return fmt.Errorf("token postgres database is not configured")
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	if _, e = q.UpdateContractCodeSource(ctx, tokensqlc.UpdateContractCodeSourceParams{CodeHash: codeHash.Bytes(), SourceCode: nullableText(sourceCode), SourceCodeFetchedAt: nullableTime(checkedAt)}); e != nil {
		return e
	}
	payload, _ := json.Marshal(research.ContractSourceObservationV1{CodeHash: codeHash, SourceAvailable: true})
	if _, e = recordObservation(ctx, q, task.ProjectID, task.DataType, research.ObservationSchemaVersionV1, payload, nil, checkedAt); e != nil {
		return e
	}
	if n, e := q.MarkProjectDataCollectionTaskSucceeded(ctx, task.ID); e != nil {
		return e
	} else if n == 0 {
		return nil
	}
	if _, e = q.CompleteProjectDataCollectionSchedule(ctx, tokensqlc.CompleteProjectDataCollectionScheduleParams{LastCheckedAt: nullableTime(checkedAt), ProjectID: task.ProjectID, DataType: string(task.DataType)}); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (s *Database) completeUnavailableContractSource(ctx context.Context, task research.ProjectDataCollectionTask, checkedAt time.Time) error {
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	n, e := q.MarkProjectDataCollectionTaskSucceeded(ctx, task.ID)
	if e != nil {
		return e
	}
	if n == 0 {
		return nil
	}
	if e = markScheduleSucceeded(ctx, q, task.ProjectID, task.DataType, checkedAt); e != nil {
		return e
	}
	return tx.Commit(ctx)
}

func (s *Database) completeTypedObservation(ctx context.Context, task research.ProjectDataCollectionTask, observation any, blockNumber *uint64, checkedAt time.Time, completeSchedule bool) (*research.ProjectObservation, error) {
	payload, e := json.Marshal(observation)
	if e != nil {
		return nil, e
	}
	return s.completeObservation(ctx, task, payload, blockNumber, checkedAt, completeSchedule)
}

func (s *Database) completeObservation(ctx context.Context, task research.ProjectDataCollectionTask, payload json.RawMessage, blockNumber *uint64, checkedAt time.Time, completeSchedule bool) (*research.ProjectObservation, error) {
	if checkedAt.IsZero() {
		checkedAt = time.Now().UTC()
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := tokensqlc.New(tx)
	obs, e := recordObservation(ctx, q, task.ProjectID, task.DataType, research.ObservationSchemaVersionV1, payload, blockNumber, checkedAt)
	if e != nil {
		return nil, e
	}
	n, e := q.MarkProjectDataCollectionTaskSucceeded(ctx, task.ID)
	if e != nil {
		return nil, e
	}
	if n == 0 {
		return nil, nil
	}
	if completeSchedule {
		_, e = q.CompleteProjectDataCollectionSchedule(ctx, tokensqlc.CompleteProjectDataCollectionScheduleParams{LastCheckedAt: nullableTime(checkedAt), ProjectID: task.ProjectID, DataType: string(task.DataType)})
	} else {
		e = markScheduleSucceeded(ctx, q, task.ProjectID, task.DataType, checkedAt)
	}
	if e != nil {
		return nil, e
	}
	if e = tx.Commit(ctx); e != nil {
		return nil, e
	}
	return obs, nil
}

func recordObservation(ctx context.Context, q *tokensqlc.Queries, projectID int64, dataType research.DataCollectionType, schemaVersion int32, payload json.RawMessage, blockNumber *uint64, checkedAt time.Time) (*research.ProjectObservation, error) {
	canonical, e := canonicalJSON(payload)
	if e != nil {
		return nil, fmt.Errorf("canonicalize %s observation: %w", dataType, e)
	}
	hashInput, _ := json.Marshal(struct {
		SchemaVersion int32           `json:"schemaVersion"`
		Payload       json.RawMessage `json:"payload"`
	}{SchemaVersion: schemaVersion, Payload: canonical})
	digest := sha256.Sum256(hashInput)
	current, e := q.GetCurrentProjectObservation(ctx, tokensqlc.GetCurrentProjectObservationParams{ProjectID: projectID, DataType: string(dataType)})
	if e == nil && current.SchemaVersion == schemaVersion && bytes.Equal(current.ContentHash, digest[:]) {
		if _, e = q.TouchCurrentProjectObservation(ctx, tokensqlc.TouchCurrentProjectObservationParams{LastCheckedAt: nullableTime(checkedAt), ProjectID: projectID, DataType: string(dataType)}); e != nil {
			return nil, e
		}
		v, e := mapCurrentProjectObservation(current)
		v.LastCheckedAt = checkedAt
		return &v, e
	}
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return nil, e
	}
	block, e := nullableUint64(blockNumber)
	if e != nil {
		return nil, e
	}
	row, e := q.InsertProjectObservation(ctx, tokensqlc.InsertProjectObservationParams{ProjectID: projectID, DataType: string(dataType), SchemaVersion: schemaVersion, ContentHash: digest[:], Payload: canonical, BlockNumber: block, ObservedAt: nullableTime(checkedAt)})
	if e != nil {
		return nil, e
	}
	if _, e = q.UpsertCurrentProjectObservation(ctx, tokensqlc.UpsertCurrentProjectObservationParams{ProjectID: projectID, DataType: string(dataType), ObservationID: row.ID, LastCheckedAt: nullableTime(checkedAt)}); e != nil {
		return nil, e
	}
	state, e := q.IncrementProjectEvidenceRevision(ctx, projectID)
	if e != nil {
		return nil, e
	}
	if _, e = q.EnqueueProjectReportBuildTask(ctx, tokensqlc.EnqueueProjectReportBuildTaskParams{ProjectID: projectID, EvidenceRevision: state.EvidenceRevision}); e != nil {
		return nil, e
	}
	v, e := mapProjectObservation(row)
	v.LastCheckedAt = checkedAt
	return &v, e
}

func canonicalJSON(payload []byte) ([]byte, error) {
	var value any
	if e := json.Unmarshal(payload, &value); e != nil {
		return nil, e
	}
	return json.Marshal(value)
}
