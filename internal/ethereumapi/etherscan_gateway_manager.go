package ethereumapi

import (
	"context"
	stderrors "errors"
	"fmt"
	"net"
	"strings"
	"sync/atomic"

	"github.com/useryege/athena/pkg/apiclient/etherscangateway"
	utilethereumapi "github.com/useryege/athena/util/ethereumapi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const etherscanGatewayAuthorizationMetadataKey = "authorization"

type EtherscanGatewayManagerOpts struct {
	APIKeys      []string
	GatewayAddrs []string
	AuthToken    string
}

type EtherscanGatewayManager struct {
	apiKeys []string
	token   string
	next    atomic.Uint64
	clients []etherscanGatewayClient
	conns   []*grpc.ClientConn
}

type etherscanGatewayClient struct {
	addr   string
	client etherscangateway.EtherscanGatewayServiceClient
}

func NewEtherscanGatewayManager(opts EtherscanGatewayManagerOpts) (*EtherscanGatewayManager, error) {
	apiKeys := normalizeStringList(opts.APIKeys)
	if len(apiKeys) == 0 {
		return nil, fmt.Errorf("ATHENA_ETHEREUM_API_ETHERSCAN_API_KEYS must contain at least one API key")
	}
	gatewayAddrs := normalizeStringList(opts.GatewayAddrs)
	if len(gatewayAddrs) == 0 {
		return nil, fmt.Errorf("ATHENA_ETHEREUM_API_ETHERSCAN_GATEWAY_ADDRS must contain at least one gateway address")
	}
	for _, gatewayAddr := range gatewayAddrs {
		if _, _, err := net.SplitHostPort(gatewayAddr); err != nil {
			return nil, fmt.Errorf("gateway address %q must be host:port: %w", gatewayAddr, err)
		}
	}
	authToken := strings.TrimSpace(opts.AuthToken)
	if authToken == "" {
		return nil, fmt.Errorf("ATHENA_ETHERSCAN_GATEWAY_AUTH_TOKEN is required")
	}

	manager := &EtherscanGatewayManager{
		apiKeys: apiKeys,
		token:   authToken,
		clients: make([]etherscanGatewayClient, 0, len(gatewayAddrs)),
		conns:   make([]*grpc.ClientConn, 0, len(gatewayAddrs)),
	}
	for _, gatewayAddr := range gatewayAddrs {
		conn, err := grpc.NewClient(gatewayAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			_ = manager.Close()
			return nil, fmt.Errorf("create Etherscan Gateway gRPC client for %s: %w", gatewayAddr, err)
		}
		manager.conns = append(manager.conns, conn)
		manager.clients = append(manager.clients, etherscanGatewayClient{
			addr:   gatewayAddr,
			client: etherscangateway.NewEtherscanGatewayServiceClient(conn),
		})
	}
	return manager, nil
}

func (m *EtherscanGatewayManager) Close() error {
	if m == nil {
		return nil
	}
	var closeErr error
	for _, conn := range m.conns {
		if err := conn.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func (m *EtherscanGatewayManager) ListNormalTransactions(ctx context.Context, opts utilethereumapi.ListNormalTransactionsOptions) (*utilethereumapi.NormalTransactionsResponse, error) {
	apiKey, gatewayClient, err := m.nextGatewayClient()
	if err != nil {
		return nil, err
	}

	callCtx := metadata.AppendToOutgoingContext(ctx, etherscanGatewayAuthorizationMetadataKey, "Bearer "+m.token)
	response, err := gatewayClient.client.ListNormalTransactions(callCtx, &etherscangateway.ListNormalTransactionsRequest{
		ApiKey:     apiKey,
		ChainId:    opts.ChainID,
		Address:    opts.Address,
		BlockRange: gatewayBlockRange(opts),
		Page:       opts.Page,
		PageSize:   opts.PageSize,
		Sort:       gatewayNormalTransactionSort(opts.Sort),
	})
	if err != nil {
		return nil, gatewayManagerAPIError(gatewayClient.addr, err)
	}

	result := make([]utilethereumapi.NormalTransactionResult, 0, len(response.GetTransactions()))
	for _, transaction := range response.GetTransactions() {
		result = append(result, normalTransactionFromGateway(transaction))
	}
	return &utilethereumapi.NormalTransactionsResponse{
		Status:  "1",
		Message: "OK",
		Result:  result,
	}, nil
}

func (m *EtherscanGatewayManager) GetSourceCode(ctx context.Context, chainID int64, contractAddress string) (*utilethereumapi.SourceCodeResponse, error) {
	apiKey, gatewayClient, err := m.nextGatewayClient()
	if err != nil {
		return nil, err
	}

	callCtx := metadata.AppendToOutgoingContext(ctx, etherscanGatewayAuthorizationMetadataKey, "Bearer "+m.token)
	response, err := gatewayClient.client.GetSourceCode(callCtx, &etherscangateway.GetSourceCodeRequest{
		ApiKey:          apiKey,
		ChainId:         chainID,
		ContractAddress: contractAddress,
	})
	if err != nil {
		return nil, gatewayManagerAPIError(gatewayClient.addr, err)
	}

	result := make([]utilethereumapi.SourceCodeResult, 0, len(response.GetItems()))
	for _, item := range response.GetItems() {
		result = append(result, sourceCodeFromGateway(item))
	}
	return &utilethereumapi.SourceCodeResponse{
		Status:  "1",
		Message: "OK",
		Result:  result,
	}, nil
}

func (m *EtherscanGatewayManager) nextGatewayClient() (string, etherscanGatewayClient, error) {
	if m == nil || len(m.apiKeys) == 0 || len(m.clients) == 0 {
		return "", etherscanGatewayClient{}, &utilethereumapi.APIError{
			Type:    utilethereumapi.APIErrorTypeUpstream,
			Message: "etherscan gateway manager is not configured",
		}
	}
	index := m.next.Add(1) - 1
	return m.apiKeys[int(index%uint64(len(m.apiKeys)))], m.clients[int(index%uint64(len(m.clients)))], nil
}

func normalizeStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func gatewayBlockRange(opts utilethereumapi.ListNormalTransactionsOptions) *etherscangateway.NormalTransactionBlockRange {
	return &etherscangateway.NormalTransactionBlockRange{
		StartBlock: opts.StartBlock,
		EndBlock:   opts.EndBlock,
	}
}

func gatewayNormalTransactionSort(sort utilethereumapi.NormalTransactionSort) etherscangateway.NormalTransactionSort {
	switch sort {
	case utilethereumapi.NormalTransactionSortDESC:
		return etherscangateway.NormalTransactionSort_NORMAL_TRANSACTION_SORT_DESC
	default:
		return etherscangateway.NormalTransactionSort_NORMAL_TRANSACTION_SORT_ASC
	}
}

func normalTransactionFromGateway(item *etherscangateway.NormalTransaction) utilethereumapi.NormalTransactionResult {
	if item == nil {
		return utilethereumapi.NormalTransactionResult{}
	}
	return utilethereumapi.NormalTransactionResult{
		BlockNumber:       formatUint(item.GetBlockNumber()),
		BlockHash:         item.GetBlockHash(),
		TimeStamp:         formatUint(item.GetBlockTimestamp()),
		Hash:              item.GetTransactionHash(),
		Nonce:             formatUint(item.GetNonce()),
		TransactionIndex:  formatUint(item.GetTransactionIndex()),
		From:              item.GetFromAddress(),
		To:                item.GetToAddress(),
		Value:             item.GetValue(),
		Gas:               formatUint(item.GetGas()),
		GasPrice:          item.GetGasPrice(),
		Input:             item.GetInput(),
		MethodID:          item.GetMethodId(),
		FunctionName:      item.GetFunctionName(),
		ContractAddress:   item.GetContractAddress(),
		CumulativeGasUsed: formatUint(item.GetCumulativeGasUsed()),
		TxReceiptStatus:   gatewayReceiptStatus(item.GetReceiptStatus()),
		GasUsed:           formatUint(item.GetGasUsed()),
		Confirmations:     formatUint(item.GetConfirmations()),
		IsError:           gatewayBool(item.GetIsError()),
	}
}

func sourceCodeFromGateway(item *etherscangateway.SourceCode) utilethereumapi.SourceCodeResult {
	if item == nil {
		return utilethereumapi.SourceCodeResult{}
	}
	return utilethereumapi.SourceCodeResult{
		SourceCode:           item.GetSourceCode(),
		ABI:                  item.GetAbi(),
		ContractName:         item.GetContractName(),
		CompilerVersion:      item.GetCompilerVersion(),
		OptimizationUsed:     item.GetOptimizationUsed(),
		Runs:                 item.GetRuns(),
		ConstructorArguments: item.GetConstructorArguments(),
		EVMVersion:           item.GetEvmVersion(),
		Library:              item.GetLibrary(),
		LicenseType:          item.GetLicenseType(),
		Proxy:                item.GetProxy(),
		Implementation:       item.GetImplementation(),
		SwarmSource:          item.GetSwarmSource(),
	}
}

func gatewayManagerAPIError(gatewayAddr string, err error) error {
	if err == nil {
		return nil
	}
	if stderrors.Is(err, context.Canceled) || stderrors.Is(err, context.DeadlineExceeded) {
		return err
	}
	code := status.Code(err)
	message := fmt.Sprintf("etherscan gateway %s request failed: %v", gatewayAddr, err)
	switch code {
	case codes.ResourceExhausted:
		return &utilethereumapi.APIError{Type: utilethereumapi.APIErrorTypeRateLimit, Message: message}
	case codes.Unauthenticated, codes.PermissionDenied:
		return &utilethereumapi.APIError{Type: utilethereumapi.APIErrorTypeAuthentication, Message: message}
	case codes.InvalidArgument:
		return &utilethereumapi.APIError{Type: utilethereumapi.APIErrorTypeInvalidRequest, Message: message}
	case codes.DataLoss:
		return &utilethereumapi.APIError{Type: utilethereumapi.APIErrorTypeMalformed, Message: message}
	default:
		return &utilethereumapi.APIError{Type: utilethereumapi.APIErrorTypeUpstream, Message: message}
	}
}

func gatewayReceiptStatus(status etherscangateway.NormalTransactionReceiptStatus) string {
	switch status {
	case etherscangateway.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_FAILED:
		return "0"
	case etherscangateway.NormalTransactionReceiptStatus_NORMAL_TRANSACTION_RECEIPT_STATUS_SUCCESS:
		return "1"
	default:
		return ""
	}
}

func gatewayBool(value bool) string {
	if value {
		return "1"
	}
	return "0"
}

func formatUint(value uint64) string {
	return fmt.Sprintf("%d", value)
}
