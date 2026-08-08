package postgres

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/policy"
	"github.com/useryege/athena/internal/token/reporting"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/selection"
	"github.com/useryege/athena/internal/token/shared"
)

const (
	defaultPageSize = int32(20)
	maxPageSize     = int32(200)
)

func bytesToHash(v []byte) shared.Hash {
	if len(v) == 0 {
		return shared.Hash{}
	}
	return shared.BytesToHash(v)
}

func bytesToAddress(v []byte) shared.Address {
	if len(v) == 0 {
		return shared.Address{}
	}
	return shared.BytesToAddress(v)
}
func optionalHashBytes(v shared.Hash) []byte {
	if v == (shared.Hash{}) {
		return nil
	}
	return v.Bytes()
}
func optionalAddressBytes(v shared.Address) []byte {
	if v == (shared.Address{}) {
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
func exactBigIntPointerFromNumeric(field string, v pgtype.Numeric) (*big.Int, error) {
	if !v.Valid {
		return nil, nil
	}
	if v.NaN || v.InfinityModifier != pgtype.Finite || v.Int == nil {
		return nil, fmt.Errorf("%s is not a finite numeric value", field)
	}
	r := new(big.Int).Set(v.Int)
	if v.Exp > 0 {
		r.Mul(r, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(v.Exp)), nil))
	} else if v.Exp < 0 {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-v.Exp)), nil)
		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(r, divisor, remainder)
		if remainder.Sign() != 0 {
			return nil, fmt.Errorf("%s is not an integer", field)
		}
		r = quotient
	}
	return r, nil
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

func mapChainCheckpointRow(row tokensqlc.GetChainProcessingCheckpointRow) (*discovery.ChainProcessingCheckpoint, error) {
	n, e := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if e != nil {
		return nil, e
	}
	return &discovery.ChainProcessingCheckpoint{ChainID: row.ChainID, ChainName: row.ChainName, Enabled: row.Enabled, CursorBlockNumber: n, Status: discovery.ChainProcessingStatus(row.Status), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}, nil
}
func mapChainCheckpointListRow(row tokensqlc.ListChainProcessingCheckpointsRow) (*discovery.ChainProcessingCheckpoint, error) {
	return mapChainCheckpointRow(tokensqlc.GetChainProcessingCheckpointRow(row))
}
func mapChainCheckpoint(row tokensqlc.ChainProcessingCheckpoint) (*discovery.ChainProcessingCheckpoint, error) {
	n, e := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if e != nil {
		return nil, e
	}
	return &discovery.ChainProcessingCheckpoint{ChainID: row.ChainID, CursorBlockNumber: n, Status: discovery.ChainProcessingStatus(row.Status), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}, nil
}
func mapContractCode(row tokensqlc.ContractCode) *catalog.ContractCode {
	return &catalog.ContractCode{CodeHash: bytesToHash(row.CodeHash), SourceCode: textValue(row.SourceCode), SourceCodeFetchedAt: timeValue(row.SourceCodeFetchedAt), DeploymentCount: row.DeploymentCount, CreatedAt: timeValue(row.CreatedAt)}
}
func mapContractCodes(rows []tokensqlc.ContractCode) []catalog.ContractCode {
	out := make([]catalog.ContractCode, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapContractCode(r))
	}
	return out
}
func mapProject(row tokensqlc.Project) (*catalog.Project, error) {
	tx, e := int64ToUint64("tx_index", row.TxIndex)
	if e != nil {
		return nil, e
	}
	deploymentNonce, e := int64ToUint64("deployment_nonce", row.DeploymentNonce)
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
	return &catalog.Project{ID: row.ID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract), TxSender: bytesToAddress(row.TxSender), TxHash: bytesToHash(row.TxHash), TxIndex: tx, DeploymentNonce: deploymentNonce, BlockNumber: bn, BlockTime: bt, CodeHash: bytesToHash(row.CodeHash), Name: row.Name, Symbol: row.Symbol, Decimals: d, TotalSupply: bigIntFromNumeric(row.TotalSupply), WethPair: bytesToAddress(row.WethPair), UsdtPair: bytesToAddress(row.UsdtPair), CreatedAt: timeValue(row.CreatedAt)}, nil
}
func mapProjects(rows []tokensqlc.Project) ([]catalog.Project, error) {
	out := make([]catalog.Project, 0, len(rows))
	for _, r := range rows {
		v, e := mapProject(r)
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}
func mapProjectRelatedWallet(row tokensqlc.ProjectRelatedWallet) *catalog.ProjectRelatedWallet {
	return &catalog.ProjectRelatedWallet{ProjectID: row.ProjectID, Wallet: bytesToAddress(row.Wallet), Role: catalog.RelatedWalletRole(row.Role), CreatedAt: timeValue(row.CreatedAt)}
}
func mapProjectRelatedWallets(rows []tokensqlc.ProjectRelatedWallet) []catalog.ProjectRelatedWallet {
	out := make([]catalog.ProjectRelatedWallet, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapProjectRelatedWallet(r))
	}
	return out
}
func mapProjectInitialRecipient(row tokensqlc.ProjectInitialRecipient) (*catalog.ProjectInitialRecipient, error) {
	bn, e := int64ToUint64("source_block_number", row.SourceBlockNumber)
	if e != nil {
		return nil, e
	}
	return &catalog.ProjectInitialRecipient{ID: row.ID, ProjectID: row.ProjectID, Wallet: bytesToAddress(row.Wallet), RatioBPS: row.RatioBps, RankIndex: row.RankIndex, SourceTxHash: bytesToHash(row.SourceTxHash), SourceBlockNumber: bn, CreatedAt: timeValue(row.CreatedAt)}, nil
}
func mapProjectInitialRecipients(rows []tokensqlc.ProjectInitialRecipient) ([]catalog.ProjectInitialRecipient, error) {
	out := make([]catalog.ProjectInitialRecipient, 0, len(rows))
	for _, r := range rows {
		v, e := mapProjectInitialRecipient(r)
		if e != nil {
			return nil, e
		}
		out = append(out, *v)
	}
	return out, nil
}

func mapContractCodeBlocklistEntry(row tokensqlc.ContractCodeBlocklist) *policy.ContractCodeBlocklistEntry {
	return &policy.ContractCodeBlocklistEntry{CodeHash: bytesToHash(row.CodeHash), Note: textValue(row.Note), SourceChainID: int64Value(row.SourceChainID), SourceContract: bytesToAddress(row.SourceContract), CreatedAt: timeValue(row.CreatedAt)}
}
func mapContractCodeBlocklistEntries(rows []tokensqlc.ContractCodeBlocklist) []policy.ContractCodeBlocklistEntry {
	out := make([]policy.ContractCodeBlocklistEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapContractCodeBlocklistEntry(r))
	}
	return out
}
func mapWalletBlocklistEntry(row tokensqlc.WalletBlocklist) *policy.WalletBlocklistEntry {
	return &policy.WalletBlocklistEntry{Wallet: bytesToAddress(row.Wallet), Note: textValue(row.Note), CreatedAt: timeValue(row.CreatedAt)}
}
func mapWalletBlocklistEntries(rows []tokensqlc.WalletBlocklist) []policy.WalletBlocklistEntry {
	out := make([]policy.WalletBlocklistEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, *mapWalletBlocklistEntry(r))
	}
	return out
}

func mapProjectDataCollectionSchedule(row tokensqlc.ProjectDataCollectionSchedule) research.ProjectDataCollectionSchedule {
	return research.ProjectDataCollectionSchedule{ProjectID: row.ProjectID, DataType: research.DataCollectionType(row.DataType), Status: research.DataCollectionScheduleStatus(row.Status), RetryInterval: time.Duration(row.RetryIntervalSeconds) * time.Second, NextRunAt: timeValue(row.NextRunAt), LatestTaskRevision: row.LatestTaskRevision, ConsecutiveFailures: row.ConsecutiveFailures, LastError: textValue(row.LastError), LastCheckedAt: timeValue(row.LastCheckedAt), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}
}
func mapProjectDataCollectionTask(row tokensqlc.ProjectDataCollectionTask) research.ProjectDataCollectionTask {
	return research.ProjectDataCollectionTask{ID: row.ID, ProjectID: row.ProjectID, DataType: research.DataCollectionType(row.DataType), Status: research.TaskStatus(row.Status), Revision: row.Revision, Attempts: row.Attempts, AvailableAt: timeValue(row.AvailableAt), LockedAt: timeValue(row.LockedAt), LeaseExpiresAt: timeValue(row.LeaseExpiresAt), LastError: textValue(row.LastError), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}
}
func mapProjectObservation(row tokensqlc.ProjectObservation) (research.ProjectObservation, error) {
	bn, e := uint64PointerFromInt64("block_number", row.BlockNumber)
	return research.ProjectObservation{ID: row.ID, ProjectID: row.ProjectID, DataType: research.DataCollectionType(row.DataType), SchemaVersion: row.SchemaVersion, ContentHash: bytesToHash(row.ContentHash), Payload: json.RawMessage(row.Payload), BlockNumber: bn, ObservedAt: timeValue(row.ObservedAt), CreatedAt: timeValue(row.CreatedAt)}, e
}
func mapCurrentProjectObservation(row tokensqlc.GetCurrentProjectObservationRow) (research.ProjectObservation, error) {
	bn, e := uint64PointerFromInt64("block_number", row.BlockNumber)
	return research.ProjectObservation{ID: row.ID, ProjectID: row.ProjectID, DataType: research.DataCollectionType(row.DataType), SchemaVersion: row.SchemaVersion, ContentHash: bytesToHash(row.ContentHash), Payload: json.RawMessage(row.Payload), BlockNumber: bn, ObservedAt: timeValue(row.ObservedAt), LastCheckedAt: timeValue(row.LastCheckedAt), CreatedAt: timeValue(row.CreatedAt)}, e
}
func mapCurrentProjectObservations(rows []tokensqlc.ListCurrentProjectObservationsRow) ([]research.ProjectObservation, error) {
	out := make([]research.ProjectObservation, 0, len(rows))
	for _, r := range rows {
		v, e := mapCurrentProjectObservation(tokensqlc.GetCurrentProjectObservationRow(r))
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, nil
}

func mapProjectResearchState(row tokensqlc.ListProjectResearchStatesRow) research.ProjectResearchState {
	return research.ProjectResearchState{ProjectID: row.ProjectID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract), Status: research.ProjectResearchStatus(row.Status), EvidenceRevision: row.EvidenceRevision, CurrentReportRevision: int64Value(row.CurrentReportRevision), CurrentSelectionID: int64Value(row.CurrentSelectionID), CurrentSelectionOutcome: row.CurrentSelectionOutcome, LastEvaluatedReportRevision: int64Value(row.LastEvaluatedReportRevision), LastEvaluatedAt: timeValue(row.LastEvaluatedAt), ExpiresAt: timeValue(row.ExpiresAt), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt)}
}
func mapProjectReportRevision(row tokensqlc.ProjectReportRevision) (reporting.ProjectReportRevision, error) {
	bn, e := uint64PointerFromInt64("observed_block_number", row.ObservedBlockNumber)
	if e != nil {
		return reporting.ProjectReportRevision{}, e
	}
	wts, e := uint64PointerFromInt64("weth_pair_last_swap_timestamp", row.WethPairLastSwapTimestamp)
	if e != nil {
		return reporting.ProjectReportRevision{}, e
	}
	uts, e := uint64PointerFromInt64("usdt_pair_last_swap_timestamp", row.UsdtPairLastSwapTimestamp)
	if e != nil {
		return reporting.ProjectReportRevision{}, e
	}
	wethQuoteUSDTValueInt, e := exactBigIntPointerFromNumeric("weth_pair_quote_usdt_value_int", row.WethPairQuoteUsdtValueInt)
	if e != nil {
		return reporting.ProjectReportRevision{}, e
	}
	usdtQuoteUSDTValueInt, e := exactBigIntPointerFromNumeric("usdt_pair_quote_usdt_value_int", row.UsdtPairQuoteUsdtValueInt)
	if e != nil {
		return reporting.ProjectReportRevision{}, e
	}
	var evidence []reporting.EvidenceReference
	if e = json.Unmarshal(row.Evidence, &evidence); e != nil {
		return reporting.ProjectReportRevision{}, fmt.Errorf("decode report evidence: %w", e)
	}
	var report reporting.ResearchReportV1
	if e = json.Unmarshal(row.Report, &report); e != nil {
		return reporting.ProjectReportRevision{}, fmt.Errorf("decode report: %w", e)
	}
	risk := report.RiskSummary
	if !equalBoolPointers(risk.WethPairIsCreated, boolPointer(row.WethPairIsCreated)) ||
		!equalBoolPointers(risk.WethPairIsRemoveLiquidity, boolPointer(row.WethPairIsRemoveLiquidity)) ||
		!equalBoolPointers(risk.WethPairIsMint, boolPointer(row.WethPairIsMint)) ||
		!equalBigIntPointers(risk.WethPairQuoteUsdtValueInt, wethQuoteUSDTValueInt) ||
		!equalUint64Pointers(risk.WethPairLastSwapTimestamp, wts) ||
		!equalBoolPointers(risk.UsdtPairIsCreated, boolPointer(row.UsdtPairIsCreated)) ||
		!equalBoolPointers(risk.UsdtPairIsRemoveLiquidity, boolPointer(row.UsdtPairIsRemoveLiquidity)) ||
		!equalBoolPointers(risk.UsdtPairIsMint, boolPointer(row.UsdtPairIsMint)) ||
		!equalBigIntPointers(risk.UsdtPairQuoteUsdtValueInt, usdtQuoteUSDTValueInt) ||
		!equalUint64Pointers(risk.UsdtPairLastSwapTimestamp, uts) {
		return reporting.ProjectReportRevision{}, fmt.Errorf("project report revision %d risk projection does not match report JSON", row.ID)
	}
	return reporting.ProjectReportRevision{ID: row.ID, ProjectID: row.ProjectID, Revision: row.Revision, SchemaVersion: row.SchemaVersion, ContentHash: bytesToHash(row.ContentHash), CompletenessStatus: row.CompletenessStatus, Evidence: evidence, Report: report, ObservedBlockNumber: bn, BuiltAt: timeValue(row.BuiltAt), CreatedAt: timeValue(row.CreatedAt)}, nil
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
func mapProjectSelection(row tokensqlc.ProjectSelection) selection.ProjectSelection {
	return selection.ProjectSelection{ID: row.ID, ProjectID: row.ProjectID, Outcome: selection.SelectionOutcome(row.Outcome), StrategyKey: row.StrategyKey, StrategyVersion: row.StrategyVersion, ReportRevision: row.ReportRevision, ReasonCodes: row.ReasonCodes, ReasonDetail: row.ReasonDetail, DecidedAt: timeValue(row.DecidedAt), CreatedAt: timeValue(row.CreatedAt)}
}
