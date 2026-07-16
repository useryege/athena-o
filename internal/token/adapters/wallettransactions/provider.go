package wallettransactions

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	ethereumapiapiclient "github.com/useryege/athena/internal/ethereumapi/apiclient"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/shared"
	utilio "github.com/useryege/athena/util/io"
)

type Provider struct {
	client ethereumapiapiclient.EthereumAPIServiceClient
	closer utilio.Closer
}

func New(address string) (*Provider, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("Ethereum API address is required")
	}
	closer, client, err := ethereumapiapiclient.NewEthereumAPIClientset(address).NewEthereumAPIServiceClient()
	if err != nil {
		return nil, err
	}
	return &Provider{client: client, closer: closer}, nil
}

func (provider *Provider) ListNormalTransactionsBefore(ctx context.Context, chainID int64, wallet shared.Address, anchorBlock, anchorTransactionIndex uint64, limit int32) ([]research.WalletNormalTransaction, error) {
	if provider == nil || provider.client == nil {
		return nil, fmt.Errorf("wallet normal transaction history provider is not configured")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("normal transaction limit must be positive")
	}
	page := int32(1)
	seen := make(map[shared.Hash]struct{}, limit)
	transactions := make([]research.WalletNormalTransaction, 0, limit)
	for {
		response, err := provider.client.ListNormalTransactions(ctx, &ethereumapiapiclient.ListNormalTransactionsRequest{
			ChainId: chainID, Address: wallet.Hex(),
			BlockRange: &ethereumapiapiclient.NormalTransactionBlockRange{StartBlock: 0, EndBlock: anchorBlock},
			Page:       page, PageSize: limit, Sort: ethereumapiapiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_DESC,
		})
		if err != nil {
			return nil, fmt.Errorf("fetch wallet normal transactions chain_id=%d wallet=%s page=%d: %w", chainID, wallet.Hex(), page, err)
		}
		if response == nil {
			return nil, fmt.Errorf("fetch wallet normal transactions chain_id=%d wallet=%s page=%d returned empty response", chainID, wallet.Hex(), page)
		}
		for index, item := range response.GetTransactions() {
			transaction, err := mapNormalTransaction(item)
			if err != nil {
				return nil, fmt.Errorf("decode wallet normal transaction chain_id=%d wallet=%s page=%d index=%d: %w", chainID, wallet.Hex(), page, index, err)
			}
			if !isBeforeAnchor(transaction, anchorBlock, anchorTransactionIndex) {
				continue
			}
			if _, exists := seen[transaction.TransactionHash]; exists {
				continue
			}
			seen[transaction.TransactionHash] = struct{}{}
			transactions = append(transactions, transaction)
		}
		if len(transactions) >= int(limit) || len(response.GetTransactions()) < int(limit) {
			break
		}
		if page == math.MaxInt32 {
			return nil, fmt.Errorf("wallet normal transaction pagination exceeded int32 page range")
		}
		page++
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
	return transactions, nil
}

func isBeforeAnchor(transaction research.WalletNormalTransaction, blockNumber, transactionIndex uint64) bool {
	return transaction.BlockNumber < blockNumber || (transaction.BlockNumber == blockNumber && transaction.TransactionIndex < transactionIndex)
}

func mapNormalTransaction(item *ethereumapiapiclient.NormalTransaction) (research.WalletNormalTransaction, error) {
	if item == nil {
		return research.WalletNormalTransaction{}, fmt.Errorf("transaction is nil")
	}
	blockHash, err := shared.HexToHash(item.GetBlockHash())
	if err != nil {
		return research.WalletNormalTransaction{}, fmt.Errorf("parse block hash: %w", err)
	}
	transactionHash, err := shared.HexToHash(item.GetTransactionHash())
	if err != nil {
		return research.WalletNormalTransaction{}, fmt.Errorf("parse transaction hash: %w", err)
	}
	from, err := shared.HexToAddress(item.GetFromAddress())
	if err != nil {
		return research.WalletNormalTransaction{}, fmt.Errorf("parse from address: %w", err)
	}
	to, err := optionalAddress(item.GetToAddress())
	if err != nil {
		return research.WalletNormalTransaction{}, fmt.Errorf("parse to address: %w", err)
	}
	contract, err := optionalAddress(item.GetContractAddress())
	if err != nil {
		return research.WalletNormalTransaction{}, fmt.Errorf("parse contract address: %w", err)
	}
	value, err := decimalInteger("value", item.GetValue())
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	gasPrice, err := decimalInteger("gas_price", item.GetGasPrice())
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	methodID, err := optionalBytes(item.GetMethodId())
	if err != nil {
		return research.WalletNormalTransaction{}, fmt.Errorf("parse method ID: %w", err)
	}
	receiptStatus, err := receiptStatus(item.GetReceiptStatus())
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	return research.WalletNormalTransaction{
		BlockNumber: item.GetBlockNumber(), BlockHash: blockHash, BlockTimestamp: item.GetBlockTimestamp(),
		TransactionHash: transactionHash, Nonce: item.GetNonce(), TransactionIndex: item.GetTransactionIndex(),
		From: from, To: to, Value: value, Gas: item.GetGas(), GasPrice: gasPrice, Input: item.GetInput(),
		MethodID: methodID, FunctionName: item.GetFunctionName(), ContractAddress: contract,
		CumulativeGasUsed: item.GetCumulativeGasUsed(), ReceiptStatus: receiptStatus,
		GasUsed: item.GetGasUsed(), Confirmations: item.GetConfirmations(), IsError: item.GetIsError(),
	}, nil
}

func optionalAddress(value string) (shared.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return shared.Address{}, nil
	}
	return shared.HexToAddress(value)
}

func optionalBytes(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	return hexutil.Decode(value)
}

func decimalInteger(field, value string) (*big.Int, error) {
	result, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
	if !ok || result.Sign() < 0 {
		return nil, fmt.Errorf("%s must be a nonnegative decimal integer", field)
	}
	return result, nil
}

func receiptStatus(value ethereumapiapiclient.NormalTransactionReceiptStatus) (research.WalletNormalTransactionReceiptStatus, error) {
	switch value {
	case ethereumapiapiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_UNSPECIFIED:
		return research.WalletNormalTransactionReceiptStatusUnspecified, nil
	case ethereumapiapiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_FAILED:
		return research.WalletNormalTransactionReceiptStatusFailed, nil
	case ethereumapiapiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_SUCCESS:
		return research.WalletNormalTransactionReceiptStatusSuccess, nil
	default:
		return "", fmt.Errorf("unsupported normal transaction receipt status %d", value)
	}
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
