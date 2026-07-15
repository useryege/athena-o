package reporting

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"
	"time"

	"github.com/useryege/athena/internal/token/shared"
)

const ObservationSchemaVersionV1 = int32(1)

func BuildReport(projectID int64, items []ObservationSnapshot, builtAt time.Time) (ProjectReportRevision, error) {
	items = append([]ObservationSnapshot(nil), items...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].DataType != items[j].DataType {
			return items[i].DataType < items[j].DataType
		}
		return items[i].ID < items[j].ID
	})
	evidence := make([]EvidenceReference, 0, len(items))
	freshness := make([]ObservationFreshness, 0, len(items))
	observations := ResearchObservationsV1{}
	var maxBlock *uint64
	for _, item := range items {
		if item.SchemaVersion != ObservationSchemaVersionV1 {
			return ProjectReportRevision{}, fmt.Errorf("unsupported %s observation schema version %d", item.DataType, item.SchemaVersion)
		}
		evidence = append(evidence, EvidenceReference{ObservationID: item.ID, DataType: item.DataType, SchemaVersion: item.SchemaVersion, ContentHash: item.ContentHash.Hex(), BlockNumber: item.BlockNumber})
		freshness = append(freshness, ObservationFreshness{DataType: item.DataType, LastCheckedAt: item.LastCheckedAt})
		if item.BlockNumber != nil && (maxBlock == nil || *item.BlockNumber > *maxBlock) {
			value := *item.BlockNumber
			maxBlock = &value
		}
		if err := setObservation(&observations, item.DataType, item.Payload); err != nil {
			return ProjectReportRevision{}, err
		}
	}

	completeness := "complete"
	if len(observations.Ave) == 0 || len(observations.ChainState) == 0 || len(observations.WalletAssets) == 0 || len(observations.Simulation) == 0 || len(observations.ContractSource) == 0 {
		completeness = "incomplete"
	}
	risk, err := buildRiskSummary(observations.ChainState)
	if err != nil {
		return ProjectReportRevision{}, err
	}
	report := ResearchReportV1{SchemaVersion: ReportSchemaVersionV1, ProjectID: projectID, CompletenessStatus: completeness, Observations: observations, Freshness: freshness, RiskSummary: risk}
	canonical, err := json.Marshal(struct {
		SchemaVersion int32               `json:"schemaVersion"`
		Evidence      []EvidenceReference `json:"evidence"`
		Report        ResearchReportV1    `json:"report"`
	}{SchemaVersion: ReportSchemaVersionV1, Evidence: evidence, Report: report})
	if err != nil {
		return ProjectReportRevision{}, err
	}
	digest := sha256.Sum256(canonical)
	return ProjectReportRevision{ProjectID: projectID, SchemaVersion: ReportSchemaVersionV1, ContentHash: shared.BytesToHash(digest[:]), CompletenessStatus: completeness, Evidence: evidence, Report: report, ObservedBlockNumber: maxBlock, BuiltAt: builtAt.UTC()}, nil
}

func setObservation(target *ResearchObservationsV1, dataType DataCollectionType, payload json.RawMessage) error {
	if !json.Valid(payload) {
		return fmt.Errorf("decode %s observation schema version %d: invalid JSON", dataType, ObservationSchemaVersionV1)
	}
	canonical := append(json.RawMessage(nil), payload...)
	switch dataType {
	case DataCollectionTypeAve:
		target.Ave = canonical
	case DataCollectionTypeChainState:
		target.ChainState = canonical
	case DataCollectionTypeWalletAssetState:
		target.WalletAssets = canonical
	case DataCollectionTypeSimulationResult:
		target.Simulation = canonical
	case DataCollectionTypeContractCodeSource:
		target.ContractSource = canonical
	default:
		return fmt.Errorf("unsupported observation data type %q", dataType)
	}
	return nil
}

func buildRiskSummary(payload json.RawMessage) (ReportRiskSummary, error) {
	if len(payload) == 0 {
		return ReportRiskSummary{}, nil
	}
	var state struct {
		WethPair struct {
			IsCreated         bool     `json:"isCreated"`
			QuoteUsdtValueInt *big.Int `json:"quoteUsdtValueInt"`
			LastSwapTimestamp uint32   `json:"lastSwapTimestamp"`
		} `json:"wethPair"`
		WethReport struct {
			IsRemoveLiquidity bool `json:"isRemoveLiquidity"`
			IsMint            bool `json:"isMint"`
		} `json:"wethReport"`
		UsdtPair struct {
			IsCreated         bool     `json:"isCreated"`
			QuoteUsdtValueInt *big.Int `json:"quoteUsdtValueInt"`
			LastSwapTimestamp uint32   `json:"lastSwapTimestamp"`
		} `json:"usdtPair"`
		UsdtReport struct {
			IsRemoveLiquidity bool `json:"isRemoveLiquidity"`
			IsMint            bool `json:"isMint"`
		} `json:"usdtReport"`
	}
	if err := json.Unmarshal(payload, &state); err != nil {
		return ReportRiskSummary{}, fmt.Errorf("decode chain_state observation risk projection: %w", err)
	}
	return ReportRiskSummary{
		WethPairIsCreated:         boolPtr(state.WethPair.IsCreated),
		WethPairIsRemoveLiquidity: boolPtr(state.WethReport.IsRemoveLiquidity),
		WethPairIsMint:            boolPtr(state.WethReport.IsMint),
		WethPairQuoteUsdtValueInt: cloneBigInt(state.WethPair.QuoteUsdtValueInt),
		WethPairLastSwapTimestamp: uint64Ptr(uint64(state.WethPair.LastSwapTimestamp)),
		UsdtPairIsCreated:         boolPtr(state.UsdtPair.IsCreated),
		UsdtPairIsRemoveLiquidity: boolPtr(state.UsdtReport.IsRemoveLiquidity),
		UsdtPairIsMint:            boolPtr(state.UsdtReport.IsMint),
		UsdtPairQuoteUsdtValueInt: cloneBigInt(state.UsdtPair.QuoteUsdtValueInt),
		UsdtPairLastSwapTimestamp: uint64Ptr(uint64(state.UsdtPair.LastSwapTimestamp)),
	}, nil
}

func boolPtr(value bool) *bool       { return &value }
func uint64Ptr(value uint64) *uint64 { return &value }
func cloneBigInt(value *big.Int) *big.Int {
	if value == nil {
		return new(big.Int)
	}
	return new(big.Int).Set(value)
}
