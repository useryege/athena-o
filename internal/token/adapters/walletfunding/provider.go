package walletfunding

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	bscinboundapiclient "github.com/useryege/athena/internal/bscinbound/apiclient"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/shared"
	utilio "github.com/useryege/athena/util/io"
)

var minimumFundingSourceValueWei = big.NewInt(10_000_000_000_000_000)

type Provider struct {
	client bscinboundapiclient.BscInboundTransactionServiceClient
	closer utilio.Closer
}

func New(address string) (*Provider, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("BSC inbound transaction service address is required")
	}
	connection, err := grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("connect to BSC inbound transaction service: %w", err)
	}
	return &Provider{client: bscinboundapiclient.NewBscInboundTransactionServiceClient(connection), closer: connection}, nil
}

func (provider *Provider) ListFundingSourcesBefore(ctx context.Context, wallet shared.Address, anchorBlock, anchorTransactionIndex uint64, limit int32) (research.WalletFundingSourceHistory, error) {
	if provider == nil || provider.client == nil {
		return research.WalletFundingSourceHistory{}, fmt.Errorf("wallet funding source provider is not configured")
	}
	if wallet.IsZero() {
		return research.WalletFundingSourceHistory{}, fmt.Errorf("wallet funding source address is required")
	}
	if limit <= 0 {
		return research.WalletFundingSourceHistory{}, fmt.Errorf("wallet funding source limit must be positive")
	}
	response, err := provider.client.ListInboundNormalTransactions(ctx, &bscinboundapiclient.ListInboundNormalTransactionsRequest{
		Address:  wallet.Hex(),
		PageSize: limit,
		BeforePosition: &bscinboundapiclient.InboundNormalTransactionPosition{
			BlockNumber: anchorBlock, TransactionIndex: anchorTransactionIndex,
		},
	})
	if err != nil {
		return research.WalletFundingSourceHistory{}, fmt.Errorf("fetch wallet funding sources wallet=%s: %w", wallet.Hex(), err)
	}
	if response == nil {
		return research.WalletFundingSourceHistory{}, fmt.Errorf("fetch wallet funding sources wallet=%s returned empty response", wallet.Hex())
	}

	seen := make(map[shared.Hash]struct{}, len(response.GetTransactions()))
	transactions := make([]research.WalletFundingSourceTransaction, 0, min(len(response.GetTransactions()), int(limit)))
	for index, item := range response.GetTransactions() {
		transaction, err := mapFundingSourceTransaction(item)
		if err != nil {
			return research.WalletFundingSourceHistory{}, fmt.Errorf("decode wallet funding source wallet=%s index=%d: %w", wallet.Hex(), index, err)
		}
		if transaction.To != wallet {
			return research.WalletFundingSourceHistory{}, fmt.Errorf("wallet funding source recipient %s does not match wallet %s", transaction.To.Hex(), wallet.Hex())
		}
		if !isBeforeAnchor(transaction, anchorBlock, anchorTransactionIndex) {
			return research.WalletFundingSourceHistory{}, fmt.Errorf("wallet funding source %s is not before anchor %d:%d", transaction.TransactionHash.Hex(), anchorBlock, anchorTransactionIndex)
		}
		if _, exists := seen[transaction.TransactionHash]; exists {
			continue
		}
		seen[transaction.TransactionHash] = struct{}{}
		transactions = append(transactions, transaction)
	}
	sort.Slice(transactions, func(left, right int) bool {
		if transactions[left].BlockNumber != transactions[right].BlockNumber {
			return transactions[left].BlockNumber > transactions[right].BlockNumber
		}
		if transactions[left].TransactionIndex != transactions[right].TransactionIndex {
			return transactions[left].TransactionIndex > transactions[right].TransactionIndex
		}
		return transactions[left].TransactionHash.Hex() > transactions[right].TransactionHash.Hex()
	})
	if len(transactions) > int(limit) {
		transactions = transactions[:limit]
	}
	return research.WalletFundingSourceHistory{
		Transactions: transactions, IndexedThroughBlock: response.GetIndexedThroughBlock(),
		IndexedThroughTimestamp: response.GetIndexedThroughTimestamp(),
	}, nil
}

func mapFundingSourceTransaction(item *bscinboundapiclient.InboundNormalTransaction) (research.WalletFundingSourceTransaction, error) {
	if item == nil {
		return research.WalletFundingSourceTransaction{}, fmt.Errorf("transaction is nil")
	}
	blockHash, err := shared.HexToHash(item.GetBlockHash())
	if err != nil {
		return research.WalletFundingSourceTransaction{}, fmt.Errorf("parse block hash: %w", err)
	}
	transactionHash, err := shared.HexToHash(item.GetTransactionHash())
	if err != nil {
		return research.WalletFundingSourceTransaction{}, fmt.Errorf("parse transaction hash: %w", err)
	}
	from, err := shared.HexToAddress(item.GetFromAddress())
	if err != nil {
		return research.WalletFundingSourceTransaction{}, fmt.Errorf("parse from address: %w", err)
	}
	to, err := shared.HexToAddress(item.GetToAddress())
	if err != nil {
		return research.WalletFundingSourceTransaction{}, fmt.Errorf("parse to address: %w", err)
	}
	valueWei, ok := new(big.Int).SetString(strings.TrimSpace(item.GetValueWei()), 10)
	if !ok || valueWei.Cmp(minimumFundingSourceValueWei) <= 0 {
		return research.WalletFundingSourceTransaction{}, fmt.Errorf("value_wei must be greater than %s", minimumFundingSourceValueWei.String())
	}
	return research.WalletFundingSourceTransaction{
		BlockNumber: item.GetBlockNumber(), BlockHash: blockHash, BlockTimestamp: item.GetBlockTimestamp(),
		TransactionHash: transactionHash, TransactionIndex: item.GetTransactionIndex(),
		From: from, To: to, ValueWei: valueWei,
	}, nil
}

func isBeforeAnchor(transaction research.WalletFundingSourceTransaction, blockNumber, transactionIndex uint64) bool {
	return transaction.BlockNumber < blockNumber || (transaction.BlockNumber == blockNumber && transaction.TransactionIndex < transactionIndex)
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
