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

func bytesToHash(v []byte) common.Hash {
	if len(v) == 0 {
		return common.Hash{}
	}
	return common.BytesToHash(v)
}
func bytesToAddress(v []byte) common.Address {
	if len(v) == 0 {
		return common.Address{}
	}
	return common.BytesToAddress(v)
}
func optionalHashBytes(v common.Hash) []byte {
	if v == (common.Hash{}) {
		return nil
	}
	return v.Bytes()
}
func optionalAddressBytes(v common.Address) []byte {
	if v == (common.Address{}) {
		return nil
	}
	return v.Bytes()
}
func nullableText(v string) pgtype.Text { return pgtype.Text{String: v, Valid: v != ""} }
func textValue(v pgtype.Text) string {
	if !v.Valid {
		return ""
	}
	return v.String
}
func nullableTime(v time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: v, Valid: !v.IsZero()}
}
func timeValue(v pgtype.Timestamptz) time.Time {
	if !v.Valid {
		return time.Time{}
	}
	return v.Time
}
func nullableInt64(v int64) pgtype.Int8 { return pgtype.Int8{Int64: v, Valid: v != 0} }
func int64Value(v pgtype.Int8) int64 {
	if !v.Valid {
		return 0
	}
	return v.Int64
}
func nullableBool(v *bool) pgtype.Bool {
	if v == nil {
		return pgtype.Bool{}
	}
	return pgtype.Bool{Bool: *v, Valid: true}
}
func boolPointer(v pgtype.Bool) *bool {
	if !v.Valid {
		return nil
	}
	r := v.Bool
	return &r
}

func numericFromBigInt(v *big.Int) pgtype.Numeric {
	if v == nil {
		v = new(big.Int)
	}
	return pgtype.Numeric{Int: new(big.Int).Set(v), Exp: 0, Valid: true}
}
func nullableNumericFromBigInt(v *big.Int) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{}
	}
	return numericFromBigInt(v)
}
func bigIntFromNumeric(v pgtype.Numeric) *big.Int {
	if !v.Valid || v.Int == nil {
		return new(big.Int)
	}
	r := new(big.Int).Set(v.Int)
	if v.Exp > 0 {
		return r.Mul(r, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(v.Exp)), nil))
	}
	if v.Exp < 0 {
		return r.Quo(r, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-v.Exp)), nil))
	}
	return r
}
func bigIntPointerFromNumeric(v pgtype.Numeric) *big.Int {
	if !v.Valid {
		return nil
	}
	return bigIntFromNumeric(v)
}

func uint64ToInt64(field string, v uint64) (int64, error) {
	if v > math.MaxInt64 {
		return 0, fmt.Errorf("%s exceeds int64 max", field)
	}
	return int64(v), nil
}
func int64ToUint64(field string, v int64) (uint64, error) {
	if v < 0 {
		return 0, fmt.Errorf("%s is negative", field)
	}
	return uint64(v), nil
}
func int16ToUint8(field string, v int16) (uint8, error) {
	if v < 0 || v > math.MaxUint8 {
		return 0, fmt.Errorf("%s exceeds uint8 range", field)
	}
	return uint8(v), nil
}
func nullableUint64(v *uint64) (pgtype.Int8, error) {
	if v == nil {
		return pgtype.Int8{}, nil
	}
	n, e := uint64ToInt64("uint64", *v)
	return pgtype.Int8{Int64: n, Valid: e == nil}, e
}
func uint64PointerFromInt64(field string, v pgtype.Int8) (*uint64, error) {
	if !v.Valid {
		return nil, nil
	}
	n, e := int64ToUint64(field, v.Int64)
	return &n, e
}
func normalizePage(page, size int32) (int32, int32, int32) {
	if page < 1 {
		page = 1
	}
	if size <= 0 {
		size = defaultPageSize
	}
	if size > maxPageSize {
		size = maxPageSize
	}
	return page, size, (page - 1) * size
}

func mapChainCheckpointRow(row tokensqlc.GetChainIngestCheckpointRow) (*ChainIngestCheckpoint, error) {
	n, e := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if e != nil {
		return nil, e
	}
	return &ChainIngestCheckpoint{ChainID: row.ChainID, ChainName: row.ChainName, Enabled: row.Enabled, CursorBlockNumber: n, Status: row.Status, CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}, nil
}
func mapChainCheckpointListRow(row tokensqlc.ListChainIngestCheckpointsRow) (*ChainIngestCheckpoint, error) {
	return mapChainCheckpointRow(tokensqlc.GetChainIngestCheckpointRow(row))
}
func mapChainCheckpoint(row tokensqlc.ChainIngestCheckpoint) (*ChainIngestCheckpoint, error) {
	n, e := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if e != nil {
		return nil, e
	}
	return &ChainIngestCheckpoint{ChainID: row.ChainID, CursorBlockNumber: n, Status: row.Status, CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}, nil
}

func mapProjectCandidate(row tokensqlc.ProjectCandidate) (*ProjectCandidate, error) {
	tx, e := int64ToUint64("tx_index", row.TxIndex)
	if e != nil {
		return nil, e
	}
	bn, e := int64ToUint64("block_number", row.BlockNumber)
	if e != nil {
		return nil, e
	}
	bt, e := int64ToUint64("block_time", row.BlockTime)
	if e != nil {
		return nil, e
	}
	return &ProjectCandidate{ID: row.ID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract), TxSender: bytesToAddress(row.TxSender), TxHash: bytesToHash(row.TxHash), TxIndex: tx, BlockNumber: bn, BlockTime: bt, Status: row.Status, CreatedAt: timeValue(row.CreatedAt)}, nil
}
func mapProjectCandidates(rows []tokensqlc.ProjectCandidate) ([]ProjectCandidate, error) {
	out := make([]ProjectCandidate, 0, len(rows))
	for _, r := range rows {
		v, e := mapProjectCandidate(r)
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}
func mapContractCode(row tokensqlc.ContractCode) *ContractCode {
	return &ContractCode{CodeHash: bytesToHash(row.CodeHash), SourceCode: textValue(row.SourceCode), SourceCodeFetchedAt: timeValue(row.SourceCodeFetchedAt), DeploymentCount: row.DeploymentCount, CreatedAt: timeValue(row.CreatedAt)}
}
func mapContractCodes(rows []tokensqlc.ContractCode) []ContractCode {
	out := make([]ContractCode, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapContractCode(r))
	}
	return out
}
func mapProject(row tokensqlc.Project) (*Project, error) {
	tx, e := int64ToUint64("tx_index", row.TxIndex)
	if e != nil {
		return nil, e
	}
	bn, e := int64ToUint64("block_number", row.BlockNumber)
	if e != nil {
		return nil, e
	}
	bt, e := int64ToUint64("block_time", row.BlockTime)
	if e != nil {
		return nil, e
	}
	d, e := int16ToUint8("decimals", row.Decimals)
	if e != nil {
		return nil, e
	}
	return &Project{ID: row.ID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract), TxSender: bytesToAddress(row.TxSender), TxHash: bytesToHash(row.TxHash), TxIndex: tx, BlockNumber: bn, BlockTime: bt, CodeHash: bytesToHash(row.CodeHash), Name: row.Name, Symbol: row.Symbol, Decimals: d, TotalSupply: bigIntFromNumeric(row.TotalSupply), WethPair: bytesToAddress(row.WethPair), UsdtPair: bytesToAddress(row.UsdtPair), CreatedAt: timeValue(row.CreatedAt)}, nil
}
func mapProjects(rows []tokensqlc.Project) ([]Project, error) {
	out := make([]Project, 0, len(rows))
	for _, r := range rows {
		v, e := mapProject(r)
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}
func mapProjectRelatedWallet(row tokensqlc.ProjectRelatedWallet) *ProjectRelatedWallet {
	return &ProjectRelatedWallet{ProjectID: row.ProjectID, Wallet: bytesToAddress(row.Wallet), Role: row.Role, CreatedAt: timeValue(row.CreatedAt)}
}
func mapProjectRelatedWallets(rows []tokensqlc.ProjectRelatedWallet) []ProjectRelatedWallet {
	out := make([]ProjectRelatedWallet, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapProjectRelatedWallet(r))
	}
	return out
}
func mapProjectInitialRecipient(row tokensqlc.ProjectInitialRecipient) (*ProjectInitialRecipient, error) {
	bn, e := int64ToUint64("source_block_number", row.SourceBlockNumber)
	if e != nil {
		return nil, e
	}
	return &ProjectInitialRecipient{ID: row.ID, ProjectID: row.ProjectID, Wallet: bytesToAddress(row.Wallet), RatioBPS: row.RatioBps, RankIndex: row.RankIndex, SourceTxHash: bytesToHash(row.SourceTxHash), SourceBlockNumber: bn, CreatedAt: timeValue(row.CreatedAt)}, nil
}
func mapProjectInitialRecipients(rows []tokensqlc.ProjectInitialRecipient) ([]ProjectInitialRecipient, error) {
	out := make([]ProjectInitialRecipient, 0, len(rows))
	for _, r := range rows {
		v, e := mapProjectInitialRecipient(r)
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}

func mapContractCodeBlocklistEntry(row tokensqlc.ContractCodeBlocklist) *ContractCodeBlocklistEntry {
	return &ContractCodeBlocklistEntry{CodeHash: bytesToHash(row.CodeHash), Note: textValue(row.Note), SourceChainID: int64Value(row.SourceChainID), SourceContract: bytesToAddress(row.SourceContract), CreatedAt: timeValue(row.CreatedAt)}
}
func mapContractCodeBlocklistEntries(rows []tokensqlc.ContractCodeBlocklist) []ContractCodeBlocklistEntry {
	out := make([]ContractCodeBlocklistEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapContractCodeBlocklistEntry(r))
	}
	return out
}
func mapWalletBlocklistEntry(row tokensqlc.WalletBlocklist) *WalletBlocklistEntry {
	return &WalletBlocklistEntry{Wallet: bytesToAddress(row.Wallet), Note: textValue(row.Note), CreatedAt: timeValue(row.CreatedAt)}
}
func mapWalletBlocklistEntries(rows []tokensqlc.WalletBlocklist) []WalletBlocklistEntry {
	out := make([]WalletBlocklistEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapWalletBlocklistEntry(r))
	}
	return out
}

func mapProjectDataCollectionSchedule(row tokensqlc.ProjectDataCollectionSchedule) ProjectDataCollectionSchedule {
	return ProjectDataCollectionSchedule{ProjectID: row.ProjectID, DataType: row.DataType, Status: row.Status, RefreshInterval: time.Duration(row.RefreshIntervalSeconds) * time.Second, NextRunAt: timeValue(row.NextRunAt), LatestTaskRevision: row.LatestTaskRevision, ConsecutiveFailures: row.ConsecutiveFailures, LastError: textValue(row.LastError), LastCheckedAt: timeValue(row.LastCheckedAt), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}
}
func mapProjectDataCollectionTask(row tokensqlc.ProjectDataCollectionTask) ProjectDataCollectionTask {
	return ProjectDataCollectionTask{ID: row.ID, ProjectID: row.ProjectID, DataType: row.DataType, Status: row.Status, Revision: row.Revision, Attempts: row.Attempts, AvailableAt: timeValue(row.AvailableAt), LockedAt: timeValue(row.LockedAt), LeaseExpiresAt: timeValue(row.LeaseExpiresAt), LastError: textValue(row.LastError), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}
}
func mapProjectObservation(row tokensqlc.ProjectObservation) (ProjectObservation, error) {
	bn, e := uint64PointerFromInt64("block_number", row.BlockNumber)
	return ProjectObservation{ID: row.ID, ProjectID: row.ProjectID, DataType: row.DataType, ContentHash: bytesToHash(row.ContentHash), Payload: json.RawMessage(row.Payload), BlockNumber: bn, ObservedAt: timeValue(row.ObservedAt), CreatedAt: timeValue(row.CreatedAt)}, e
}
func mapCurrentProjectObservation(row tokensqlc.GetCurrentProjectObservationRow) (ProjectObservation, error) {
	bn, e := uint64PointerFromInt64("block_number", row.BlockNumber)
	return ProjectObservation{ID: row.ID, ProjectID: row.ProjectID, DataType: row.DataType, ContentHash: bytesToHash(row.ContentHash), Payload: json.RawMessage(row.Payload), BlockNumber: bn, ObservedAt: timeValue(row.ObservedAt), LastCheckedAt: timeValue(row.LastCheckedAt), CreatedAt: timeValue(row.CreatedAt)}, e
}
func mapCurrentProjectObservations(rows []tokensqlc.ListCurrentProjectObservationsRow) ([]ProjectObservation, error) {
	out := make([]ProjectObservation, 0, len(rows))
	for _, r := range rows {
		v, e := mapCurrentProjectObservation(tokensqlc.GetCurrentProjectObservationRow(r))
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, nil
}

func mapProjectResearchState(row tokensqlc.ListProjectResearchStatesRow) ProjectResearchState {
	return ProjectResearchState{ProjectID: row.ProjectID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract), Status: row.Status, EvidenceRevision: row.EvidenceRevision, CurrentReportRevision: int64Value(row.CurrentReportRevision), CurrentSelectionID: int64Value(row.CurrentSelectionID), CurrentSelectionOutcome: row.CurrentSelectionOutcome, LastEvaluatedReportRevision: int64Value(row.LastEvaluatedReportRevision), LastEvaluatedAt: timeValue(row.LastEvaluatedAt), ExpiresAt: timeValue(row.ExpiresAt), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}
}
func mapProjectReportRevision(row tokensqlc.ProjectReportRevision) (ProjectReportRevision, error) {
	bn, e := uint64PointerFromInt64("observed_block_number", row.ObservedBlockNumber)
	if e != nil {
		return ProjectReportRevision{}, e
	}
	wts, e := uint64PointerFromInt64("weth_pair_last_swap_timestamp", row.WethPairLastSwapTimestamp)
	if e != nil {
		return ProjectReportRevision{}, e
	}
	uts, e := uint64PointerFromInt64("usdt_pair_last_swap_timestamp", row.UsdtPairLastSwapTimestamp)
	if e != nil {
		return ProjectReportRevision{}, e
	}
	return ProjectReportRevision{ID: row.ID, ProjectID: row.ProjectID, Revision: row.Revision, ContentHash: bytesToHash(row.ContentHash), CompletenessStatus: row.CompletenessStatus, Evidence: json.RawMessage(row.Evidence), Report: json.RawMessage(row.Report), ObservedBlockNumber: bn, WethPairIsCreated: boolPointer(row.WethPairIsCreated), WethPairIsRemoveLiquidity: boolPointer(row.WethPairIsRemoveLiquidity), WethPairIsMint: boolPointer(row.WethPairIsMint), WethPairQuoteUsdtValueInt: bigIntPointerFromNumeric(row.WethPairQuoteUsdtValueInt), WethPairLastSwapTimestamp: wts, UsdtPairIsCreated: boolPointer(row.UsdtPairIsCreated), UsdtPairIsRemoveLiquidity: boolPointer(row.UsdtPairIsRemoveLiquidity), UsdtPairIsMint: boolPointer(row.UsdtPairIsMint), UsdtPairQuoteUsdtValueInt: bigIntPointerFromNumeric(row.UsdtPairQuoteUsdtValueInt), UsdtPairLastSwapTimestamp: uts, BuiltAt: timeValue(row.BuiltAt), CreatedAt: timeValue(row.CreatedAt)}, nil
}
func mapProjectSelection(row tokensqlc.ProjectSelection) ProjectSelection {
	return ProjectSelection{ID: row.ID, ProjectID: row.ProjectID, Outcome: row.Outcome, StrategyKey: row.StrategyKey, StrategyVersion: row.StrategyVersion, ReportRevision: row.ReportRevision, ReasonCodes: row.ReasonCodes, ReasonDetail: row.ReasonDetail, DecidedAt: timeValue(row.DecidedAt), CreatedAt: timeValue(row.CreatedAt)}
}
