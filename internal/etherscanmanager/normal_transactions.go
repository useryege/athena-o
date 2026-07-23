package etherscanmanager

import (
	"context"
	stderrors "errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/useryege/athena/internal/etherscanmanager/apiclient"
	"github.com/useryege/athena/util/etherscanapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultNormalTransactionEndBlock = uint64(999999999)
	defaultNormalTransactionPage     = int32(1)
	defaultNormalTransactionPageSize = int32(100)
	maxNormalTransactionPageSize     = int32(1000)
)

func (s *Service) ListNormalTransactions(
	ctx context.Context,
	req *apiclient.ListNormalTransactionsRequest,
) (*apiclient.ListNormalTransactionsResponse, error) {
	query, err := normalizeNormalTransactionQuery(req)
	if err != nil {
		return nil, err
	}
	response, err := s.manager.ListNormalTransactions(ctx, query)
	if err != nil {
		return nil, grpcErrorFromEtherscanAPI(err)
	}
	if response == nil {
		return nil, status.Error(codes.DataLoss, "normal transactions response is empty")
	}

	transactions := make([]*apiclient.NormalTransaction, 0, len(response.Result))
	for index, item := range response.Result {
		transaction, err := normalTransactionFromEtherscanAPI(item)
		if err != nil {
			return nil, status.Errorf(codes.DataLoss, "decode normal transaction %d: %v", index, err)
		}
		transactions = append(transactions, transaction)
	}
	return &apiclient.ListNormalTransactionsResponse{
		Transactions: transactions,
		Page:         query.Page,
		PageSize:     query.PageSize,
	}, nil
}

func normalizeNormalTransactionQuery(req *apiclient.ListNormalTransactionsRequest) (etherscanapi.ListNormalTransactionsOptions, error) {
	if req == nil {
		return etherscanapi.ListNormalTransactionsOptions{}, status.Error(codes.InvalidArgument, "request is required")
	}
	if req.GetChainId() <= 0 {
		return etherscanapi.ListNormalTransactionsOptions{}, status.Error(codes.InvalidArgument, "chain_id must be positive")
	}
	addressText := strings.TrimSpace(req.GetAddress())
	if !ethcommon.IsHexAddress(addressText) {
		return etherscanapi.ListNormalTransactionsOptions{}, status.Error(codes.InvalidArgument, "address must be a valid EVM address")
	}

	startBlock := uint64(0)
	endBlock := defaultNormalTransactionEndBlock
	if blockRange := req.GetBlockRange(); blockRange != nil {
		startBlock = blockRange.GetStartBlock()
		endBlock = blockRange.GetEndBlock()
		if startBlock > endBlock {
			return etherscanapi.ListNormalTransactionsOptions{}, status.Error(codes.InvalidArgument, "start_block must not exceed end_block")
		}
	}
	if startBlock > math.MaxInt64 || endBlock > math.MaxInt64 {
		return etherscanapi.ListNormalTransactionsOptions{}, status.Error(codes.InvalidArgument, "block range exceeds the supported range")
	}

	page := req.GetPage()
	if page == 0 {
		page = defaultNormalTransactionPage
	}
	if page < 1 {
		return etherscanapi.ListNormalTransactionsOptions{}, status.Error(codes.InvalidArgument, "page must be positive")
	}
	pageSize := req.GetPageSize()
	if pageSize == 0 {
		pageSize = defaultNormalTransactionPageSize
	}
	if pageSize < 1 || pageSize > maxNormalTransactionPageSize {
		return etherscanapi.ListNormalTransactionsOptions{}, status.Errorf(
			codes.InvalidArgument,
			"page_size must be between 1 and %d",
			maxNormalTransactionPageSize,
		)
	}

	sortOrder := etherscanapi.NormalTransactionSortASC
	switch req.GetSort() {
	case apiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_UNSPECIFIED,
		apiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_ASC:
	case apiclient.NormalTransactionSort_NORMAL_TRANSACTION_SORT_DESC:
		sortOrder = etherscanapi.NormalTransactionSortDESC
	default:
		return etherscanapi.ListNormalTransactionsOptions{}, status.Error(codes.InvalidArgument, "sort is invalid")
	}

	return etherscanapi.ListNormalTransactionsOptions{
		ChainID:    req.GetChainId(),
		Address:    ethcommon.HexToAddress(addressText).Hex(),
		StartBlock: startBlock,
		EndBlock:   endBlock,
		Page:       page,
		PageSize:   pageSize,
		Sort:       sortOrder,
	}, nil
}

func normalTransactionFromEtherscanAPI(item etherscanapi.NormalTransactionResult) (*apiclient.NormalTransaction, error) {
	blockNumber, err := parseUint64Field("blockNumber", item.BlockNumber)
	if err != nil {
		return nil, err
	}
	blockHash, err := parseHashField("blockHash", item.BlockHash)
	if err != nil {
		return nil, err
	}
	blockTimestamp, err := parseUint64Field("timeStamp", item.TimeStamp)
	if err != nil {
		return nil, err
	}
	txHash, err := parseHashField("hash", item.Hash)
	if err != nil {
		return nil, err
	}
	nonce, err := parseUint64Field("nonce", item.Nonce)
	if err != nil {
		return nil, err
	}
	transactionIndex, err := parseUint64Field("transactionIndex", item.TransactionIndex)
	if err != nil {
		return nil, err
	}
	fromAddress, err := parseAddressField("from", item.From)
	if err != nil {
		return nil, err
	}
	toAddress, err := parseOptionalAddressField("to", item.To)
	if err != nil {
		return nil, err
	}
	value, err := parseDecimalField("value", item.Value)
	if err != nil {
		return nil, err
	}
	gas, err := parseUint64Field("gas", item.Gas)
	if err != nil {
		return nil, err
	}
	gasPrice, err := parseDecimalField("gasPrice", item.GasPrice)
	if err != nil {
		return nil, err
	}
	methodID, err := parseOptionalFixedBytesField("methodId", item.MethodID, 4)
	if err != nil {
		return nil, err
	}
	contractAddress, err := parseOptionalAddressField("contractAddress", item.ContractAddress)
	if err != nil {
		return nil, err
	}
	cumulativeGasUsed, err := parseUint64Field("cumulativeGasUsed", item.CumulativeGasUsed)
	if err != nil {
		return nil, err
	}
	receiptStatus, err := parseReceiptStatus(item.TxReceiptStatus)
	if err != nil {
		return nil, err
	}
	gasUsed, err := parseUint64Field("gasUsed", item.GasUsed)
	if err != nil {
		return nil, err
	}
	confirmations, err := parseUint64Field("confirmations", item.Confirmations)
	if err != nil {
		return nil, err
	}
	isError, err := parseZeroOneBool("isError", item.IsError)
	if err != nil {
		return nil, err
	}

	toAddressText := ""
	if toAddress != nil {
		toAddressText = toAddress.Hex()
	}
	contractAddressText := ""
	if contractAddress != nil {
		contractAddressText = contractAddress.Hex()
	}
	methodIDText := ""
	if len(methodID) > 0 {
		methodIDText = hexutil.Encode(methodID)
	}

	return &apiclient.NormalTransaction{
		BlockNumber:       blockNumber,
		BlockHash:         blockHash.Hex(),
		BlockTimestamp:    blockTimestamp,
		TransactionHash:   txHash.Hex(),
		Nonce:             nonce,
		TransactionIndex:  transactionIndex,
		FromAddress:       fromAddress.Hex(),
		ToAddress:         toAddressText,
		Value:             value,
		Gas:               gas,
		GasPrice:          gasPrice,
		Input:             item.Input,
		MethodId:          methodIDText,
		FunctionName:      item.FunctionName,
		ContractAddress:   contractAddressText,
		CumulativeGasUsed: cumulativeGasUsed,
		ReceiptStatus:     receiptStatus,
		GasUsed:           gasUsed,
		Confirmations:     confirmations,
		IsError:           isError,
	}, nil
}

func parseUint64Field(name string, value string) (uint64, error) {
	parsed, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s must be a uint64 decimal integer", name)
	}
	return parsed, nil
}

func parseDecimalField(name string, value string) (string, error) {
	integer, ok := new(big.Int).SetString(strings.TrimSpace(value), 10)
	if !ok || integer.Sign() < 0 {
		return "", fmt.Errorf("%s must be a nonnegative decimal integer", name)
	}
	return integer.String(), nil
}

func parseHashField(name string, value string) (ethcommon.Hash, error) {
	decoded, err := hexutil.Decode(strings.TrimSpace(value))
	if err != nil || len(decoded) != ethcommon.HashLength {
		return ethcommon.Hash{}, fmt.Errorf("%s must be a 32-byte hex hash", name)
	}
	return ethcommon.BytesToHash(decoded), nil
}

func parseAddressField(name string, value string) (ethcommon.Address, error) {
	if !ethcommon.IsHexAddress(strings.TrimSpace(value)) {
		return ethcommon.Address{}, fmt.Errorf("%s must be a valid EVM address", name)
	}
	return ethcommon.HexToAddress(value), nil
}

func parseOptionalAddressField(name string, value string) (*ethcommon.Address, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "0x" {
		return nil, nil
	}
	address, err := parseAddressField(name, value)
	if err != nil {
		return nil, err
	}
	return &address, nil
}

func parseOptionalFixedBytesField(name string, value string, length int) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "0x" {
		return nil, nil
	}
	decoded, err := hexutil.Decode(value)
	if err != nil || len(decoded) != length {
		return nil, fmt.Errorf("%s must be %d bytes of hex data", name, length)
	}
	return decoded, nil
}

func parseReceiptStatus(value string) (apiclient.NormalTransactionReceiptStatus, error) {
	switch strings.TrimSpace(value) {
	case "":
		return apiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_UNSPECIFIED, nil
	case "0":
		return apiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_FAILED, nil
	case "1":
		return apiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_SUCCESS, nil
	default:
		return apiclient.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_UNSPECIFIED, fmt.Errorf("txreceipt_status must be empty, 0, or 1")
	}
}

func parseZeroOneBool(name string, value string) (bool, error) {
	switch strings.TrimSpace(value) {
	case "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, fmt.Errorf("%s must be 0 or 1", name)
	}
}

func grpcErrorFromEtherscanAPI(err error) error {
	if err == nil {
		return nil
	}
	if contextError := contextErrorStatus(err); contextError != nil {
		return contextError
	}
	var apiErr *etherscanapi.APIError
	if stderrors.As(err, &apiErr) {
		switch apiErr.Type {
		case etherscanapi.APIErrorTypeRateLimit:
			return status.Error(codes.ResourceExhausted, apiErr.Error())
		case etherscanapi.APIErrorTypeAuthentication, etherscanapi.APIErrorTypePlan:
			return status.Error(codes.FailedPrecondition, apiErr.Error())
		case etherscanapi.APIErrorTypeInvalidRequest:
			return status.Error(codes.InvalidArgument, apiErr.Error())
		case etherscanapi.APIErrorTypeMalformed:
			return status.Error(codes.DataLoss, apiErr.Error())
		}
		return status.Error(codes.Unavailable, apiErr.Error())
	}
	if strings.Contains(strings.ToLower(err.Error()), "decode etherscan") {
		return status.Errorf(codes.DataLoss, "invalid etherscan response: %v", err)
	}
	return status.Errorf(codes.Unavailable, "etherscan request failed: %v", err)
}

func contextErrorStatus(err error) error {
	switch {
	case stderrors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, context.Canceled.Error())
	case stderrors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, context.DeadlineExceeded.Error())
	default:
		return nil
	}
}
