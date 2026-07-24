package normaltransactions

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	etherscanmanagerapiclient "github.com/useryege/athena/internal/etherscanmanager/apiclient"
	"github.com/useryege/athena/internal/token/research"
	"github.com/useryege/athena/internal/token/shared"
	utilio "github.com/useryege/athena/util/io"
)

const normalTransactionPageSize = int32(300)

type Provider struct {
	client etherscanmanagerapiclient.EtherscanManagerServiceClient
	closer utilio.Closer
}

func New(address string) (*Provider, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return nil, fmt.Errorf("Etherscan Manager address is required")
	}
	closer, client, err := etherscanmanagerapiclient.NewEtherscanManagerClientset(address).NewEtherscanManagerServiceClient()
	if err != nil {
		return nil, err
	}
	return &Provider{client: client, closer: closer}, nil
}

func (provider *Provider) ListNormalTransactions(
	ctx context.Context,
	chainID int64,
	wallet shared.Address,
	endBlock uint64,
) ([]research.WalletNormalTransaction, error) {
	response, err := provider.client.ListNormalTransactions(ctx, &etherscanmanagerapiclient.ListNormalTransactionsRequest{
		ChainId: chainID,
		Address: wallet.Hex(),
		BlockRange: &etherscanmanagerapiclient.NormalTransactionBlockRange{
			StartBlock: 0,
			EndBlock:   endBlock,
		},
		Page:     1,
		PageSize: normalTransactionPageSize,
		Sort:     etherscanmanagerapiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_DESC,
	})
	if err != nil {
		return nil, fmt.Errorf(
			"fetch etherscan-manager normal transactions chain_id=%d wallet=%s end_block=%d: %w",
			chainID,
			wallet.Hex(),
			endBlock,
			err,
		)
	}
	if response == nil {
		return nil, fmt.Errorf(
			"fetch etherscan-manager normal transactions chain_id=%d wallet=%s end_block=%d returned empty response",
			chainID,
			wallet.Hex(),
			endBlock,
		)
	}
	items := response.GetTransactions()
	if len(items) > int(normalTransactionPageSize) {
		return nil, fmt.Errorf(
			"fetch etherscan-manager normal transactions chain_id=%d wallet=%s returned %d transactions, limit is %d",
			chainID,
			wallet.Hex(),
			len(items),
			normalTransactionPageSize,
		)
	}
	result := make([]research.WalletNormalTransaction, 0, len(items))
	for index, item := range items {
		if item != nil && item.GetBlockNumber() > endBlock {
			return nil, fmt.Errorf(
				"normal transaction %d for wallet %s has block %d after requested end block %d",
				index,
				wallet.Hex(),
				item.GetBlockNumber(),
				endBlock,
			)
		}
		transaction, err := mapNormalTransaction(wallet, item)
		if err != nil {
			return nil, fmt.Errorf("map normal transaction %d for wallet %s: %w", index, wallet.Hex(), err)
		}
		if transaction.FromAddress != wallet && transaction.ToAddress != wallet {
			return nil, fmt.Errorf(
				"normal transaction %d does not involve requested wallet %s",
				index,
				wallet.Hex(),
			)
		}
		result = append(result, transaction)
	}
	return result, nil
}

func mapNormalTransaction(
	wallet shared.Address,
	item *etherscanmanagerapiclient.NormalTransaction,
) (research.WalletNormalTransaction, error) {
	if item == nil {
		return research.WalletNormalTransaction{}, fmt.Errorf("transaction is required")
	}
	transactionHash, err := shared.HexToHash(item.GetTransactionHash())
	if err != nil {
		return research.WalletNormalTransaction{}, fmt.Errorf("transaction_hash: %w", err)
	}
	fromAddress, err := shared.HexToAddress(item.GetFromAddress())
	if err != nil {
		return research.WalletNormalTransaction{}, fmt.Errorf("from_address: %w", err)
	}
	var toAddress shared.Address
	if text := strings.TrimSpace(item.GetToAddress()); text != "" {
		toAddress, err = shared.HexToAddress(text)
		if err != nil {
			return research.WalletNormalTransaction{}, fmt.Errorf("to_address: %w", err)
		}
	}
	value, err := parseUnsignedDecimal("value", item.GetValue())
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	gasPrice, err := parseUnsignedDecimal("gas_price", item.GetGasPrice())
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	receiptStatus, err := mapReceiptStatus(item.GetReceiptStatus())
	if err != nil {
		return research.WalletNormalTransaction{}, err
	}
	return research.WalletNormalTransaction{
		Wallet:           wallet,
		TransactionHash:  transactionHash,
		BlockNumber:      item.GetBlockNumber(),
		BlockTimestamp:   item.GetBlockTimestamp(),
		TransactionIndex: item.GetTransactionIndex(),
		Nonce:            item.GetNonce(),
		FromAddress:      fromAddress,
		ToAddress:        toAddress,
		Value:            value,
		Gas:              item.GetGas(),
		GasPrice:         gasPrice,
		GasUsed:          item.GetGasUsed(),
		Input:            item.GetInput(),
		MethodID:         item.GetMethodId(),
		FunctionName:     item.GetFunctionName(),
		ReceiptStatus:    receiptStatus,
		IsError:          item.GetIsError(),
	}, nil
}

func parseUnsignedDecimal(field, value string) (*big.Int, error) {
	result, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
	if !ok || result.Sign() < 0 {
		return nil, fmt.Errorf("%s must be a non-negative decimal integer", field)
	}
	return result, nil
}

func mapReceiptStatus(
	status etherscanmanagerapiclient.NormalTransactionReceiptStatus,
) (research.NormalTransactionReceiptStatus, error) {
	switch status {
	case etherscanmanagerapiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_UNSPECIFIED:
		return research.NormalTransactionReceiptStatusUnspecified, nil
	case etherscanmanagerapiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_FAILED:
		return research.NormalTransactionReceiptStatusFailed, nil
	case etherscanmanagerapiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_SUCCESS:
		return research.NormalTransactionReceiptStatusSuccess, nil
	default:
		return "", fmt.Errorf("receipt_status is invalid")
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
