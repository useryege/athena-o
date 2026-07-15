package store

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/useryege/athena/internal/token/domain"
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

func uuidParam(v uuid.UUID) pgtype.UUID {
	if v == uuid.Nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: [16]byte(v), Valid: true}
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

func mapChainCheckpointRow(row tokensqlc.GetChainIngestCheckpointRow) (*domain.ChainIngestCheckpoint, error) {
	n, e := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if e != nil {
		return nil, e
	}
	return &domain.ChainIngestCheckpoint{ChainID: row.ChainID, ChainName: row.ChainName, Enabled: row.Enabled, CursorBlockNumber: n, Status: domain.ChainIngestStatus(row.Status), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}, nil
}
func mapChainCheckpointListRow(row tokensqlc.ListChainIngestCheckpointsRow) (*domain.ChainIngestCheckpoint, error) {
	return mapChainCheckpointRow(tokensqlc.GetChainIngestCheckpointRow(row))
}
func mapChainCheckpoint(row tokensqlc.ChainIngestCheckpoint) (*domain.ChainIngestCheckpoint, error) {
	n, e := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if e != nil {
		return nil, e
	}
	return &domain.ChainIngestCheckpoint{ChainID: row.ChainID, CursorBlockNumber: n, Status: domain.ChainIngestStatus(row.Status), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}, nil
}

func mapProjectCandidate(row tokensqlc.ProjectCandidate) (*domain.ProjectCandidate, error) {
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
	lockToken := uuid.Nil
	if row.ValidationLockToken.Valid {
		lockToken = uuid.UUID(row.ValidationLockToken.Bytes)
	}
	return &domain.ProjectCandidate{ID: row.ID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract), TxSender: bytesToAddress(row.TxSender), TxHash: bytesToHash(row.TxHash), TxIndex: tx, BlockNumber: bn, BlockTime: bt, Status: domain.ProjectCandidateStatus(row.Status), ValidationLockToken: lockToken, ValidationLockedAt: timeValue(row.ValidationLockedAt), ValidationLeaseExpiresAt: timeValue(row.ValidationLeaseExpiresAt), CreatedAt: timeValue(row.CreatedAt)}, nil
}
func mapProjectCandidates(rows []tokensqlc.ProjectCandidate) ([]domain.ProjectCandidate, error) {
	out := make([]domain.ProjectCandidate, 0, len(rows))
	for _, r := range rows {
		v, e := mapProjectCandidate(r)
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}
func mapContractCode(row tokensqlc.ContractCode) *domain.ContractCode {
	return &domain.ContractCode{CodeHash: bytesToHash(row.CodeHash), SourceCode: textValue(row.SourceCode), SourceCodeFetchedAt: timeValue(row.SourceCodeFetchedAt), DeploymentCount: row.DeploymentCount, CreatedAt: timeValue(row.CreatedAt)}
}
func mapContractCodes(rows []tokensqlc.ContractCode) []domain.ContractCode {
	out := make([]domain.ContractCode, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapContractCode(r))
	}
	return out
}
func mapProject(row tokensqlc.Project) (*domain.Project, error) {
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
	return &domain.Project{ID: row.ID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract), TxSender: bytesToAddress(row.TxSender), TxHash: bytesToHash(row.TxHash), TxIndex: tx, BlockNumber: bn, BlockTime: bt, CodeHash: bytesToHash(row.CodeHash), Name: row.Name, Symbol: row.Symbol, Decimals: d, TotalSupply: bigIntFromNumeric(row.TotalSupply), WethPair: bytesToAddress(row.WethPair), UsdtPair: bytesToAddress(row.UsdtPair), CreatedAt: timeValue(row.CreatedAt)}, nil
}
func mapProjects(rows []tokensqlc.Project) ([]domain.Project, error) {
	out := make([]domain.Project, 0, len(rows))
	for _, r := range rows {
		v, e := mapProject(r)
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}
func mapProjectRelatedWallet(row tokensqlc.ProjectRelatedWallet) *domain.ProjectRelatedWallet {
	return &domain.ProjectRelatedWallet{ProjectID: row.ProjectID, Wallet: bytesToAddress(row.Wallet), Role: domain.RelatedWalletRole(row.Role), CreatedAt: timeValue(row.CreatedAt)}
}
func mapProjectRelatedWallets(rows []tokensqlc.ProjectRelatedWallet) []domain.ProjectRelatedWallet {
	out := make([]domain.ProjectRelatedWallet, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapProjectRelatedWallet(r))
	}
	return out
}
func mapProjectInitialRecipient(row tokensqlc.ProjectInitialRecipient) (*domain.ProjectInitialRecipient, error) {
	bn, e := int64ToUint64("source_block_number", row.SourceBlockNumber)
	if e != nil {
		return nil, e
	}
	return &domain.ProjectInitialRecipient{ID: row.ID, ProjectID: row.ProjectID, Wallet: bytesToAddress(row.Wallet), RatioBPS: row.RatioBps, RankIndex: row.RankIndex, SourceTxHash: bytesToHash(row.SourceTxHash), SourceBlockNumber: bn, CreatedAt: timeValue(row.CreatedAt)}, nil
}
func mapProjectInitialRecipients(rows []tokensqlc.ProjectInitialRecipient) ([]domain.ProjectInitialRecipient, error) {
	out := make([]domain.ProjectInitialRecipient, 0, len(rows))
	for _, r := range rows {
		v, e := mapProjectInitialRecipient(r)
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}

func mapContractCodeBlocklistEntry(row tokensqlc.ContractCodeBlocklist) *domain.ContractCodeBlocklistEntry {
	return &domain.ContractCodeBlocklistEntry{CodeHash: bytesToHash(row.CodeHash), Note: textValue(row.Note), SourceChainID: int64Value(row.SourceChainID), SourceContract: bytesToAddress(row.SourceContract), CreatedAt: timeValue(row.CreatedAt)}
}
func mapContractCodeBlocklistEntries(rows []tokensqlc.ContractCodeBlocklist) []domain.ContractCodeBlocklistEntry {
	out := make([]domain.ContractCodeBlocklistEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapContractCodeBlocklistEntry(r))
	}
	return out
}
func mapWalletBlocklistEntry(row tokensqlc.WalletBlocklist) *domain.WalletBlocklistEntry {
	return &domain.WalletBlocklistEntry{Wallet: bytesToAddress(row.Wallet), Note: textValue(row.Note), CreatedAt: timeValue(row.CreatedAt)}
}
func mapWalletBlocklistEntries(rows []tokensqlc.WalletBlocklist) []domain.WalletBlocklistEntry {
	out := make([]domain.WalletBlocklistEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapWalletBlocklistEntry(r))
	}
	return out
}

func mapProjectDataCollectionSchedule(row tokensqlc.ProjectDataCollectionSchedule) domain.ProjectDataCollectionSchedule {
	return domain.ProjectDataCollectionSchedule{ProjectID: row.ProjectID, DataType: domain.DataCollectionType(row.DataType), Status: domain.DataCollectionScheduleStatus(row.Status), RefreshInterval: time.Duration(row.RefreshIntervalSeconds) * time.Second, NextRunAt: timeValue(row.NextRunAt), LatestTaskRevision: row.LatestTaskRevision, ConsecutiveFailures: row.ConsecutiveFailures, LastError: textValue(row.LastError), LastCheckedAt: timeValue(row.LastCheckedAt), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}
}
func mapProjectDataCollectionTask(row tokensqlc.ProjectDataCollectionTask) domain.ProjectDataCollectionTask {
	return domain.ProjectDataCollectionTask{ID: row.ID, ProjectID: row.ProjectID, DataType: domain.DataCollectionType(row.DataType), Status: domain.TaskStatus(row.Status), Revision: row.Revision, Attempts: row.Attempts, AvailableAt: timeValue(row.AvailableAt), LockedAt: timeValue(row.LockedAt), LeaseExpiresAt: timeValue(row.LeaseExpiresAt), LastError: textValue(row.LastError), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}
}
func mapProjectObservation(row tokensqlc.ProjectObservation) (domain.ProjectObservation, error) {
	bn, e := uint64PointerFromInt64("block_number", row.BlockNumber)
	return domain.ProjectObservation{ID: row.ID, ProjectID: row.ProjectID, DataType: domain.DataCollectionType(row.DataType), SchemaVersion: row.SchemaVersion, ContentHash: bytesToHash(row.ContentHash), Payload: json.RawMessage(row.Payload), BlockNumber: bn, ObservedAt: timeValue(row.ObservedAt), CreatedAt: timeValue(row.CreatedAt)}, e
}
func mapCurrentProjectObservation(row tokensqlc.GetCurrentProjectObservationRow) (domain.ProjectObservation, error) {
	bn, e := uint64PointerFromInt64("block_number", row.BlockNumber)
	return domain.ProjectObservation{ID: row.ID, ProjectID: row.ProjectID, DataType: domain.DataCollectionType(row.DataType), SchemaVersion: row.SchemaVersion, ContentHash: bytesToHash(row.ContentHash), Payload: json.RawMessage(row.Payload), BlockNumber: bn, ObservedAt: timeValue(row.ObservedAt), LastCheckedAt: timeValue(row.LastCheckedAt), CreatedAt: timeValue(row.CreatedAt)}, e
}
func mapCurrentProjectObservations(rows []tokensqlc.ListCurrentProjectObservationsRow) ([]domain.ProjectObservation, error) {
	out := make([]domain.ProjectObservation, 0, len(rows))
	for _, r := range rows {
		v, e := mapCurrentProjectObservation(tokensqlc.GetCurrentProjectObservationRow(r))
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, nil
}

func mapProjectResearchState(row tokensqlc.ListProjectResearchStatesRow) domain.ProjectResearchState {
	return domain.ProjectResearchState{ProjectID: row.ProjectID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract), Status: domain.ProjectResearchStatus(row.Status), EvidenceRevision: row.EvidenceRevision, CurrentReportRevision: int64Value(row.CurrentReportRevision), CurrentSelectionID: int64Value(row.CurrentSelectionID), CurrentSelectionOutcome: domain.SelectionOutcome(row.CurrentSelectionOutcome), LastEvaluatedReportRevision: int64Value(row.LastEvaluatedReportRevision), LastEvaluatedAt: timeValue(row.LastEvaluatedAt), ExpiresAt: timeValue(row.ExpiresAt), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}
}
func mapProjectReportRevision(row tokensqlc.ProjectReportRevision) (domain.ProjectReportRevision, error) {
	bn, e := uint64PointerFromInt64("observed_block_number", row.ObservedBlockNumber)
	if e != nil {
		return domain.ProjectReportRevision{}, e
	}
	wts, e := uint64PointerFromInt64("weth_pair_last_swap_timestamp", row.WethPairLastSwapTimestamp)
	if e != nil {
		return domain.ProjectReportRevision{}, e
	}
	uts, e := uint64PointerFromInt64("usdt_pair_last_swap_timestamp", row.UsdtPairLastSwapTimestamp)
	if e != nil {
		return domain.ProjectReportRevision{}, e
	}
	var evidence []domain.EvidenceReference
	if e = json.Unmarshal(row.Evidence, &evidence); e != nil {
		return domain.ProjectReportRevision{}, fmt.Errorf("decode report evidence: %w", e)
	}
	var report domain.ResearchReportV1
	if e = json.Unmarshal(row.Report, &report); e != nil {
		return domain.ProjectReportRevision{}, fmt.Errorf("decode report: %w", e)
	}
	risk := report.RiskSummary
	if !equalBoolPointers(risk.WethPairIsCreated, boolPointer(row.WethPairIsCreated)) ||
		!equalBoolPointers(risk.WethPairIsRemoveLiquidity, boolPointer(row.WethPairIsRemoveLiquidity)) ||
		!equalBoolPointers(risk.WethPairIsMint, boolPointer(row.WethPairIsMint)) ||
		!equalBigIntPointers(risk.WethPairQuoteUsdtValueInt, bigIntPointerFromNumeric(row.WethPairQuoteUsdtValueInt)) ||
		!equalUint64Pointers(risk.WethPairLastSwapTimestamp, wts) ||
		!equalBoolPointers(risk.UsdtPairIsCreated, boolPointer(row.UsdtPairIsCreated)) ||
		!equalBoolPointers(risk.UsdtPairIsRemoveLiquidity, boolPointer(row.UsdtPairIsRemoveLiquidity)) ||
		!equalBoolPointers(risk.UsdtPairIsMint, boolPointer(row.UsdtPairIsMint)) ||
		!equalBigIntPointers(risk.UsdtPairQuoteUsdtValueInt, bigIntPointerFromNumeric(row.UsdtPairQuoteUsdtValueInt)) ||
		!equalUint64Pointers(risk.UsdtPairLastSwapTimestamp, uts) {
		return domain.ProjectReportRevision{}, fmt.Errorf("project report revision %d risk projection does not match report JSON", row.ID)
	}
	return domain.ProjectReportRevision{ID: row.ID, ProjectID: row.ProjectID, Revision: row.Revision, SchemaVersion: row.SchemaVersion, ContentHash: bytesToHash(row.ContentHash), CompletenessStatus: row.CompletenessStatus, Evidence: evidence, Report: report, ObservedBlockNumber: bn, BuiltAt: timeValue(row.BuiltAt), CreatedAt: timeValue(row.CreatedAt)}, nil
}

func equalBoolPointers(left, right *bool) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}

func equalUint64Pointers(left, right *uint64) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && *left == *right)
}

func equalBigIntPointers(left, right *big.Int) bool {
	return (left == nil && right == nil) || (left != nil && right != nil && left.Cmp(right) == 0)
}
func mapProjectSelection(row tokensqlc.ProjectSelection) domain.ProjectSelection {
	return domain.ProjectSelection{ID: row.ID, ProjectID: row.ProjectID, Outcome: domain.SelectionOutcome(row.Outcome), StrategyKey: row.StrategyKey, StrategyVersion: row.StrategyVersion, ReportRevision: row.ReportRevision, ReasonCodes: row.ReasonCodes, ReasonDetail: row.ReasonDetail, DecidedAt: timeValue(row.DecidedAt), CreatedAt: timeValue(row.CreatedAt)}
}
