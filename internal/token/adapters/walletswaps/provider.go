package walletswaps

import (
	"context"
	"fmt"
	"math"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	bscswapapiclient "github.com/useryege/athena/internal/bscswap/apiclient"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/shared"
	utilio "github.com/useryege/athena/util/io"
)

type Provider struct {
	client bscswapapiclient.BscSwapTransactionServiceClient
	closer utilio.Closer
}

func New(address string) (*Provider, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("BSC swap transaction service address is required")
	}
	connection, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect to BSC swap transaction service: %w", err)
	}
	return &Provider{client: bscswapapiclient.NewBscSwapTransactionServiceClient(connection), closer: connection}, nil
}

func (provider *Provider) ListSwapTransactionsBeforeBlock(ctx context.Context, wallet shared.Address, beforeBlock uint64, limit int32) (research.WalletSwapTransactionHistory, error) {
	if provider == nil || provider.client == nil {
		return research.WalletSwapTransactionHistory{}, fmt.Errorf("wallet swap transaction provider is not configured")
	}
	if wallet.IsZero() {
		return research.WalletSwapTransactionHistory{}, fmt.Errorf("wallet swap transaction address is required")
	}
	if beforeBlock == 0 || beforeBlock > math.MaxInt64 {
		return research.WalletSwapTransactionHistory{}, fmt.Errorf("wallet swap transaction before block is invalid")
	}
	if limit <= 0 {
		return research.WalletSwapTransactionHistory{}, fmt.Errorf("wallet swap transaction limit must be positive")
	}
	response, err := provider.client.ListSwapTransactions(ctx, &bscswapapiclient.ListSwapTransactionsRequest{
		WalletAddress: wallet.Hex(), BeforeBlockNumber: beforeBlock, PageSize: limit,
	})
	if err != nil {
		return research.WalletSwapTransactionHistory{}, fmt.Errorf("fetch wallet swap transactions wallet=%s: %w", wallet.Hex(), err)
	}
	if response == nil {
		return research.WalletSwapTransactionHistory{}, fmt.Errorf("fetch wallet swap transactions wallet=%s returned empty response", wallet.Hex())
	}
	if len(response.GetTransactionHashes()) > int(limit) {
		return research.WalletSwapTransactionHistory{}, fmt.Errorf("fetch wallet swap transactions wallet=%s returned %d hashes for limit %d", wallet.Hex(), len(response.GetTransactionHashes()), limit)
	}
	if response.GetIndexedThroughBlock() > math.MaxInt64 || response.GetIndexedThroughTimestamp() > math.MaxInt64 {
		return research.WalletSwapTransactionHistory{}, fmt.Errorf("fetch wallet swap transactions wallet=%s returned invalid index position", wallet.Hex())
	}

	seen := make(map[shared.Hash]struct{}, len(response.GetTransactionHashes()))
	transactions := make([]research.WalletSwapTransaction, 0, len(response.GetTransactionHashes()))
	for index, value := range response.GetTransactionHashes() {
		hash, err := shared.HexToHash(value)
		if err != nil || hash.IsZero() {
			return research.WalletSwapTransactionHistory{}, fmt.Errorf("decode wallet swap transaction wallet=%s index=%d: invalid transaction hash", wallet.Hex(), index)
		}
		if _, exists := seen[hash]; exists {
			continue
		}
		seen[hash] = struct{}{}
		transactions = append(transactions, research.WalletSwapTransaction{TransactionHash: hash})
	}
	return research.WalletSwapTransactionHistory{
		Transactions: transactions, IndexedThroughBlock: response.GetIndexedThroughBlock(),
		IndexedThroughTimestamp: response.GetIndexedThroughTimestamp(),
	}, nil
}

func (provider *Provider) Close() error {
	if provider == nil || provider.closer == nil {
		return nil
	}
	err := provider.closer.Close()
	provider.closer = nil
	provider.client = nil
	return err
}
