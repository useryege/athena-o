package etherscangateway

import (
	"context"
	stderrors "errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/useryege/athena/pkg/apiclient/etherscangateway"
	utilethereumapi "github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultNormalTransactionEndBlock = uint64(999999999)
	defaultNormalTransactionPage     = int32(1)
	defaultNormalTransactionPageSize = int32(100)
	maxNormalTransactionPageSize     = int32(1000)
)

var apiKeyQueryParamPattern = regexp.MustCompile(`(?i)(apikey=)[^&\s]+`)

type ServiceOpts struct {
	EtherscanBaseURL string
	Timeout          time.Duration
}

type Service struct {
	etherscangateway.UnimplementedEtherscanGatewayServiceServer

	etherscanBaseURL string
	timeout          time.Duration

	startStopMu sync.Mutex
	started     bool
}

func NewService(opts ServiceOpts) *Service {
	baseURL := strings.TrimSpace(opts.EtherscanBaseURL)
	if baseURL == "" {
		baseURL = utilethereumapi.DefaultBaseURL
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = utilethereumapi.DefaultTimeout
	}
	return &Service{
		etherscanBaseURL: baseURL,
		timeout:          timeout,
	}
}

func (s *Service) Start(context.Context) error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = true
	return nil
}

func (s *Service) Stop() error {
	s.startStopMu.Lock()
	defer s.startStopMu.Unlock()
	s.started = false
	return nil
}

func (s *Service) GetEtherscanGatewayStatus(context.Context, *etherscangateway.GetEtherscanGatewayStatusRequest) (*etherscangateway.GetEtherscanGatewayStatusResponse, error) {
	s.startStopMu.Lock()
	started := s.started
	s.startStopMu.Unlock()

	statusText := "stopped"
	if started {
		statusText = "running"
	}
	return &etherscangateway.GetEtherscanGatewayStatusResponse{
		Started:          started,
		Status:           statusText,
		EtherscanBaseUrl: s.etherscanBaseURL,
	}, nil
}

func (s *Service) ListNormalTransactions(
	ctx context.Context,
	req *etherscangateway.ListNormalTransactionsRequest,
) (*etherscangateway.ListNormalTransactionsResponse, error) {
	query, apiKey, err := normalizeNormalTransactionRequest(req)
	if err != nil {
		return nil, err
	}

	client := s.newEtherscanClient(apiKey)
	response, err := client.ListNormalTransactions(ctx, query.options)
	if err != nil {
		return nil, grpcErrorFromEthereumAPI(err, apiKey)
	}

	transactions := make([]*etherscangateway.NormalTransaction, 0, len(response.Result))
	for index, item := range response.Result {
		transaction, err := normalTransactionFromEthereumAPI(item)
		if err != nil {
			return nil, status.Errorf(codes.DataLoss, "decode normal transaction %d: %v", index, err)
		}
		transactions = append(transactions, transaction)
	}
	return &etherscangateway.ListNormalTransactionsResponse{
		Transactions: transactions,
		Page:         query.page,
		PageSize:     query.pageSize,
	}, nil
}

func (s *Service) GetSourceCode(
	ctx context.Context,
	req *etherscangateway.GetSourceCodeRequest,
) (*etherscangateway.GetSourceCodeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	apiKey := strings.TrimSpace(req.GetApiKey())
	if apiKey == "" {
		return nil, status.Error(codes.InvalidArgument, "api_key is required")
	}
	if req.GetChainId() <= 0 {
		return nil, status.Error(codes.InvalidArgument, "chain_id must be positive")
	}
	contractAddress := strings.TrimSpace(req.GetContractAddress())
	if !ethcommon.IsHexAddress(contractAddress) {
		return nil, status.Error(codes.InvalidArgument, "contract_address must be a valid EVM address")
	}

	client := s.newEtherscanClient(apiKey)
	response, err := client.GetSourceCode(ctx, req.GetChainId(), ethcommon.HexToAddress(contractAddress).Hex())
	if err != nil {
		return nil, grpcErrorFromEthereumAPI(err, apiKey)
	}

	items := make([]*etherscangateway.SourceCode, 0, len(response.Result))
	for _, item := range response.Result {
		items = append(items, sourceCodeFromEthereumAPI(item))
	}
	return &etherscangateway.GetSourceCodeResponse{Items: items}, nil
}

func (s *Service) newEtherscanClient(apiKey string) utilethereumapi.EthereumAPI {
	return utilethereumapi.NewEthereumAPIWithConfig(utilethereumapi.Config{
		BaseURL: s.etherscanBaseURL,
		APIKey:  apiKey,
		Timeout: s.timeout,
	})
}

type normalizedNormalTransactionRequest struct {
	options  utilethereumapi.ListNormalTransactionsOptions
	page     int32
	pageSize int32
}

func normalizeNormalTransactionRequest(req *etherscangateway.ListNormalTransactionsRequest) (normalizedNormalTransactionRequest, string, error) {
	if req == nil {
		return normalizedNormalTransactionRequest{}, "", status.Error(codes.InvalidArgument, "request is required")
	}
	apiKey := strings.TrimSpace(req.GetApiKey())
	if apiKey == "" {
		return normalizedNormalTransactionRequest{}, "", status.Error(codes.InvalidArgument, "api_key is required")
	}
	if req.GetChainId() <= 0 {
		return normalizedNormalTransactionRequest{}, "", status.Error(codes.InvalidArgument, "chain_id must be positive")
	}
	addressText := strings.TrimSpace(req.GetAddress())
	if !ethcommon.IsHexAddress(addressText) {
		return normalizedNormalTransactionRequest{}, "", status.Error(codes.InvalidArgument, "address must be a valid EVM address")
	}

	startBlock := uint64(0)
	endBlock := defaultNormalTransactionEndBlock
	if blockRange := req.GetBlockRange(); blockRange != nil {
		startBlock = blockRange.GetStartBlock()
		endBlock = blockRange.GetEndBlock()
		if startBlock > endBlock {
			return normalizedNormalTransactionRequest{}, "", status.Error(codes.InvalidArgument, "start_block must not exceed end_block")
		}
	}
	if startBlock > math.MaxInt64 || endBlock > math.MaxInt64 {
		return normalizedNormalTransactionRequest{}, "", status.Error(codes.InvalidArgument, "block range exceeds the supported range")
	}

	page := req.GetPage()
	if page == 0 {
		page = defaultNormalTransactionPage
	}
	if page < 1 {
		return normalizedNormalTransactionRequest{}, "", status.Error(codes.InvalidArgument, "page must be positive")
	}
	pageSize := req.GetPageSize()
	if pageSize == 0 {
		pageSize = defaultNormalTransactionPageSize
	}
	if pageSize < 1 || pageSize > maxNormalTransactionPageSize {
		return normalizedNormalTransactionRequest{}, "", status.Errorf(
			codes.InvalidArgument,
			"page_size must be between 1 and %d",
			maxNormalTransactionPageSize,
		)
	}

	sortOrder := utilethereumapi.NormalTransactionSortASC
	switch req.GetSort() {
	case etherscangateway.NormalTransactionSort_NORMAL_TRANSACTION_SORT_UNSPECIFIED,
		etherscangateway.NormalTransactionSort_NORMAL_TRANSACTION_SORT_ASC:
	case etherscangateway.NormalTransactionSort_NORMAL_TRANSACTION_SORT_DESC:
		sortOrder = utilethereumapi.NormalTransactionSortDESC
	default:
		return normalizedNormalTransactionRequest{}, "", status.Error(codes.InvalidArgument, "sort is invalid")
	}

	return normalizedNormalTransactionRequest{
		options: utilethereumapi.ListNormalTransactionsOptions{
			ChainID:    req.GetChainId(),
			Address:    ethcommon.HexToAddress(addressText).Hex(),
			StartBlock: startBlock,
			EndBlock:   endBlock,
			Page:       page,
			PageSize:   pageSize,
			Sort:       sortOrder,
		},
		page:     page,
		pageSize: pageSize,
	}, apiKey, nil
}

func normalTransactionFromEthereumAPI(item utilethereumapi.NormalTransactionResult) (*etherscangateway.NormalTransaction, error) {
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

	return &etherscangateway.NormalTransaction{
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

func sourceCodeFromEthereumAPI(item utilethereumapi.SourceCodeResult) *etherscangateway.SourceCode {
	return &etherscangateway.SourceCode{
		SourceCode:           item.SourceCode,
		Abi:                  item.ABI,
		ContractName:         item.ContractName,
		CompilerVersion:      item.CompilerVersion,
		OptimizationUsed:     item.OptimizationUsed,
		Runs:                 item.Runs,
		ConstructorArguments: item.ConstructorArguments,
		EvmVersion:           item.EVMVersion,
		Library:              item.Library,
		LicenseType:          item.LicenseType,
		Proxy:                item.Proxy,
		Implementation:       item.Implementation,
		SwarmSource:          item.SwarmSource,
	}
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

func parseReceiptStatus(value string) (etherscangateway.NormalTransactionReceiptStatus, error) {
	switch strings.TrimSpace(value) {
	case "":
		return etherscangateway.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_UNSPECIFIED, nil
	case "0":
		return etherscangateway.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_FAILED, nil
	case "1":
		return etherscangateway.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_SUCCESS, nil
	default:
		return 0, fmt.Errorf("txreceipt_status must be empty, 0, or 1")
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

func grpcErrorFromEthereumAPI(err error, apiKey string) error {
	if err == nil {
		return nil
	}
	if contextError := contextErrorStatus(err); contextError != nil {
		return contextError
	}

	message := sanitizeEtherscanError(err, apiKey)
	var apiErr *utilethereumapi.APIError
	if stderrors.As(err, &apiErr) {
		switch apiErr.Type {
		case utilethereumapi.APIErrorTypeRateLimit:
			return status.Error(codes.ResourceExhausted, message)
		case utilethereumapi.APIErrorTypeAuthentication:
			return status.Error(codes.Unauthenticated, message)
		case utilethereumapi.APIErrorTypePlan:
			return status.Error(codes.PermissionDenied, message)
		case utilethereumapi.APIErrorTypeInvalidRequest:
			return status.Error(codes.InvalidArgument, message)
		case utilethereumapi.APIErrorTypeMalformed:
			return status.Error(codes.DataLoss, message)
		}
		return status.Error(codes.Unavailable, message)
	}

	lower := strings.ToLower(message)
	switch {
	case strings.Contains(lower, "rate limit"):
		return status.Error(codes.ResourceExhausted, message)
	case strings.Contains(lower, "invalid api key") || strings.Contains(lower, "invalid api-key"):
		return status.Error(codes.Unauthenticated, message)
	case strings.Contains(lower, "free api access") || strings.Contains(lower, "upgrade your api plan"):
		return status.Error(codes.PermissionDenied, message)
	case strings.Contains(lower, "invalid address") ||
		strings.Contains(lower, "unsupported chain") ||
		strings.Contains(lower, "invalid action") ||
		strings.Contains(lower, "missing"):
		return status.Error(codes.InvalidArgument, message)
	case strings.Contains(lower, "decode etherscan") || strings.Contains(lower, "malformed etherscan"):
		return status.Errorf(codes.DataLoss, "invalid etherscan response: %s", message)
	default:
		return status.Errorf(codes.Unavailable, "etherscan request failed: %s", message)
	}
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

func sanitizeEtherscanError(err error, apiKey string) string {
	if err == nil {
		return ""
	}
	message := apiKeyQueryParamPattern.ReplaceAllString(err.Error(), "${1}<redacted>")
	if apiKey = strings.TrimSpace(apiKey); apiKey != "" {
		message = strings.ReplaceAll(message, apiKey, "<redacted>")
	}
	return message
}
