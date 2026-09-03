package postgres

import (
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	tokensqlc "github.com/useryege/athena/internal/token/adapters/postgres/sqlc"
	"github.com/useryege/athena/internal/token/catalog"
	"github.com/useryege/athena/internal/token/collection"
	"github.com/useryege/athena/internal/token/discovery"
	"github.com/useryege/athena/internal/token/policy"
	"github.com/useryege/athena/internal/token/shared"
)

const (
	defaultPageSize = int32(20)
	maxPageSize     = int32(200)
)

func bytesToHash(value []byte) shared.Hash {
	if len(value) == 0 {
		return shared.Hash{}
	}
	return shared.BytesToHash(value)
}

func bytesToAddress(value []byte) shared.Address {
	if len(value) == 0 {
		return shared.Address{}
	}
	return shared.BytesToAddress(value)
}

func optionalHashBytes(value shared.Hash) []byte {
	if value.IsZero() {
		return nil
	}
	return value.Bytes()
}

func optionalAddressBytes(value shared.Address) []byte {
	if value.IsZero() {
		return nil
	}
	return value.Bytes()
}

func nullableText(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func textValue(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableTime(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: !value.IsZero()}
}

func timeValue(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func nullableInt64(value int64) pgtype.Int8 {
	return pgtype.Int8{Int64: value, Valid: value != 0}
}

func int64Value(value pgtype.Int8) int64 {
	if !value.Valid {
		return 0
	}
	return value.Int64
}

func numericFromBigInt(value *big.Int) pgtype.Numeric {
	if value == nil {
		value = new(big.Int)
	}
	return pgtype.Numeric{Int: new(big.Int).Set(value), Exp: 0, Valid: true}
}

func nullableNumericFromBigInt(value *big.Int) pgtype.Numeric {
	if value == nil {
		return pgtype.Numeric{}
	}
	return numericFromBigInt(value)
}

func bigIntFromNumeric(value pgtype.Numeric) *big.Int {
	if !value.Valid || value.Int == nil {
		return new(big.Int)
	}
	result := new(big.Int).Set(value.Int)
	if value.Exp > 0 {
		return result.Mul(result, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(value.Exp)), nil))
	}
	if value.Exp < 0 {
		return result.Quo(result, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-value.Exp)), nil))
	}
	return result
}

func exactBigIntPointerFromNumeric(field string, value pgtype.Numeric) (*big.Int, error) {
	if !value.Valid {
		return nil, nil
	}
	if value.NaN || value.InfinityModifier != pgtype.Finite || value.Int == nil {
		return nil, fmt.Errorf("%s is not a finite numeric value", field)
	}
	result := new(big.Int).Set(value.Int)
	if value.Exp > 0 {
		result.Mul(result, new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(value.Exp)), nil))
	} else if value.Exp < 0 {
		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(-value.Exp)), nil)
		quotient, remainder := new(big.Int), new(big.Int)
		quotient.QuoRem(result, divisor, remainder)
		if remainder.Sign() != 0 {
			return nil, fmt.Errorf("%s is not an integer", field)
		}
		result = quotient
	}
	return result, nil
}

func nullableNumericFromDecimal(value *collection.Decimal) (pgtype.Numeric, error) {
	if value == nil {
		return pgtype.Numeric{}, nil
	}
	var result pgtype.Numeric
	if err := result.ScanScientific(value.String()); err != nil {
		return pgtype.Numeric{}, err
	}
	return result, nil
}

func decimalFromNumeric(field string, value pgtype.Numeric) (*collection.Decimal, error) {
	if !value.Valid {
		return nil, nil
	}
	if value.NaN || value.InfinityModifier != pgtype.Finite {
		return nil, fmt.Errorf("%s is not a finite numeric value", field)
	}
	raw, err := value.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("marshal %s numeric: %w", field, err)
	}
	parsed, err := collection.ParseDecimal(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse %s numeric: %w", field, err)
	}
	return &parsed, nil
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

func nullableUint64(value *uint64) (pgtype.Int8, error) {
	if value == nil {
		return pgtype.Int8{}, nil
	}
	mapped, err := uint64ToInt64("uint64", *value)
	return pgtype.Int8{Int64: mapped, Valid: err == nil}, err
}

func uint64PointerFromInt64(field string, value pgtype.Int8) (*uint64, error) {
	if !value.Valid {
		return nil, nil
	}
	mapped, err := int64ToUint64(field, value.Int64)
	if err != nil {
		return nil, err
	}
	return &mapped, nil
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
	cursor, err := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	return &discovery.ChainProcessingCheckpoint{
		ChainID: row.ChainID, ChainName: row.ChainName, Enabled: row.Enabled, CursorBlockNumber: cursor,
		Status: discovery.ChainProcessingStatus(row.Status), CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt),
	}, nil
}

func mapChainCheckpointListRow(row tokensqlc.ListChainProcessingCheckpointsRow) (*discovery.ChainProcessingCheckpoint, error) {
	return mapChainCheckpointRow(tokensqlc.GetChainProcessingCheckpointRow(row))
}

func mapChainCheckpoint(row tokensqlc.ChainProcessingCheckpoint) (*discovery.ChainProcessingCheckpoint, error) {
	cursor, err := int64ToUint64("cursor_block_number", row.CursorBlockNumber)
	if err != nil {
		return nil, err
	}
	return &discovery.ChainProcessingCheckpoint{
		ChainID: row.ChainID, CursorBlockNumber: cursor, Status: discovery.ChainProcessingStatus(row.Status),
		CreatedAt: timeValue(row.CreatedAt), UpdatedAt: timeValue(row.UpdatedAt),
	}, nil
}

func mapContractCode(row tokensqlc.ContractCode) *catalog.ContractCode {
	return &catalog.ContractCode{
		CodeHash: bytesToHash(row.CodeHash), SourceCode: textValue(row.SourceCode),
		SourceCodeFetchedAt: timeValue(row.SourceCodeFetchedAt), DeploymentCount: row.DeploymentCount,
		CreatedAt: timeValue(row.CreatedAt),
	}
}

func mapContractCodes(rows []tokensqlc.ContractCode) []catalog.ContractCode {
	result := make([]catalog.ContractCode, 0, len(rows))
	for _, row := range rows {
		result = append(result, *mapContractCode(row))
	}
	return result
}

func mapProject(row tokensqlc.Project) (*catalog.Project, error) {
	txIndex, err := int64ToUint64("tx_index", row.TxIndex)
	if err != nil {
		return nil, err
	}
	deploymentNonce, err := int64ToUint64("deployment_nonce", row.DeploymentNonce)
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
	if row.Decimals < 0 || row.Decimals > math.MaxUint8 {
		return nil, fmt.Errorf("decimals exceeds uint8 range")
	}
	return &catalog.Project{
		ID: row.ID, ChainID: row.ChainID, Contract: bytesToAddress(row.Contract), TxSender: bytesToAddress(row.TxSender),
		TxHash: bytesToHash(row.TxHash), TxIndex: txIndex, DeploymentNonce: deploymentNonce,
		BlockNumber: blockNumber, BlockTime: blockTime, CodeHash: bytesToHash(row.CodeHash), Name: row.Name,
		Symbol: row.Symbol, Decimals: uint8(row.Decimals), TotalSupply: bigIntFromNumeric(row.TotalSupply),
		WethPair: bytesToAddress(row.WethPair), UsdtPair: bytesToAddress(row.UsdtPair), CreatedAt: timeValue(row.CreatedAt),
	}, nil
}

func mapProjects(rows []tokensqlc.Project) ([]catalog.Project, error) {
	result := make([]catalog.Project, 0, len(rows))
	for _, row := range rows {
		project, err := mapProject(row)
		if err != nil {
			return nil, err
		}
		result = append(result, *project)
	}
	return result, nil
}

func mapProjectRelatedWallet(row tokensqlc.ProjectRelatedWallet) *catalog.ProjectRelatedWallet {
	return &catalog.ProjectRelatedWallet{
		ProjectID: row.ProjectID, Wallet: bytesToAddress(row.Wallet), Role: catalog.RelatedWalletRole(row.Role),
		CreatedAt: timeValue(row.CreatedAt),
	}
}

func mapProjectRelatedWallets(rows []tokensqlc.ProjectRelatedWallet) []catalog.ProjectRelatedWallet {
	result := make([]catalog.ProjectRelatedWallet, 0, len(rows))
	for _, row := range rows {
		result = append(result, *mapProjectRelatedWallet(row))
	}
	return result
}

func mapProjectInitialRecipient(row tokensqlc.ProjectInitialRecipient) (*catalog.ProjectInitialRecipient, error) {
	blockNumber, err := int64ToUint64("source_block_number", row.SourceBlockNumber)
	if err != nil {
		return nil, err
	}
	return &catalog.ProjectInitialRecipient{
		ID: row.ID, ProjectID: row.ProjectID, Wallet: bytesToAddress(row.Wallet), RatioBPS: row.RatioBps,
		RankIndex: row.RankIndex, SourceTxHash: bytesToHash(row.SourceTxHash), SourceBlockNumber: blockNumber,
		CreatedAt: timeValue(row.CreatedAt),
	}, nil
}

func mapProjectInitialRecipients(rows []tokensqlc.ProjectInitialRecipient) ([]catalog.ProjectInitialRecipient, error) {
	result := make([]catalog.ProjectInitialRecipient, 0, len(rows))
	for _, row := range rows {
		recipient, err := mapProjectInitialRecipient(row)
		if err != nil {
			return nil, err
		}
		result = append(result, *recipient)
	}
	return result, nil
}

func mapContractCodeBlocklistEntry(row tokensqlc.ContractCodeBlocklist) *policy.ContractCodeBlocklistEntry {
	return &policy.ContractCodeBlocklistEntry{
		CodeHash: bytesToHash(row.CodeHash), Note: textValue(row.Note), SourceChainID: int64Value(row.SourceChainID),
		SourceContract: bytesToAddress(row.SourceContract), CreatedAt: timeValue(row.CreatedAt),
	}
}

func mapContractCodeBlocklistEntries(rows []tokensqlc.ContractCodeBlocklist) []policy.ContractCodeBlocklistEntry {
	result := make([]policy.ContractCodeBlocklistEntry, 0, len(rows))
	for _, row := range rows {
		result = append(result, *mapContractCodeBlocklistEntry(row))
	}
	return result
}

func mapWalletBlocklistEntry(row tokensqlc.WalletBlocklist) *policy.WalletBlocklistEntry {
	return &policy.WalletBlocklistEntry{Wallet: bytesToAddress(row.Wallet), Note: textValue(row.Note), CreatedAt: timeValue(row.CreatedAt)}
}

func mapWalletBlocklistEntries(rows []tokensqlc.WalletBlocklist) []policy.WalletBlocklistEntry {
	result := make([]policy.WalletBlocklistEntry, 0, len(rows))
	for _, row := range rows {
		result = append(result, *mapWalletBlocklistEntry(row))
	}
	return result
}
