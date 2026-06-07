package tokenapi

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	tokenstore "github.com/useryege/athena/internal/token/store"
	"github.com/useryege/athena/internal/tokenapi/apiclient"
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

func validateChainIngestStatus(value string) error {
	value = strings.TrimSpace(value)
	switch value {
	case tokenstore.ChainIngestStatusRunning, tokenstore.ChainIngestStatusStopped:
		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "status must be %q or %q", tokenstore.ChainIngestStatusRunning, tokenstore.ChainIngestStatusStopped)
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

func mapChainIngestCheckpoint(item tokenstore.ChainIngestCheckpoint) *apiclient.ChainIngestCheckpoint {
	return &apiclient.ChainIngestCheckpoint{
		ChainId:           item.ChainID,
		ChainName:         item.ChainName,
		Enabled:           item.Enabled,
		CursorBlockNumber: item.CursorBlockNumber,
		Status:            item.Status,
		CreatedAt:         formatTime(item.CreatedAt),
	}
}

func mapChainIngestCheckpoints(items []tokenstore.ChainIngestCheckpoint) []*apiclient.ChainIngestCheckpoint {
	results := make([]*apiclient.ChainIngestCheckpoint, 0, len(items))
	for _, item := range items {
		results = append(results, mapChainIngestCheckpoint(item))
	}
	return results
}

func mapContractCode(item tokenstore.ContractCode) *apiclient.ContractCode {
	return &apiclient.ContractCode{
		CodeHash:            item.CodeHash.Hex(),
		SourceCode:          item.SourceCode,
		SourceCodeHash:      formatHash(item.SourceCodeHash),
		SourceCodeFetchedAt: formatTime(item.SourceCodeFetchedAt),
		CreatedAt:           formatTime(item.CreatedAt),
		DeploymentCount:     item.DeploymentCount,
	}
}

func mapContractCodes(items []tokenstore.ContractCode) []*apiclient.ContractCode {
	results := make([]*apiclient.ContractCode, 0, len(items))
	for _, item := range items {
		results = append(results, mapContractCode(item))
	}
	return results
}

func mapBytecodeBlacklist(item tokenstore.BytecodeBlacklistEntry) *apiclient.BytecodeBlacklist {
	sourceContract := ""
	if item.SourceContract != (common.Address{}) {
		sourceContract = item.SourceContract.Hex()
	}
	return &apiclient.BytecodeBlacklist{
		CodeHash:       item.CodeHash.Hex(),
		Note:           item.Note,
		SourceChainId:  item.SourceChainID,
		SourceContract: sourceContract,
		CreatedAt:      formatTime(item.CreatedAt),
	}
}

func mapBytecodeBlacklists(items []tokenstore.BytecodeBlacklistEntry) []*apiclient.BytecodeBlacklist {
	results := make([]*apiclient.BytecodeBlacklist, 0, len(items))
	for _, item := range items {
		results = append(results, mapBytecodeBlacklist(item))
	}
	return results
}

func mapWalletBlacklist(item tokenstore.WalletBlacklistEntry) *apiclient.WalletBlacklist {
	return &apiclient.WalletBlacklist{
		Wallet:    item.Wallet.Hex(),
		Note:      item.Note,
		CreatedAt: formatTime(item.CreatedAt),
	}
}

func mapWalletBlacklists(items []tokenstore.WalletBlacklistEntry) []*apiclient.WalletBlacklist {
	results := make([]*apiclient.WalletBlacklist, 0, len(items))
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
