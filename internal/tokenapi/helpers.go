package tokenapi

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/pkg/apis/application/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func parseAddressField(name, value string) (common.Address, error) {
	value = strings.TrimSpace(value)
	if !common.IsHexAddress(value) {
		return common.Address{}, status.Errorf(codes.InvalidArgument, "%s must be a valid hex address", name)
	}
	return common.HexToAddress(value), nil
}

func parseOptionalAddressField(name, value string) (common.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return common.Address{}, nil
	}
	return parseAddressField(name, value)
}

func parseHashField(name, value string) (common.Hash, error) {
	value = strings.TrimSpace(value)
	raw := strings.TrimPrefix(strings.TrimPrefix(value, "0x"), "0X")
	if len(raw) != 64 {
		return common.Hash{}, status.Errorf(codes.InvalidArgument, "%s must be a 32-byte hex hash", name)
	}
	bytes, err := hex.DecodeString(raw)
	if err != nil {
		return common.Hash{}, status.Errorf(codes.InvalidArgument, "%s must be a valid hex hash", name)
	}
	return common.BytesToHash(bytes), nil
}

func parseOptionalHashField(name, value string) (common.Hash, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return common.Hash{}, nil
	}
	return parseHashField(name, value)
}

func validatePositiveInt64Field(name string, value int64) error {
	if value <= 0 {
		return status.Errorf(codes.InvalidArgument, "%s must be positive", name)
	}
	return nil
}

func validateNonNegativeInt64Field(name string, value int64) error {
	if value < 0 {
		return status.Errorf(codes.InvalidArgument, "%s must not be negative", name)
	}
	return nil
}

func validateChainIngestStatus(value string) error {
	value = strings.TrimSpace(value)
	switch value {
	case tokenstore.ChainIngestStatusRunning, tokenstore.ChainIngestStatusStopped:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "status must be %q or %q", tokenstore.ChainIngestStatusRunning, tokenstore.ChainIngestStatusStopped)
	}
}

func validateRequiredProjectDataCollectionType(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return status.Error(codes.InvalidArgument, "data_type must not be empty")
	}
	return validateProjectDataCollectionType(value)
}

func validateProjectDataCollectionType(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch value {
	case tokenstore.ProjectDataCollectionTypeAve,
		tokenstore.ProjectDataCollectionTypeChainState,
		tokenstore.ProjectDataCollectionTypeWalletAssetState,
		tokenstore.ProjectDataCollectionTypeSimulationResult,
		tokenstore.ProjectDataCollectionTypeContractCodeSource:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "data_type must be one of %q, %q, %q, %q, or %q",
			tokenstore.ProjectDataCollectionTypeAve,
			tokenstore.ProjectDataCollectionTypeChainState,
			tokenstore.ProjectDataCollectionTypeWalletAssetState,
			tokenstore.ProjectDataCollectionTypeSimulationResult,
			tokenstore.ProjectDataCollectionTypeContractCodeSource,
		)
	}
}

func validateProjectDataCollectionStatus(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	switch value {
	case tokenstore.ProjectDataCollectionStatusPending,
		tokenstore.ProjectDataCollectionStatusSucceeded,
		tokenstore.ProjectDataCollectionStatusFailed:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "status must be one of %q, %q, or %q",
			tokenstore.ProjectDataCollectionStatusPending,
			tokenstore.ProjectDataCollectionStatusSucceeded,
			tokenstore.ProjectDataCollectionStatusFailed,
		)
	}
}

func grpcStoreError(err error) error {
	if err == nil {
		return nil
	}
	return status.Error(codes.Internal, err.Error())
}

func requiredStore(store *tokenstore.SQLStore) (*tokenstore.SQLStore, error) {
	if store == nil {
		return nil, status.Error(codes.Internal, "token postgres database is not configured")
	}
	return store, nil
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func formatHash(value common.Hash) string {
	if value == (common.Hash{}) {
		return ""
	}
	return value.Hex()
}

func mapChainIngestCheckpoint(item tokenstore.ChainIngestCheckpoint) *v1alpha1.TokenAPIChainIngestCheckpoint {
	return &v1alpha1.TokenAPIChainIngestCheckpoint{
		ChainID:           item.ChainID,
		ChainName:         item.ChainName,
		Enabled:           item.Enabled,
		CursorBlockNumber: item.CursorBlockNumber,
		Status:            item.Status,
		CreatedAt:         formatTime(item.CreatedAt),
	}
}

func mapChainIngestCheckpoints(items []tokenstore.ChainIngestCheckpoint) []*v1alpha1.TokenAPIChainIngestCheckpoint {
	results := make([]*v1alpha1.TokenAPIChainIngestCheckpoint, 0, len(items))
	for _, item := range items {
		results = append(results, mapChainIngestCheckpoint(item))
	}
	return results
}

func mapChainOptions(items []tokenstore.Chain) []v1alpha1.TokenAPIChainOption {
	results := make([]v1alpha1.TokenAPIChainOption, 0, len(items))
	for _, item := range items {
		results = append(results, v1alpha1.TokenAPIChainOption{
			ChainID:   item.ID,
			ChainName: item.Name,
		})
	}
	return results
}

func mapContractCode(item tokenstore.ContractCode) *v1alpha1.TokenAPIContractCode {
	return &v1alpha1.TokenAPIContractCode{
		CodeHash:            item.CodeHash.Hex(),
		SourceCode:          item.SourceCode,
		SourceCodeFetchedAt: formatTime(item.SourceCodeFetchedAt),
		CreatedAt:           formatTime(item.CreatedAt),
		DeploymentCount:     item.DeploymentCount,
	}
}

func mapContractCodes(items []tokenstore.ContractCode) []*v1alpha1.TokenAPIContractCode {
	results := make([]*v1alpha1.TokenAPIContractCode, 0, len(items))
	for _, item := range items {
		results = append(results, mapContractCode(item))
	}
	return results
}

func mapProject(item tokenstore.Project) *v1alpha1.TokenAPIProject {
	return &v1alpha1.TokenAPIProject{
		ProjectID:   item.ID,
		ChainID:     item.ChainID,
		Name:        item.Name,
		Symbol:      item.Symbol,
		Contract:    item.Contract.Hex(),
		Creator:     item.Creator.Hex(),
		TxHash:      item.TxHash.Hex(),
		TxIndex:     item.TxIndex,
		BlockNumber: item.BlockNumber,
		BlockTime:   item.BlockTime,
		CodeHash:    item.CodeHash.Hex(),
		CreatedAt:   formatTime(item.CreatedAt),
	}
}

func mapProjects(items []tokenstore.Project) []*v1alpha1.TokenAPIProject {
	results := make([]*v1alpha1.TokenAPIProject, 0, len(items))
	for _, item := range items {
		results = append(results, mapProject(item))
	}
	return results
}

func mapProjectDataCollectionTask(item tokenstore.ProjectDataCollectionTask) *v1alpha1.TokenAPIProjectDataCollectionTask {
	return &v1alpha1.TokenAPIProjectDataCollectionTask{
		ProjectID:     item.ProjectID,
		DataType:      item.DataType,
		Status:        item.Status,
		Attempts:      item.Attempts,
		NextAttemptAt: formatTime(item.NextAttemptAt),
		LastError:     item.LastError,
		CreatedAt:     formatTime(item.CreatedAt),
	}
}

func mapProjectDataCollectionTasks(items []tokenstore.ProjectDataCollectionTask) []*v1alpha1.TokenAPIProjectDataCollectionTask {
	results := make([]*v1alpha1.TokenAPIProjectDataCollectionTask, 0, len(items))
	for _, item := range items {
		results = append(results, mapProjectDataCollectionTask(item))
	}
	return results
}

func mapBytecodeBlacklist(item tokenstore.BytecodeBlacklistEntry) *v1alpha1.TokenAPIBytecodeBlacklist {
	sourceContract := ""
	if item.SourceContract != (common.Address{}) {
		sourceContract = item.SourceContract.Hex()
	}
	return &v1alpha1.TokenAPIBytecodeBlacklist{
		CodeHash:       item.CodeHash.Hex(),
		Note:           item.Note,
		SourceChainID:  item.SourceChainID,
		SourceContract: sourceContract,
		CreatedAt:      formatTime(item.CreatedAt),
	}
}

func mapBytecodeBlacklists(items []tokenstore.BytecodeBlacklistEntry) []*v1alpha1.TokenAPIBytecodeBlacklist {
	results := make([]*v1alpha1.TokenAPIBytecodeBlacklist, 0, len(items))
	for _, item := range items {
		results = append(results, mapBytecodeBlacklist(item))
	}
	return results
}

func mapWalletBlacklist(item tokenstore.WalletBlacklistEntry) *v1alpha1.TokenAPIWalletBlacklist {
	return &v1alpha1.TokenAPIWalletBlacklist{
		Wallet:    item.Wallet.Hex(),
		Note:      item.Note,
		CreatedAt: formatTime(item.CreatedAt),
	}
}

func mapWalletBlacklists(items []tokenstore.WalletBlacklistEntry) []*v1alpha1.TokenAPIWalletBlacklist {
	results := make([]*v1alpha1.TokenAPIWalletBlacklist, 0, len(items))
	for _, item := range items {
		results = append(results, mapWalletBlacklist(item))
	}
	return results
}

func wrapStoreError(action string, err error) error {
	if err == nil {
		return nil
	}
	return grpcStoreError(fmt.Errorf("%s: %w", action, err))
}
