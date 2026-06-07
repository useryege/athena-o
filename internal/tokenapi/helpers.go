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
