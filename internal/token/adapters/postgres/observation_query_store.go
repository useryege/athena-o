package postgres

import (
	"context"
	"encoding/json"

	"github.com/useryege/athena/internal/token/reporting"
)

func (repository *ReportingRepository) ListObservationSnapshots(ctx context.Context, projectID int64) ([]reporting.ObservationSnapshot, error) {
	queries, err := repository.querier()
	if err != nil {
		return nil, err
	}
	rows, err := queries.ListCurrentProjectObservations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	result := make([]reporting.ObservationSnapshot, 0, len(rows))
	for _, row := range rows {
		blockNumber, err := uint64PointerFromInt64("block_number", row.BlockNumber)
		if err != nil {
			return nil, err
		}
		result = append(result, reporting.ObservationSnapshot{ID: row.ID, ProjectID: row.ProjectID, DataType: reporting.DataCollectionType(row.DataType), SchemaVersion: row.SchemaVersion, ContentHash: bytesToHash(row.ContentHash), Payload: json.RawMessage(row.Payload), BlockNumber: blockNumber, ObservedAt: timeValue(row.ObservedAt), LastCheckedAt: timeValue(row.LastCheckedAt)})
	}
	return result, nil
}
