package store

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
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

func nullableInt64(value int64) pgtype.Int8 {
	if value == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: value, Valid: true}
}

func int64Value(value pgtype.Int8) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func int16ToUint8(field string, value int16) (uint8, error) {
	if value < 0 || value > math.MaxUint8 {
		return 0, fmt.Errorf("%s exceeds uint8 range", field)
	}
	return uint8(value), nil
}

func numericFromBigInt(value *big.Int) pgtype.Numeric {
	if value == nil {
		return pgtype.Numeric{Int: new(big.Int), Exp: 0, Valid: true}
	}
	return pgtype.Numeric{Int: new(big.Int).Set(value), Exp: 0, Valid: true}
}

func bigIntFromNumeric(value pgtype.Numeric) *big.Int {
	if !value.Valid || value.Int == nil {
		return new(big.Int)
	}
	if value.Exp >= 0 {
		result := new(big.Int).Set(value.Int)
		if value.Exp > 0 {
			result.Mul(result, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(value.Exp)), nil))
		}
		return result
	}
	result := new(big.Int).Set(value.Int)
	return result.Quo(result, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-value.Exp)), nil))
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
		SourceCodeFetchedAt: timeValue(row.SourceCodeFetchedAt),
		DeploymentCount:     row.DeploymentCount,
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
	decimals, err := int16ToUint8("decimals", row.Decimals)
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
		Name:        row.Name,
		Symbol:      row.Symbol,
		Decimals:    decimals,
		TotalSupply: bigIntFromNumeric(row.TotalSupply),
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

func mapProjectInitialRecipient(row tokensqlc.ProjectInitialRecipient) (*ProjectInitialRecipient, error) {
	sourceBlockNumber, err := int64ToUint64("source_block_number", row.SourceBlockNumber)
	if err != nil {
		return nil, err
	}
	return &ProjectInitialRecipient{
		ID:                row.ID,
		ProjectID:         row.ProjectID,
		Wallet:            bytesToAddress(row.Wallet),
		RatioBPS:          row.RatioBps,
		RankIndex:         row.RankIndex,
		SourceTxHash:      bytesToHash(row.SourceTxHash),
		SourceBlockNumber: sourceBlockNumber,
		CreatedAt:         timeValue(row.CreatedAt),
	}, nil
}

func mapProjectInitialRecipients(rows []tokensqlc.ProjectInitialRecipient) ([]ProjectInitialRecipient, error) {
	items := make([]ProjectInitialRecipient, 0, len(rows))
	for _, row := range rows {
		item, err := mapProjectInitialRecipient(row)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, nil
}

func mapWalletAssetState(row tokensqlc.WalletAssetState) *WalletAssetState {
	return &WalletAssetState{
		ChainID:       row.ChainID,
		Wallet:        bytesToAddress(row.Wallet),
		WethBalance:   bigIntFromNumeric(row.WethBalance),
		UsdtBalance:   bigIntFromNumeric(row.UsdtBalance),
		NativeBalance: bigIntFromNumeric(row.NativeBalance),
		UsdtValue:     bigIntFromNumeric(row.UsdtValue),
		FetchedAt:     timeValue(row.FetchedAt),
		CreatedAt:     timeValue(row.CreatedAt),
	}
}

func mapWalletAssetStates(rows []tokensqlc.WalletAssetState) []WalletAssetState {
	items := make([]WalletAssetState, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapWalletAssetState(row))
	}
	return items
}

func mapProjectSimulationResult(row tokensqlc.ProjectSimulationResult) *ProjectSimulationResult {
	return &ProjectSimulationResult{
		ProjectID:                          row.ProjectID,
		Wallet:                             bytesToAddress(row.Wallet),
		CanMintFromDeadViaTransferFrom:     row.CanMintFromDeadViaTransferFrom,
		CanMintFromZeroViaTransferFrom:     row.CanMintFromZeroViaTransferFrom,
		CanMintFromWethPairViaTransferFrom: row.CanMintFromWethPairViaTransferFrom,
		CanMintFromUsdtPairViaTransferFrom: row.CanMintFromUsdtPairViaTransferFrom,
		CanMintViaTransferToWethPair:       row.CanMintViaTransferToWethPair,
		CanMintViaTransferToUsdtPair:       row.CanMintViaTransferToUsdtPair,
		FetchedAt:                          timeValue(row.FetchedAt),
		CreatedAt:                          timeValue(row.CreatedAt),
	}
}

func mapProjectSimulationResults(rows []tokensqlc.ProjectSimulationResult) []ProjectSimulationResult {
	items := make([]ProjectSimulationResult, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapProjectSimulationResult(row))
	}
	return items
}

func mapBytecodeBlacklistEntry(row tokensqlc.BytecodeBlacklist) *BytecodeBlacklistEntry {
	return &BytecodeBlacklistEntry{
		CodeHash:       bytesToHash(row.CodeHash),
		Note:           textValue(row.Note),
		SourceChainID:  int64Value(row.SourceChainID),
		SourceContract: bytesToAddress(row.SourceContract),
		CreatedAt:      timeValue(row.CreatedAt),
	}
}

func mapBytecodeBlacklistEntries(rows []tokensqlc.BytecodeBlacklist) []BytecodeBlacklistEntry {
	items := make([]BytecodeBlacklistEntry, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapBytecodeBlacklistEntry(row))
	}
	return items
}

func mapWalletBlacklistEntry(row tokensqlc.WalletBlacklist) *WalletBlacklistEntry {
	return &WalletBlacklistEntry{
		Wallet:    bytesToAddress(row.Wallet),
		Note:      textValue(row.Note),
		CreatedAt: timeValue(row.CreatedAt),
	}
}

func mapWalletBlacklistEntries(rows []tokensqlc.WalletBlacklist) []WalletBlacklistEntry {
	items := make([]WalletBlacklistEntry, 0, len(rows))
	for _, row := range rows {
		items = append(items, *mapWalletBlacklistEntry(row))
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
	decimals, err := int16ToUint8("decimals", row.Decimals)
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
			Name:        row.Name,
			Symbol:      row.Symbol,
			Decimals:    decimals,
			TotalSupply: bigIntFromNumeric(row.TotalSupply),
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
