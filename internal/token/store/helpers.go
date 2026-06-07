package store

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5/pgtype"
	tokensqlc "github.com/useryege/athena/internal/token/store/sqlc"
)

const (
	defaultPageSize = int32(20)
	maxPageSize     = int32(200)
)

func bytesToHash(value []byte) common.Hash {
	if len(value) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(value)
}

func bytesToAddress(value []byte) common.Address {
	if len(value) == 0 {
		return common.Address{}
	}
	return common.BytesToAddress(value)
}

func optionalHashBytes(value common.Hash) []byte {
	if value == (common.Hash{}) {
		return nil
	}
	return value.Bytes()
}

func optionalAddressBytes(value common.Address) []byte {
	if value == (common.Address{}) {
		return nil
	}
	return value.Bytes()
}

func nullableText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func textValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableTime(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func timeValue(value pgtype.Timestamptz) time.Time {
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

func normalizePage(page, pageSize int32) (int32, int32, int32) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return page, pageSize, (page - 1) * pageSize
}

func mapChainCheckpointRow(row tokensqlc.GetChainIngestCheckpointRow) (*ChainIngestCheckpoint, error) {
	cursorBlockNumber, err := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	return &ChainIngestCheckpoint{
		ChainID:           row.ChainID,
		ChainName:         row.ChainName,
		Enabled:           row.Enabled,
		CursorBlockNumber: cursorBlockNumber,
		Status:            row.Status,
		CreatedAt:         timeValue(row.CreatedAt),
	}, nil
}

func mapChainCheckpointListRow(row tokensqlc.ListChainIngestCheckpointsRow) (*ChainIngestCheckpoint, error) {
	return mapChainCheckpointRow(tokensqlc.GetChainIngestCheckpointRow(row))
}

func mapChainCheckpoint(row tokensqlc.ChainIngestCheckpoint) (*ChainIngestCheckpoint, error) {
	cursorBlockNumber, err := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	return &ChainIngestCheckpoint{
		ChainID:           row.ChainID,
		CursorBlockNumber: cursorBlockNumber,
		Status:            row.Status,
		CreatedAt:         timeValue(row.CreatedAt),
	}, nil
}

func mapProjectCandidate(row tokensqlc.ProjectCandidate) (*ProjectCandidate, error) {
	txIndex, err := int64ToUint64("tx_index", row.TxIndex)
	if err != nil {
		return nil, err
	}
	blockNumber, err := int64ToUint64("block_number", row.BlockNumber)
	if err != nil {
		return nil, err
	}
	blockTime, err := int64ToUint64("block_time", row.BlockTime)
	if err != nil {
		return nil, err
	}
	return &ProjectCandidate{
		ID:          row.ID,
		ChainID:     row.ChainID,
		Contract:    bytesToAddress(row.Contract),
		Creator:     bytesToAddress(row.Creator),
		TxHash:      bytesToHash(row.TxHash),
		TxIndex:     txIndex,
		BlockNumber: blockNumber,
		BlockTime:   blockTime,
		Status:      row.Status,
		CreatedAt:   timeValue(row.CreatedAt),
	}, nil
}

func mapProjectCandidates(rows []tokensqlc.ProjectCandidate) ([]ProjectCandidate, error) {
	items := make([]ProjectCandidate, 0, len(rows))
	for _, row := range rows {
		item, err := mapProjectCandidate(row)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

func mapContractCode(row tokensqlc.ContractCode) *ContractCode {
	return &ContractCode{
		CodeHash:            bytesToHash(row.CodeHash),
		SourceCode:          textValue(row.SourceCode),
		SourceCodeHash:      bytesToHash(row.SourceCodeHash),
		SourceCodeFetchedAt: timeValue(row.SourceCodeFetchedAt),
		SourceCodeOrigin:    textValue(row.SourceCodeOrigin),
		CreatedAt:           timeValue(row.CreatedAt),
	}
}

func mapContractCodes(rows []tokensqlc.ContractCode) []ContractCode {
	items := make([]ContractCode, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapContractCode(row))
	}
	return items
}

func mapProject(row tokensqlc.Project) (*Project, error) {
	txIndex, err := int64ToUint64("tx_index", row.TxIndex)
	if err != nil {
		return nil, err
	}
	blockNumber, err := int64ToUint64("block_number", row.BlockNumber)
	if err != nil {
		return nil, err
	}
	blockTime, err := int64ToUint64("block_time", row.BlockTime)
	if err != nil {
		return nil, err
	}
	return &Project{
		ID:          row.ID,
		ChainID:     row.ChainID,
		Contract:    bytesToAddress(row.Contract),
		Creator:     bytesToAddress(row.Creator),
		TxHash:      bytesToHash(row.TxHash),
		TxIndex:     txIndex,
		BlockNumber: blockNumber,
		BlockTime:   blockTime,
		CodeHash:    bytesToHash(row.CodeHash),
		WethPair:    bytesToAddress(row.WethPair),
		UsdtPair:    bytesToAddress(row.UsdtPair),
		CreatedAt:   timeValue(row.CreatedAt),
	}, nil
}

func mapProjects(rows []tokensqlc.Project) ([]Project, error) {
	items := make([]Project, 0, len(rows))
	for _, row := range rows {
		item, err := mapProject(row)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

func mapProjectAveData(row tokensqlc.ProjectAveDatum) *ProjectAveData {
	return &ProjectAveData{
		ProjectID:   row.ProjectID,
		AveResponse: json.RawMessage(row.AveResponse),
		FetchedAt:   timeValue(row.FetchedAt),
		CreatedAt:   timeValue(row.CreatedAt),
	}
}

func mapProjectChainStateData(row tokensqlc.ProjectChainState) *ProjectChainStateData {
	return &ProjectChainStateData{
		ProjectID:  row.ProjectID,
		ChainState: json.RawMessage(row.ChainState),
		FetchedAt:  timeValue(row.FetchedAt),
		CreatedAt:  timeValue(row.CreatedAt),
	}
}

func mapProjectRelatedWallet(row tokensqlc.ProjectRelatedWallet) *ProjectRelatedWallet {
	return &ProjectRelatedWallet{
		ProjectID: row.ProjectID,
		Wallet:    bytesToAddress(row.Wallet),
		Role:      row.Role,
		CreatedAt: timeValue(row.CreatedAt),
	}
}

func mapProjectRelatedWallets(rows []tokensqlc.ProjectRelatedWallet) []ProjectRelatedWallet {
	items := make([]ProjectRelatedWallet, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapProjectRelatedWallet(row))
	}
	return items
}

func mapProjectDataCollectionTask(row tokensqlc.ProjectDataCollectionTask) *ProjectDataCollectionTask {
	return &ProjectDataCollectionTask{
		ProjectID:     row.ProjectID,
		DataType:      row.DataType,
		Status:        row.Status,
		Attempts:      row.Attempts,
		NextAttemptAt: timeValue(row.NextAttemptAt),
		LastError:     textValue(row.LastError),
		CreatedAt:     timeValue(row.CreatedAt),
	}
}

func mapDueProjectDataCollectionTask(row tokensqlc.ListDueProjectDataCollectionTasksRow) (*ProjectDataCollectionTaskWithProject, error) {
	txIndex, err := int64ToUint64("tx_index", row.TxIndex)
	if err != nil {
		return nil, err
	}
	blockNumber, err := int64ToUint64("block_number", row.BlockNumber)
	if err != nil {
		return nil, err
	}
	blockTime, err := int64ToUint64("block_time", row.BlockTime)
	if err != nil {
		return nil, err
	}
	return &ProjectDataCollectionTaskWithProject{
		Task: ProjectDataCollectionTask{
			ProjectID:     row.ProjectID,
			DataType:      row.DataType,
			Status:        row.Status,
			Attempts:      row.Attempts,
			NextAttemptAt: timeValue(row.NextAttemptAt),
			LastError:     textValue(row.LastError),
			CreatedAt:     timeValue(row.CreatedAt),
		},
		Project: Project{
			ID:          row.ProjectID,
			ChainID:     row.ChainID,
			Contract:    bytesToAddress(row.Contract),
			Creator:     bytesToAddress(row.Creator),
			TxHash:      bytesToHash(row.TxHash),
			TxIndex:     txIndex,
			BlockNumber: blockNumber,
			BlockTime:   blockTime,
			CodeHash:    bytesToHash(row.CodeHash),
			WethPair:    bytesToAddress(row.WethPair),
			UsdtPair:    bytesToAddress(row.UsdtPair),
			CreatedAt:   timeValue(row.ProjectCreatedAt),
		},
	}, nil
}

func mapDueProjectDataCollectionTasks(rows []tokensqlc.ListDueProjectDataCollectionTasksRow) ([]ProjectDataCollectionTaskWithProject, error) {
	items := make([]ProjectDataCollectionTaskWithProject, 0, len(rows))
	for _, row := range rows {
		item, err := mapDueProjectDataCollectionTask(row)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}
